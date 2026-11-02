package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/program"
)

// Training max decisions. There are exactly two points where the user is asked
// to settle a training max before training goes on:
//
//   - after a lift's 7th week TM test, but only if it failed: fewer than 3 reps
//     at 100% offers to lower that lift's training max, and the anchor waits
//     until the user lowers it or keeps it. A pass is reported, not asked
//     about. Nothing goes up here, so the anchor never runs on a training max
//     raised partway through the block; and
//   - at the start of every block after the first, the one place a training
//     max goes up, recommended from the last block's TM test and anchor.
//
// Both are offers. Nothing is written until the user accepts, and what is
// written is whatever they left in the inputs.

func reviewKey(blockID int64, cycleIdx int) string { return fmt.Sprintf("b%d:c%d", blockID, cycleIdx) }

// tmTestResults reports the current block's TM tests worth showing: every
// failed lift still waiting on a decision, and the session just finished while
// nothing has been started after it.
func (s *Service) tmTestResults(ctx context.Context, st *blockState) ([]TMTestResult, error) {
	tops, err := s.q.ListTMTestTopSets(ctx, st.Block.ID)
	if err != nil {
		return nil, fmt.Errorf("list TM test results in block %d: %w", st.Block.ID, err)
	}
	if len(tops) == 0 {
		return nil, nil
	}

	var latestID int64
	latest, err := s.q.GetLatestWorkoutInBlock(ctx, st.Block.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("load latest workout in block %d: %w", st.Block.ID, err)
	}
	if err == nil && latest.CompletedAt.Valid {
		latestID = latest.ID
	}

	names := make(map[string]string, len(st.Lifts))
	for _, l := range st.Lifts {
		names[l.Slug] = l.Name
	}

	var out []TMTestResult
	for _, t := range tops {
		reps := int(t.ActualReps.Int64)
		verdict := program.TMTestVerdict(reps)
		failed := verdict == program.VerdictTooHigh
		// A failure stays until it is settled. A pass is news only until the
		// next session starts.
		waiting := failed && !t.TmReviewedAt.Valid
		justDone := !failed && t.ID == latestID
		if !waiting && !justDone {
			continue
		}

		current := st.TMs[t.LiftSlug]
		r := TMTestResult{
			WorkoutID: t.ID,
			LiftName:  names[t.LiftSlug],
			LiftSlug:  t.LiftSlug,
			Unit:      st.Raw.Unit,
			Tested:    t.TrainingMax,
			Reps:      reps,
			Current:   current,
		}
		switch verdict {
		case program.VerdictTooHigh:
			r.Headline = "Your TM is too high."
			r.Detail = "Under 3 reps at 100%. Lower it before the anchor, or keep it."
			r.Tone = "down"
			r.NeedsDecision = true
			r.Proposed = program.TMTestProposal(current, reps, st.Settings)
		case program.VerdictBump:
			r.Headline = "Strong set. This TM goes up at the end of the block."
			r.Detail = "6 or more at 100% means there is room to grow; the bump comes with the next block."
			r.Tone = "up"
		default:
			r.Headline = "Nice job, your TM is right."
			r.Detail = "3 to 5 reps at 100% is right where it should be."
			r.Tone = "hold"
		}
		out = append(out, r)
	}
	return out, nil
}

// awaitingTMTestDecision reports whether a failed TM test is holding training.
// Only the anchor waits on it; the rest of the TM test week carries on.
func awaitingTMTestDecision(st *blockState, results []TMTestResult) bool {
	if !st.HasNext {
		return false
	}
	if cycle, err := st.Config.CycleAt(st.Next.CycleIndex); err == nil && cycle.Phase == program.PhaseTMTest {
		return false
	}
	for _, r := range results {
		if r.NeedsDecision {
			return true
		}
	}
	return false
}

// trainingHold gathers what the dashboard shows above the next session, and
// reports whether that session has to wait for a decision.
func (s *Service) trainingHold(ctx context.Context, st *blockState) (results []TMTestResult, review *TMReview, held bool, err error) {
	if results, err = s.tmTestResults(ctx, st); err != nil {
		return nil, nil, false, err
	}
	if review, err = s.pendingReview(ctx, st); err != nil {
		return nil, nil, false, err
	}
	return results, review, review != nil || awaitingTMTestDecision(st, results), nil
}

