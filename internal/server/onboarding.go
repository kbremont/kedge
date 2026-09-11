package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/program"
	"github.com/kbremont/531-tracker/internal/views"
)

// Onboarding lets someone who is already mid-programme start using the app
// without keying in their whole training history. They name the cycle they are
// on and tick off the sessions of that cycle they have already done; earlier
// cycles are excluded from the block rather than fabricated.

func doneField(round int, slug string) string { return fmt.Sprintf("done_r%d_%s", round, slug) }
func repsField(round int, slug string) string { return fmt.Sprintf("reps_r%d_%s", round, slug) }

func bbbModeField(slug string) string  { return "bbb_mode_" + slug }
func bbbValueField(slug string) string { return "bbb_value_" + slug }

func (s *Server) handleSetupPosition(w http.ResponseWriter, r *http.Request) {
	_, cfg, err := s.activeProgram(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}

	vm := views.SetupPositionVM{}
	for i, c := range cfg.Cycles {
		vm.Cycles = append(vm.Cycles, views.SetupCycleOption{
			Index:  i,
			Label:  c.Label,
			Detail: cycleDetail(c),
		})
	}
	s.render(w, r, views.SetupPosition(vm))
}

func (s *Server) handleSetupPositionSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	choice := r.PostFormValue("cycle")
	if choice == "" || choice == "fresh" {
		// Nothing to record; the block already starts at cycle zero.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	_, cfg, err := s.activeProgram(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	idx, err := strconv.Atoi(choice)
	if err != nil {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	if _, err := cfg.CycleAt(idx); err != nil {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/setup/history?cycle=%d", idx), http.StatusSeeOther)
}

func (s *Server) handleSetupHistory(w http.ResponseWriter, r *http.Request) {
	vm, _, err := s.buildSetupHistory(r.Context(), r.URL.Query().Get("cycle"))
	if errors.Is(err, errNeedsTrainingMaxes) {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	if errors.Is(err, errBadCycle) {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, views.SetupHistory(*vm))
}

func (s *Server) handleSetupHistorySave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	vm, ctxData, err := s.buildSetupHistory(ctx, r.PostFormValue("cycle"))
	if errors.Is(err, errNeedsTrainingMaxes) {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	if errors.Is(err, errBadCycle) {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}

	prog, cfg, err := s.activeProgram(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	block, err := s.currentBlock(ctx, prog.ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	err = s.tx(ctx, func(q *db.Queries) error {
		if err := q.SetBlockStartCycle(ctx, db.SetBlockStartCycleParams{
			StartCycleIndex: int64(vm.CycleIndex),
			ID:              block.ID,
		}); err != nil {
			return fmt.Errorf("set block start cycle: %w", err)
		}

		for _, rd := range vm.Rounds {
			for _, day := range rd.Days {
				if r.PostFormValue(day.Field) == "" {
					continue
				}
				prReps := 0
				if vm.AsksReps {
					if n, err := strconv.Atoi(r.PostFormValue(day.RepsField)); err == nil && n > 0 && n <= 500 {
						prReps = n
					}
				}
				if err := s.recordPastSession(ctx, q, cfg, ctxData, block.ID, vm.CycleIndex, rd.Round, day.Slug, prReps); err != nil {
					return err
				}
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

// recordPastSession writes a session the user says they already performed. The
// sets come from the same pure generator the live flow uses, so the weights
// match exactly what the app would have prescribed at this training max.
func (s *Server) recordPastSession(
	ctx context.Context,
	q *db.Queries,
	cfg program.Config,
	data *setupData,
	blockID int64,
	cycleIdx, round int,
	slug string,
	prReps int,
) error {
	// Re-running onboarding must not collide with what is already recorded.
	if _, err := q.GetWorkoutAt(ctx, db.GetWorkoutAtParams{
		BlockID:    blockID,
		CycleIndex: int64(cycleIdx),
		Round:      int64(round),
		LiftSlug:   slug,
	}); err == nil {
		return nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("look up existing session: %w", err)
	}

	tm := data.TMs[slug]
	if tm <= 0 {
		return fmt.Errorf("no training max for %s", slug)
	}
	cycle, err := cfg.CycleAt(cycleIdx)
	if err != nil {
		return err
	}
	generated, err := program.GenerateWorkout(cfg.WithBBBPct(data.BBBPct[slug]), cycleIdx, round, tm, data.Settings)
	if err != nil {
		return err
	}

	pos := program.Position{CycleIndex: cycleIdx, Round: round}
	workout, err := q.InsertWorkout(ctx, db.InsertWorkoutParams{
		BlockID:     blockID,
		LiftSlug:    slug,
		CycleIndex:  int64(cycleIdx),
		CycleLabel:  cfg.Describe(pos),
		Phase:       cycle.Phase,
		Round:       int64(round),
		TrainingMax: tm,
	})
	if err != nil {
		return fmt.Errorf("insert past session: %w", err)
	}

	for _, set := range generated.Sets {
		reps := set.TargetReps
		if set.IsAMRAP && prReps > 0 {
			reps = prReps
		}
		amrap := int64(0)
		if set.IsAMRAP {
			amrap = 1
		}
		if err := q.InsertCompletedSet(ctx, db.InsertCompletedSetParams{
			WorkoutID:  workout.ID,
			Idx:        int64(set.Index),
			Kind:       string(set.Kind),
			Weight:     set.Weight,
			Pct:        set.Pct,
			TargetReps: int64(set.TargetReps),
			IsAmrap:    amrap,
			ActualReps: sql.NullInt64{Int64: int64(reps), Valid: true},
		}); err != nil {
			return fmt.Errorf("insert past set %d: %w", set.Index, err)
		}
	}

	if err := q.CompleteWorkout(ctx, workout.ID); err != nil {
		return fmt.Errorf("complete past session: %w", err)
	}
	return nil
}

// setupData is the shared lookup needed to both render and save the checklist.
type setupData struct {
	Settings program.Settings
	TMs      map[string]float64
	BBBPct   map[string]float64
}

var (
	errBadCycle           = errors.New("unknown cycle")
	errNeedsTrainingMaxes = errors.New("training maxes not set")
)

// buildSetupHistory assembles the checklist for a cycle. Each row carries the
// weight of its top main set, which is what makes a session recognisable.
func (s *Server) buildSetupHistory(ctx context.Context, rawCycle string) (*views.SetupHistoryVM, *setupData, error) {
	idx, err := strconv.Atoi(rawCycle)
	if err != nil {
		return nil, nil, errBadCycle
	}

	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, nil, err
	}
	_, cfg, err := s.activeProgram(ctx)
	if err != nil {
		return nil, nil, err
	}
	cycle, err := cfg.CycleAt(idx)
	if err != nil {
		return nil, nil, errBadCycle
	}

	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list lifts: %w", err)
	}
	tmRows, err := s.q.ListCurrentTrainingMaxes(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list training maxes: %w", err)
	}
	if len(tmRows) < len(lifts) {
		return nil, nil, errNeedsTrainingMaxes
	}
	tms := make(map[string]float64, len(tmRows))
	for _, row := range tmRows {
		tms[row.Slug] = row.Value
	}
	bbbPct := make(map[string]float64, len(lifts))
	for _, lift := range lifts {
		bbbPct[lift.Slug] = bbbPctFor(lift, tms[lift.Slug])
	}

	vm := &views.SetupHistoryVM{
		CycleIndex: idx,
		CycleLabel: cycle.Label,
		Unit:       raw.Unit,
		AsksReps:   cycle.Main == program.Scheme531PR,
	}
	for round := 1; round <= cycle.Rounds; round++ {
		rd := views.SetupRoundVM{Round: round}
		for _, lift := range lifts {
			generated, err := program.GenerateWorkout(cfg.WithBBBPct(bbbPct[lift.Slug]), idx, round, tms[lift.Slug], settings)
			if err != nil {
				return nil, nil, err
			}
			top, prTarget := topMainSet(generated)
			rd.Days = append(rd.Days, views.SetupDayVM{
				Slug:      lift.Slug,
				Name:      lift.Name,
				TopWeight: top,
				PRTarget:  prTarget,
				Field:     doneField(round, lift.Slug),
				RepsField: repsField(round, lift.Slug),
			})
		}
		vm.Rounds = append(vm.Rounds, rd)
	}
	return vm, &setupData{Settings: settings, TMs: tms, BBBPct: bbbPct}, nil
}

// topMainSet returns the heaviest main set of a session, and the rep target of
// its PR set if it has one.
func topMainSet(wk program.Workout) (weight float64, prTarget int) {
	for _, set := range wk.Sets {
		if set.Kind != program.KindMain {
			continue
		}
		if set.Weight > weight {
			weight = set.Weight
		}
		if set.IsAMRAP {
			prTarget = set.TargetReps
		}
	}
	return weight, prTarget
}

// cycleDetail summarises a cycle for the position picker, built from the config
// so a renamed or reordered cycle needs no change here.
func cycleDetail(c program.CycleSpec) string {
	var main string
	switch c.Main {
	case program.SchemeFivesPro:
		main = "5's PRO"
	case program.Scheme531PR:
		main = "5/3/1 with a PR set"
	case program.SchemeDeload:
		main = "deload"
	default:
		main = string(c.Main)
	}

	var supp string
	switch c.Supp.Kind {
	case program.SuppBBB:
		supp = fmt.Sprintf(" + BBB %dx%d @ %s", c.Supp.Sets, c.Supp.Reps, views.FmtPct(c.Supp.Pct))
	case program.SuppFSL:
		supp = fmt.Sprintf(" + FSL %dx%d", c.Supp.Sets, c.Supp.Reps)
	}

	if c.Rounds == 1 {
		return main + supp
	}
	return fmt.Sprintf("%d rounds, %s%s", c.Rounds, main, supp)
}
