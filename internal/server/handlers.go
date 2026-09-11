package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/program"
	"github.com/kbremont/531-tracker/internal/views"
)

const recentLimit = 8

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state, err := s.loadState(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// No training maxes means the app has never been set up.
	if len(state.TMOrder) < len(state.Lifts) {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	vm := views.DashboardVM{
		Unit:        state.Raw.Unit,
		ProgramName: state.Config.Name,
	}
	for _, tm := range state.TMOrder {
		vm.TMs = append(vm.TMs, views.LiftTM{Slug: tm.Slug, Name: tm.Name, Value: tm.Value})
	}

	switch {
	case state.InProgress != nil:
		var lift db.Lift
		for _, l := range state.Lifts {
			if l.Slug == state.InProgress.LiftSlug {
				lift = l
				break
			}
		}
		vm.Next = &views.NextUp{
			LiftName:     lift.Name,
			LiftSlug:     lift.Slug,
			CycleLabel:   state.InProgress.CycleLabel,
			Phase:        state.InProgress.Phase,
			TrainingMax:  state.InProgress.TrainingMax,
			InProgress:   true,
			WorkoutID:    state.InProgress.ID,
			BlockNumber:  int(state.Block.Number),
			WeekOfBlock:  weekOfBlock(state.Config, program.Position{CycleIndex: int(state.InProgress.CycleIndex), Week: int(state.InProgress.Week)}),
			WeeksInBlock: state.Config.WeeksPerBlock(),
		}
	case state.HasNext:
		lift := state.LiftByDay[state.Next.Day]
		vm.Next = &views.NextUp{
			LiftName:     lift.Name,
			LiftSlug:     lift.Slug,
			CycleLabel:   state.Config.Describe(state.Next),
			TrainingMax:  state.TMs[lift.Slug],
			Day:          state.Next.Day,
			BlockNumber:  int(state.Block.Number),
			WeekOfBlock:  weekOfBlock(state.Config, state.Next),
			WeeksInBlock: state.Config.WeeksPerBlock(),
		}
		if cycle, err := state.Config.CycleAt(state.Next.CycleIndex); err == nil {
			vm.Next.Phase = cycle.Phase
		}
	}

	s.addCycleRollover(&vm, state)

	recent, err := s.recentHistory(ctx, recentLimit)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	vm.Recent = recent

	s.render(w, r, views.Dashboard(vm))
}

// addCycleRollover offers a training max bump when a cycle has just closed
// out. It is only ever an offer; nothing here writes a training max.
func (s *Server) addCycleRollover(vm *views.DashboardVM, state *blockState) {
	if state.InProgress != nil || !state.HasNext {
		return
	}
	// A cycle boundary is the first day of the first week of a cycle, with
	// something behind it.
	atBoundary := state.Next.Week == 1 && state.Next.Day == 1 &&
		(state.Next.CycleIndex > 0 || state.Block.Number > 1)
	if !atBoundary {
		return
	}

	key := fmt.Sprintf("b%d:c%d", state.Block.ID, state.Next.CycleIndex)
	if state.Raw.DismissedCycle == key {
		return
	}

	finishedIdx := state.Next.CycleIndex - 1
	if finishedIdx < 0 {
		finishedIdx = len(state.Config.Cycles) - 1
	}
	finished, err := state.Config.CycleAt(finishedIdx)
	if err != nil {
		return
	}

	vm.CycleDone = true
	vm.CycleDoneLabel = finished.Label
	vm.CycleDoneKey = key
	for _, tm := range state.TMOrder {
		vm.Suggestions = append(vm.Suggestions, views.TMSuggestion{
			Slug:    tm.Slug,
			Name:    tm.Name,
			Current: tm.Value,
			Next:    tm.Value + program.TMIncrementFor(state.Config, tm.IsUpper != 0),
		})
	}
}

func (s *Server) handleDismissSuggestions(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.q.DismissCycle(r.Context(), r.PostFormValue("key")); err != nil {
		s.fail(w, r, fmt.Errorf("dismiss cycle prompt: %w", err))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list lifts: %w", err))
		return
	}

	vm := views.SetupVM{Unit: raw.Unit}
	for _, l := range lifts {
		item := views.LiftTM{Slug: l.Slug, Name: l.Name}
		if tm, err := s.q.GetCurrentTrainingMax(ctx, l.Slug); err == nil {
			item.Value = tm.Value
		}
		vm.Lifts = append(vm.Lifts, item)
	}
	s.render(w, r, views.Setup(vm))
}

func (s *Server) handleSetupSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list lifts: %w", err))
		return
	}

	values := make(map[string]float64, len(lifts))
	for _, l := range lifts {
		v, err := strconv.ParseFloat(r.PostFormValue("tm_"+l.Slug), 64)
		if err != nil || v <= 0 {
			s.renderSetupError(w, r, lifts, fmt.Sprintf("Enter a positive training max for %s.", l.Name))
			return
		}
		values[l.Slug] = v
	}

	err = s.tx(ctx, func(q *db.Queries) error {
		for slug, v := range values {
			if _, err := q.SetTrainingMax(ctx, db.SetTrainingMaxParams{
				LiftSlug: slug,
				Value:    v,
				Note:     "initial setup",
			}); err != nil {
				return fmt.Errorf("set training max for %s: %w", slug, err)
			}
		}
		return nil
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) renderSetupError(w http.ResponseWriter, r *http.Request, lifts []db.Lift, msg string) {
	raw, _, _ := s.loadSettings(r.Context())
	vm := views.SetupVM{Unit: raw.Unit, Error: msg}
	for _, l := range lifts {
		v, _ := strconv.ParseFloat(r.PostFormValue("tm_"+l.Slug), 64)
		vm.Lifts = append(vm.Lifts, views.LiftTM{Slug: l.Slug, Name: l.Name, Value: v})
	}
	w.WriteHeader(http.StatusBadRequest)
	s.render(w, r, views.Setup(vm))
}

// handleSetTrainingMax appends to the training max log. It never updates in
// place, so past sessions keep the weights they were performed at.
func (s *Server) handleSetTrainingMax(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	slug := r.PostFormValue("lift")
	value, err := strconv.ParseFloat(r.PostFormValue("value"), 64)
	if err != nil || value <= 0 {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	if _, err := s.q.SetTrainingMax(r.Context(), db.SetTrainingMaxParams{
		LiftSlug: slug,
		Value:    value,
		Note:     r.PostFormValue("note"),
	}); err != nil {
		s.fail(w, r, fmt.Errorf("set training max for %s: %w", slug, err))
		return
	}

	if r.PostFormValue("note") == "cycle rollover" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	tms, err := s.q.ListCurrentTrainingMaxes(ctx)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list training maxes: %w", err))
		return
	}

	vm := views.SettingsVM{
		Unit:        raw.Unit,
		Rounding:    raw.Rounding,
		BarWeight:   raw.BarWeight,
		Plates:      formatPlates(settings.Plates),
		Warmups:     settings.Warmups,
		RestSeconds: int(raw.RestSeconds),
		Saved:       r.URL.Query().Get("saved") == "1",
	}
	for _, tm := range tms {
		vm.TMs = append(vm.TMs, views.LiftTM{Slug: tm.Slug, Name: tm.Name, Value: tm.Value})
	}
	s.render(w, r, views.Settings(vm))
}

func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	plates, err := parsePlates(r.PostFormValue("plates"))
	if err != nil {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	encoded, err := json.Marshal(plates)
	if err != nil {
		s.fail(w, r, fmt.Errorf("encode plates: %w", err))
		return
	}

	warmups := int64(0)
	if r.PostFormValue("warmups") != "" {
		warmups = 1
	}

	params := db.UpdateSettingsParams{
		Unit:        raw.Unit, // pounds only for now; the column exists for later
		Rounding:    positiveOr(r.PostFormValue("rounding"), raw.Rounding),
		BarWeight:   positiveOr(r.PostFormValue("bar_weight"), raw.BarWeight),
		Plates:      string(encoded),
		Warmups:     warmups,
		RestSeconds: int64(positiveOr(r.PostFormValue("rest_seconds"), float64(raw.RestSeconds))),
	}
	if err := s.q.UpdateSettings(ctx, params); err != nil {
		s.fail(w, r, fmt.Errorf("update settings: %w", err))
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	items, err := s.recentHistory(ctx, 60)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list lifts: %w", err))
		return
	}

	vm := views.HistoryVM{Items: items}
	for _, l := range lifts {
		vm.Lifts = append(vm.Lifts, views.LiftTM{Slug: l.Slug, Name: l.Name})
	}
	s.render(w, r, views.History(vm))
}

// recentHistory summarises completed sessions. The per-workout set lookup is
// a query each, which is the right trade against a more complex join when the
// database is a local file and the list is at most a screen or two long.
func (s *Server) recentHistory(ctx context.Context, limit int64) ([]views.HistoryItem, error) {
	rows, err := s.q.ListRecentWorkouts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent workouts: %w", err)
	}

	items := make([]views.HistoryItem, 0, len(rows))
	for _, row := range rows {
		sets, err := s.q.ListSets(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("list sets for workout %d: %w", row.ID, err)
		}
		items = append(items, views.HistoryItem{
			WorkoutID:  row.ID,
			LiftName:   row.LiftName,
			LiftSlug:   row.LiftSlug,
			CycleLabel: row.CycleLabel,
			Date:       formatDate(row.CompletedAt),
			Summary:    topSetSummary(sets),
		})
	}
	return items, nil
}