// pendingReview returns the block review waiting to be accepted, or nil. It
// sits before the first session of every block after the first; accepting it
// records its key, which is what stops it reappearing.
func (s *Service) pendingReview(ctx context.Context, st *blockState) (*TMReview, error) {
	if st.InProgress != nil || !st.HasNext || st.Block.Number < 2 {
		return nil, nil
	}
	next := st.Next
	if len(st.Schedule) == 0 || next != st.Schedule[0] || st.cycleStarted(next.CycleIndex) {
		return nil, nil
	}
	key := reviewKey(st.Block.ID, next.CycleIndex)
	if st.Raw.DismissedCycle == key {
		return nil, nil
	}
	return s.blockReview(ctx, st, key)
}

func (s *Service) blockReview(ctx context.Context, st *blockState, key string) (*TMReview, error) {
	prev, err := s.q.GetBlockByNumber(ctx, st.Block.Number-1)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load block %d: %w", st.Block.Number-1, err)
	}

	tops, err := s.q.ListTMTestTopSets(ctx, prev.ID)
	if err != nil {
		return nil, fmt.Errorf("list TM test results in block %d: %w", prev.ID, err)
	}
	testReps := make(map[string]int, len(tops))
	for _, t := range tops {
		testReps[t.LiftSlug] = int(t.ActualReps.Int64)
	}

	prRows, err := s.q.ListAnchorPRSets(ctx, prev.ID)
	if err != nil {
		return nil, fmt.Errorf("list anchor PR sets in block %d: %w", prev.ID, err)
	}
	anchor := make(map[string][]program.PRSet)
	for _, pr := range prRows {
		anchor[pr.LiftSlug] = append(anchor[pr.LiftSlug], program.PRSet{
			Target: int(pr.TargetReps),
			Actual: int(pr.ActualReps.Int64),
		})
	}

	vm := &TMReview{
		Title: fmt.Sprintf("Block %d: new training maxes", st.Block.Number),
		Intro: "Recommended from last block's TM test and anchor PR sets. Edit any of them, then accept to start the block.",
		Key:   key,
		Unit:  st.Raw.Unit,
	}
	for _, tm := range st.TMOrder {
		reps, hasTest := testReps[tm.Slug]
		proposed, reason := program.RecommendBump(st.Config, tm.Value, tm.IsUpper != 0, anchor[tm.Slug], reps, hasTest, st.Settings)
		verdict, tone := bumpVerdict(reason, proposed-tm.Value, st.Raw.Unit)
		vm.Rows = append(vm.Rows, TMReviewRow{
			Slug:     tm.Slug,
			Name:     tm.Name,
			Evidence: bumpEvidence(anchor[tm.Slug], reps, hasTest),
			Verdict:  verdict,
			Tone:     tone,
			Current:  tm.Value,
			Proposed: proposed,
		})
	}
	return vm, nil
}

// bumpEvidence summarises what a block-end recommendation was based on, for
// example "PR sets 8, 5, 3 (min 5, 3, 1) · TM test 7".
func bumpEvidence(anchor []program.PRSet, testReps int, hasTest bool) string {
	var parts []string
	if len(anchor) == 0 {
		parts = append(parts, "No anchor PR sets recorded")
	} else {
		did := make([]string, len(anchor))
		need := make([]string, len(anchor))
		for i, pr := range anchor {
			did[i] = strconv.Itoa(pr.Actual)
			need[i] = strconv.Itoa(pr.Target)
		}
		parts = append(parts, fmt.Sprintf("PR sets %s (min %s)", strings.Join(did, ", "), strings.Join(need, ", ")))
	}
	if hasTest {
		parts = append(parts, fmt.Sprintf("TM test %d", testReps))
	}
	return strings.Join(parts, " · ")
}

func bumpVerdict(reason program.BumpReason, delta float64, unit string) (text, tone string) {
	switch reason {
	case program.ReasonMissedPR:
		return "Missed a PR set: reset 10%", "down"
	case program.ReasonHitPRAndTM:
		return "PR sets and TM test: +" + FormatWeightUnit(delta, unit), "up"
	case program.ReasonHitPR:
		return "Hit every PR set: +" + FormatWeightUnit(delta, unit), "up"
	default:
		return "Hold", "hold"
	}
}

