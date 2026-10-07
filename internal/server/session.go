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

// handleStartWorkout materialises the next scheduled session. Sets are written
// with absolute weights, computed from the training max in force right now, so
// a later change to that training max leaves this session untouched.
func (s *Server) handleStartWorkout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	state, err := s.loadState(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if state.InProgress != nil {
		http.Redirect(w, r, fmt.Sprintf("/workout/%d", state.InProgress.ID), http.StatusSeeOther)
		return
	}
	if !state.HasNext {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	// Training waits on a pending training max decision; the dashboard shows it.
	if _, _, held, err := s.trainingHold(ctx, state); err != nil {
		s.fail(w, r, err)
		return
	} else if held {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	pos := state.Next
	lift, ok := state.LiftByDay[pos.Day]
	if !ok {
		s.fail(w, r, fmt.Errorf("no lift configured for day %d", pos.Day))
		return
	}
	tm, ok := state.TMs[lift.Slug]
	if !ok || tm <= 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}

	cycle, err := state.Config.CycleAt(pos.CycleIndex)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	cfg := state.Config.WithBBBPct(bbbPctFor(lift, tm))
	generated, err := program.GenerateWorkout(cfg, pos.CycleIndex, pos.Round, tm, state.Settings)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	var workoutID int64
	err = s.tx(ctx, func(q *db.Queries) error {
		row, err := q.InsertWorkout(ctx, db.InsertWorkoutParams{
			BlockID:     state.Block.ID,
			LiftSlug:    lift.Slug,
			CycleIndex:  int64(pos.CycleIndex),
			CycleLabel:  state.Config.Describe(pos),
			Phase:       cycle.Phase,
			Round:       int64(pos.Round),
			TrainingMax: tm,
		})
		if err != nil {
			return fmt.Errorf("insert workout: %w", err)
		}
		workoutID = row.ID

		for _, set := range generated.Sets {
			amrap := int64(0)
			if set.IsAMRAP {
				amrap = 1
			}
			if err := q.InsertSet(ctx, db.InsertSetParams{
				WorkoutID:  row.ID,
				Idx:        int64(set.Index),
				Kind:       string(set.Kind),
				Weight:     set.Weight,
				Pct:        set.Pct,
				TargetReps: int64(set.TargetReps),
				MaxReps:    int64(set.MaxReps),
				IsAmrap:    amrap,
			}); err != nil {
				return fmt.Errorf("insert set %d: %w", set.Index, err)
			}
		}
		return nil
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/workout/%d", workoutID), http.StatusSeeOther)
}

func (s *Server) handleWorkout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workout, sets, err := s.workoutWithSets(ctx, r.PathValue("id"))
	if err != nil {
		s.notFoundOrFail(w, r, err)
		return
	}

	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	_, cfg, err := s.activeProgram(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	lift, err := s.q.GetLift(ctx, workout.LiftSlug)
	if err != nil {
		s.fail(w, r, fmt.Errorf("load lift %s: %w", workout.LiftSlug, err))
		return
	}

	suppKind := program.SuppNone
	if cycle, err := cfg.CycleAt(int(workout.CycleIndex)); err == nil {
		suppKind = cycle.Supp.Kind
	}

	vm := views.WorkoutVM{
		ID:          workout.ID,
		LiftName:    lift.Name,
		CycleLabel:  workout.CycleLabel,
		Phase:       workout.Phase,
		Unit:        raw.Unit,
		TrainingMax: workout.TrainingMax,
		CurrentIdx:  -1,
		RestSeconds: restSecondsFor(raw, program.KindMain),
		ShowRest:    r.URL.Query().Get("rest") == "1",
		Completed:   workout.CompletedAt.Valid,
	}

	for _, set := range sets {
		pairs, unloadable := program.PlatesFor(set.Weight, settings)
		vm.Sets = append(vm.Sets, views.SetVM{
			Idx:        int(set.Idx),
			Tag:        views.SetTag(program.SetKind(set.Kind), suppKind),
			Weight:     set.Weight,
			Pct:        set.Pct,
			TargetReps: int(set.TargetReps),
			MaxReps:    int(set.MaxReps),
			IsAMRAP:    set.IsAmrap != 0,
			Plates:     views.FmtPlates(pairs),
			Unloadable: unloadable,
			Done:       set.CompletedAt.Valid,
			ActualReps: int(set.ActualReps.Int64),
		})
		if vm.CurrentIdx < 0 && !set.CompletedAt.Valid {
			vm.CurrentIdx = int(set.Idx)
		}
		if set.CompletedAt.Valid {
			vm.RestSeconds = restSecondsFor(raw, program.SetKind(set.Kind))
		}
	}

	// Give the PR set a number to beat.
	if best, err := s.bestE1RM(ctx, workout.LiftSlug); err == nil {
		vm.BestE1RM = best
	}

	s.render(w, r, views.Workout(vm))
}

func (s *Server) handleLogSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	workoutID, idx, err := pathIDAndIdx(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	reps, err := strconv.Atoi(r.PostFormValue("reps"))
	if err != nil || reps < 0 || reps > 500 {
		http.Redirect(w, r, fmt.Sprintf("/workout/%d#set-%d", workoutID, idx), http.StatusSeeOther)
		return
	}

	if err := s.q.LogSet(ctx, db.LogSetParams{
		ActualReps: sql.NullInt64{Int64: int64(reps), Valid: true},
		WorkoutID:  workoutID,
		Idx:        idx,
	}); err != nil {
		s.fail(w, r, fmt.Errorf("log set %d of workout %d: %w", idx, workoutID, err))
		return
	}

	s.redirectToNextSet(w, r, workoutID)
}

func (s *Server) handleUndoSet(w http.ResponseWriter, r *http.Request) {
	workoutID, idx, err := pathIDAndIdx(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.q.ClearSet(r.Context(), db.ClearSetParams{WorkoutID: workoutID, Idx: idx}); err != nil {
		s.fail(w, r, fmt.Errorf("clear set %d of workout %d: %w", idx, workoutID, err))
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d#set-%d", workoutID, idx), http.StatusSeeOther)
}

// redirectToNextSet sends the browser back to the session, anchored on the
// set now due. The page load doubles as the start of the rest timer.
func (s *Server) redirectToNextSet(w http.ResponseWriter, r *http.Request, workoutID int64) {
	sets, err := s.q.ListSets(r.Context(), workoutID)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list sets for workout %d: %w", workoutID, err))
		return
	}
	for _, set := range sets {
		if !set.CompletedAt.Valid {
			http.Redirect(w, r, fmt.Sprintf("/workout/%d?rest=1#rest", workoutID), http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d", workoutID), http.StatusSeeOther)
}

func (s *Server) handleFinishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.q.CompleteWorkout(r.Context(), id); err != nil {
		s.fail(w, r, fmt.Errorf("complete workout %d: %w", id, err))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleAbandonWorkout removes a session that was started by mistake. Sets
// cascade, and the day becomes available again.
func (s *Server) handleAbandonWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.q.DeleteWorkout(r.Context(), id); err != nil {
		s.fail(w, r, fmt.Errorf("delete workout %d: %w", id, err))
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) workoutWithSets(ctx context.Context, rawID string) (db.Workout, []db.Set, error) {
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return db.Workout{}, nil, errNotFound
	}
	workout, err := s.q.GetWorkout(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return db.Workout{}, nil, errNotFound
	}
	if err != nil {
		return db.Workout{}, nil, fmt.Errorf("load workout %d: %w", id, err)
	}
	sets, err := s.q.ListSets(ctx, id)
	if err != nil {
		return db.Workout{}, nil, fmt.Errorf("list sets for workout %d: %w", id, err)
	}
	return workout, sets, nil
}

// bestE1RM returns the best estimated one rep max recorded for a lift. Only
// anchor rounds and the TM test produce AMRAP sets, so this stays empty
// through the leaders.
func (s *Server) bestE1RM(ctx context.Context, slug string) (float64, error) {
	rows, err := s.q.ListAMRAPSets(ctx, slug)
	if err != nil {
		return 0, err
	}
	var best float64
	for _, row := range rows {
		if e := program.Epley1RM(row.Weight, int(row.ActualReps.Int64)); e > best {
			best = e
		}
	}
	return best, nil
}

var errNotFound = errors.New("not found")

func (s *Server) notFoundOrFail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, errNotFound) {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	s.fail(w, r, err)
}

func pathIDAndIdx(r *http.Request) (int64, int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	idx, err := strconv.ParseInt(r.PathValue("idx"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return id, idx, nil
}
