package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/program"
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
		LiftSlug:    lift.Slug,
		CycleLabel:  workout.CycleLabel,
		Phase:       workout.Phase,
		Unit:        raw.Unit,
		TrainingMax: workout.TrainingMax,
		CurrentIdx:  -1,
		RestSeconds: restSecondsFor(raw, program.KindMain),
		Completed:   workout.CompletedAt.Valid,
	}

	for _, set := range sets {
		// A weight changed at the bar is what the set was, in every figure.
		weight, pct := liftedWeight(set), set.Pct
		if set.ActualWeight.Valid {
			pct = weight / workout.TrainingMax
		}
		pairs, unloadable := program.PlatesFor(weight, settings)
		w.Sets = append(w.Sets, Set{
			Idx:            int(set.Idx),
			Tag:            setTag(program.SetKind(set.Kind), suppKind),
			Weight:         weight,
			WeightEditable: set.IsAmrap == 0,
			Pct:            pct,
			TargetReps:     int(set.TargetReps),
			MaxReps:        int(set.MaxReps),
			IsAMRAP:        set.IsAmrap != 0,
			Plates:         formatPlates(pairs),
			Unloadable:     unloadable,
			Done:           set.CompletedAt.Valid,
			ActualReps:     int(set.ActualReps.Int64),
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

// maxLoggedReps and maxLoggedWeight bound what a set can be logged with.
const (
	maxLoggedReps   = 500
	maxLoggedWeight = 2000
)

// liftedWeight is the weight a set was lifted at: the one the lifter changed
// it to, or else the one suggested.
func liftedWeight(set db.Set) float64 {
	if set.ActualWeight.Valid {
		return set.ActualWeight.Float64
	}
	return set.Weight
}

func checkWeight(weight float64) error {
	if !(weight >= minEstimateWeight && weight <= maxLoggedWeight) { // the negation also catches NaN
		return inputError(fmt.Sprintf("Weight must be between %d and %d.", minEstimateWeight, maxLoggedWeight))
	}
	return nil
}

// LogSet records the reps done on a set, and the weight they were done at
// when that isn't the one suggested. A nil weight means the suggested one.
func (s *Service) LogSet(ctx context.Context, workoutID, idx int64, reps int, weight *float64) error {
	if reps < 0 || reps > maxLoggedReps {
		return inputError(fmt.Sprintf("Reps must be between 0 and %d.", maxLoggedReps))
	}
	changed, err := s.changedWeight(ctx, workoutID, idx, weight)
	if err != nil {
		return err
	}
	if err := s.q.LogSet(ctx, db.LogSetParams{
		ActualReps:   sql.NullInt64{Int64: int64(reps), Valid: true},
		ActualWeight: changed,
		WorkoutID:    workoutID,
		Idx:          idx,
	}); err != nil {
		return fmt.Errorf("log set %d of workout %d: %w", idx, workoutID, err)
	}
	return nil
}

// changedWeight is what to store as a set's actual weight: NULL unless it
// differs from the suggestion, so a value there always means it was changed.
// A PR set or TM test keeps its weight, since the verdicts read its reps as
// done at that weight.
func (s *Service) changedWeight(ctx context.Context, workoutID, idx int64, weight *float64) (sql.NullFloat64, error) {
	if weight == nil {
		return sql.NullFloat64{}, nil
	}
	set, err := s.q.GetSet(ctx, db.GetSetParams{WorkoutID: workoutID, Idx: idx})
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullFloat64{}, ErrNotFound
	}
	if err != nil {
		return sql.NullFloat64{}, fmt.Errorf("load set %d of workout %d: %w", idx, workoutID, err)
	}
	if *weight == set.Weight {
		return sql.NullFloat64{}, nil
	}
	if set.IsAmrap != 0 {
		return sql.NullFloat64{}, inputError("The weight of a PR set or TM test can't be changed.")
	}
	if err := checkWeight(*weight); err != nil {
		return sql.NullFloat64{}, err
	}
	return sql.NullFloat64{Float64: *weight, Valid: true}, nil
}

// PlatesFor says how to load the bar for a weight the lifter is typing, so a
// change at the bar is loaded by the same math as a suggestion. It writes
// nothing.
func (s *Service) PlatesFor(ctx context.Context, weight float64) (*Plates, error) {
	if err := checkWeight(weight); err != nil {
		return nil, err
	}
	_, settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	pairs, unloadable := program.PlatesFor(weight, settings)
	return &Plates{Plates: formatPlates(pairs), Unloadable: unloadable}, nil
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