// AcceptReview writes the training maxes the user settled on at the start of
// a block, from values by lift slug. The review is rebuilt here rather than
// trusted from the caller, so a stale screen cannot accept a review that is no
// longer pending: that returns ErrStale. Every lift needs a positive value, or
// nothing is written.
func (s *Service) AcceptReview(ctx context.Context, key string, values map[string]float64) error {
	state, err := s.loadState(ctx)
	if err != nil {
		return err
	}
	review, err := s.pendingReview(ctx, state)
	if err != nil {
		return err
	}
	if review == nil || review.Key != key {
		return ErrStale
	}

	for _, row := range review.Rows {
		if v := values[row.Slug]; v <= 0 {
			// All or nothing: a half-applied review is worse than none.
			return inputError(fmt.Sprintf("Enter a positive training max for %s.", row.Name))
		}
	}

	return s.tx(ctx, func(q *db.Queries) error {
		for _, row := range review.Rows {
			if values[row.Slug] == row.Current {
				continue
			}
			if _, err := q.SetTrainingMax(ctx, db.SetTrainingMaxParams{
				LiftSlug: row.Slug,
				Value:    values[row.Slug],
				Note:     "block bump",
			}); err != nil {
				return fmt.Errorf("set training max for %s: %w", row.Slug, err)
			}
		}
		if err := q.DismissCycle(ctx, review.Key); err != nil {
			return fmt.Errorf("record accepted review: %w", err)
		}
		return nil
	})
}

// DecideTMTest settles a failed TM test: lower the training max to tm, or keep
// it. Either way the anchor stops waiting on it. The offer is rebuilt from the
// database rather than trusted from the caller, so a test with no decision
// waiting returns ErrStale. It can only ever lower: a value at or above the
// current training max keeps it as it is.
func (s *Service) DecideTMTest(ctx context.Context, workoutID int64, keep bool, tm float64) error {
	state, err := s.loadState(ctx)
	if err != nil {
		return err
	}
	results, err := s.tmTestResults(ctx, state)
	if err != nil {
		return err
	}
	var offer *TMTestResult
	for i := range results {
		if results[i].WorkoutID == workoutID && results[i].NeedsDecision {
			offer = &results[i]
		}
	}
	if offer == nil {
		return ErrStale
	}

	value := offer.Current
	if !keep {
		if tm <= 0 {
			return inputError("Enter a positive training max.")
		}
		value = tm
	}

	workout, err := s.q.GetWorkout(ctx, workoutID)
	if err != nil {
		return fmt.Errorf("load workout %d: %w", workoutID, err)
	}
	return s.tx(ctx, func(q *db.Queries) error {
		// Only a lower value is written; anything else keeps the training max.
		if value < offer.Current {
			if _, err := q.SetTrainingMax(ctx, db.SetTrainingMaxParams{
				LiftSlug: workout.LiftSlug,
				Value:    value,
				Note:     "tm test",
			}); err != nil {
				return fmt.Errorf("set training max for %s: %w", workout.LiftSlug, err)
			}
		}
		if err := q.MarkTMTestReviewed(ctx, workoutID); err != nil {
			return fmt.Errorf("settle TM test %d: %w", workoutID, err)
		}
		return nil
	})
}

// SkipCycle passes on an optional cycle before its first session. The next
// state load leaves it out of the schedule; skipping the block's last cycle
// ends the block, and the new one opens with its training max review. It
// returns ErrStale when the next cycle cannot be skipped, and ErrHeld while a
// training max decision is waiting.
func (s *Service) SkipCycle(ctx context.Context) error {
	state, err := s.loadState(ctx)
	if err != nil {
		return err
	}
	if !state.canSkipNext() {
		return ErrStale
	}
	if _, _, held, err := s.trainingHold(ctx, state); err != nil {
		return err
	} else if held {
		return ErrHeld
	}

	if err := s.q.SkipCycle(ctx, db.SkipCycleParams{
		BlockID:    state.Block.ID,
		CycleIndex: int64(state.Next.CycleIndex),
	}); err != nil {
		return fmt.Errorf("skip cycle %d: %w", state.Next.CycleIndex, err)
	}
	return nil
}