// topSetSummary picks the set worth remembering: the PR set when there is
// one, otherwise the heaviest main set that was completed.
func topSetSummary(sets []db.Set) string {
	var best db.Set
	var found bool
	for _, set := range sets {
		if !set.CompletedAt.Valid || set.Kind != string(program.KindMain) {
			continue
		}
		switch {
		case !found:
			best, found = set, true
		case set.IsAmrap != 0 && best.IsAmrap == 0:
			best = set
		case set.IsAmrap == best.IsAmrap && set.Weight > best.Weight:
			best = set
		}
	}
	if !found {
		return ""
	}
	return fmt.Sprintf("%s x %d", views.FmtWeight(best.Weight), best.ActualReps.Int64)
}

func positiveOr(raw string, fallback float64) float64 {
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// formatDate renders a SQLite UTC timestamp in the server's local zone, which
// is what the TZ environment variable in the compose file is for.
func formatDate(ts sql.NullString) string {
	if !ts.Valid {
		return ""
	}
	t, err := time.Parse("2006-01-02 15:04:05", ts.String)
	if err != nil {
		return ts.String
	}
	return t.UTC().In(time.Local).Format("Jan 2")
}

// formatTimestamp renders a non-null SQLite UTC timestamp in local time.
func formatTimestamp(ts string) string {
	return formatDate(sql.NullString{String: ts, Valid: true})
}
