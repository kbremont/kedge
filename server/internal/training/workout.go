package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kbremont/531-tracker/server/internal/db"
	"github.com/kbremont/531-tracker/server/internal/program"
)

// StartWorkout materialises the next scheduled session and returns its id.
// Sets are written with absolute weights, computed from the training max in
// force right now, so a later change to that training max leaves this session
// untouched. A session already in progress is returned instead of starting
// another.
//
// It returns ErrNothingScheduled when there is nothing to start, ErrHeld while
// a training max decision is waiting, and ErrNeedsSetup when the lift has no
// training max.
func (s *Service) StartWorkout(ctx context.Context) (int64, error) {
	state, err := s.loadState(ctx)
	if err != nil {
		return 0, err
	}
	if state.InProgress != nil {
		return state.InProgress.ID, nil
	}
	if !state.HasNext {
		return 0, ErrNothingScheduled
	}
	// Training waits on a pending training max decision; the dashboard shows it.
	if _, _, held, err := s.trainingHold(ctx, state); err != nil {
		return 0, err
	} else if held {
		return 0, ErrHeld
	}

	pos := state.Next
	lift, ok := state.LiftByDay[pos.Day]
	if !ok {
		return 0, fmt.Errorf("no lift configured for day %d", pos.Day)
	}
	tm, ok := state.TMs[lift.Slug]
	if !ok || tm <= 0 {
		return 0, ErrNeedsSetup
	}

	cycle, err := state.Config.CycleAt(pos.CycleIndex)
	if err != nil {
		return 0, err
	}
	cfg := state.Config.WithBBBPct(bbbPctFor(lift, tm))
	generated, err := program.GenerateWorkout(cfg, pos.CycleIndex, pos.Round, tm, state.Settings)
	if err != nil {
		return 0, err
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
		return 0, err
	}
	return workoutID, nil
}

// Workout returns a session, in progress or finished, or ErrNotFound.
func (s *Service) Workout(ctx context.Context, id int64) (*Workout, error) {
	workout, err := s.q.GetWorkout(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load workout %d: %w", id, err)
	}
	sets, err := s.q.ListSets(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list sets for workout %d: %w", id, err)
	}

	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	_, cfg, err := s.activeProgram(ctx)
	if err != nil {
		return nil, err
	}
	lift, err := s.q.GetLift(ctx, workout.LiftSlug)
	if err != nil {
		return nil, fmt.Errorf("load lift %s: %w", workout.LiftSlug, err)
	}

	suppKind := program.SuppNone
	if cycle, err := cfg.CycleAt(int(workout.CycleIndex)); err == nil {
		suppKind = cycle.Supp.Kind
	}

	w := &Workout{
		ID:          workout.ID,
		LiftName:    lift.Name,
		CycleLabel:  workout.CycleLabel,
		Phase:       workout.Phase,
		Unit:        raw.Unit,
		TrainingMax: workout.TrainingMax,
		CurrentIdx:  -1,
		RestSeconds: restSecondsFor(raw, program.KindMain),
		Completed:   workout.CompletedAt.Valid,
	}

	for _, set := range sets {
		pairs, unloadable := program.PlatesFor(set.Weight, settings)
		w.Sets = append(w.Sets, Set{
			Idx:        int(set.Idx),
			Tag:        setTag(program.SetKind(set.Kind), suppKind),
			Weight:     set.Weight,
			Pct:        set.Pct,
			TargetReps: int(set.TargetReps),
			MaxReps:    int(set.MaxReps),
			IsAMRAP:    set.IsAmrap != 0,
			Plates:     formatPlates(pairs),
			Unloadable: unloadable,
			Done:       set.CompletedAt.Valid,
			ActualReps: int(set.ActualReps.Int64),
		})
		if w.CurrentIdx < 0 && !set.CompletedAt.Valid {
			w.CurrentIdx = int(set.Idx)
		}
		if set.CompletedAt.Valid {
			w.RestSeconds = restSecondsFor(raw, program.SetKind(set.Kind))
		}
	}

	// Give the PR set a number to beat.
	if best, err := s.BestE1RM(ctx, workout.LiftSlug); err == nil {
		w.BestE1RM = best
	}
	return w, nil
}

// maxLoggedReps bounds the reps a set can be logged with.
const maxLoggedReps = 500

// LogSet records the reps done on a set.
func (s *Service) LogSet(ctx context.Context, workoutID, idx int64, reps int) error {
	if reps < 0 || reps > maxLoggedReps {
		return inputError(fmt.Sprintf("Reps must be between 0 and %d.", maxLoggedReps))
	}
	if err := s.q.LogSet(ctx, db.LogSetParams{
		ActualReps: sql.NullInt64{Int64: int64(reps), Valid: true},
		WorkoutID:  workoutID,
		Idx:        idx,
	}); err != nil {
		return fmt.Errorf("log set %d of workout %d: %w", idx, workoutID, err)
	}
	return nil
}

// UndoSet clears a logged set.
func (s *Service) UndoSet(ctx context.Context, workoutID, idx int64) error {
	if err := s.q.ClearSet(ctx, db.ClearSetParams{WorkoutID: workoutID, Idx: idx}); err != nil {
		return fmt.Errorf("clear set %d of workout %d: %w", idx, workoutID, err)
	}
	return nil
}

// FinishWorkout marks a session complete.
func (s *Service) FinishWorkout(ctx context.Context, id int64) error {
	if err := s.q.CompleteWorkout(ctx, id); err != nil {
		return fmt.Errorf("complete workout %d: %w", id, err)
	}
	return nil
}

// AbandonWorkout removes a session that was started by mistake. Sets cascade,
// and the day becomes available again.
func (s *Service) AbandonWorkout(ctx context.Context, id int64) error {
	if err := s.q.DeleteWorkout(ctx, id); err != nil {
		return fmt.Errorf("delete workout %d: %w", id, err)
	}
	return nil
}

// BestE1RM returns the best estimated one rep max known for a lift, from the
// estimates entered from before the app as well as the sets logged in it. Only
// anchor rounds and the TM test produce AMRAP sets, so without past estimates
// this stays empty through the leaders.
func (s *Service) BestE1RM(ctx context.Context, slug string) (float64, error) {
	ests, err := s.estimates(ctx, slug)
	if err != nil {
		return 0, err
	}
	var best float64
	for _, est := range ests {
		if e := program.Epley1RM(est.Weight, est.Reps); e > best {
			best = e
		}
	}
	return best, nil
}
