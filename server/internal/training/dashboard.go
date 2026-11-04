package training

import (
	"context"
	"fmt"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/program"
)

const recentLimit = 8

// Dashboard returns the home screen: the next session, any training max
// decision waiting in front of it, the training maxes, and recent sessions.
func (s *Service) Dashboard(ctx context.Context) (*Dashboard, error) {
	state, err := s.loadState(ctx)
	if err != nil {
		return nil, err
	}
	// No training maxes means the app has never been set up.
	if len(state.TMOrder) < len(state.Lifts) {
		return &Dashboard{NeedsSetup: true}, nil
	}

	d := &Dashboard{
		Unit:        state.Raw.Unit,
		ProgramName: state.Config.Name,
	}
	for _, tm := range state.TMOrder {
		d.TMs = append(d.TMs, LiftTM{Slug: tm.Slug, Name: tm.Name, Value: tm.Value})
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
		d.Next = &NextUp{
			LiftName:      lift.Name,
			LiftSlug:      lift.Slug,
			CycleLabel:    state.InProgress.CycleLabel,
			Phase:         state.InProgress.Phase,
			TrainingMax:   state.InProgress.TrainingMax,
			InProgress:    true,
			WorkoutID:     state.InProgress.ID,
			BlockNumber:   int(state.Block.Number),
			RoundOfBlock:  roundOfBlock(state.Config, program.Position{CycleIndex: int(state.InProgress.CycleIndex), Round: int(state.InProgress.Round)}),
			RoundsInBlock: state.Config.RoundsPerBlock(),
		}
	case state.HasNext:
		lift := state.LiftByDay[state.Next.Day]
		d.Next = &NextUp{
			LiftName:      lift.Name,
			LiftSlug:      lift.Slug,
			CycleLabel:    state.Config.Describe(state.Next),
			TrainingMax:   state.TMs[lift.Slug],
			Day:           state.Next.Day,
			BlockNumber:   int(state.Block.Number),
			RoundOfBlock:  roundOfBlock(state.Config, state.Next),
			RoundsInBlock: state.Config.RoundsPerBlock(),
		}
		if cycle, err := state.Config.CycleAt(state.Next.CycleIndex); err == nil {
			d.Next.Phase = cycle.Phase
		}
	}

	results, review, held, err := s.trainingHold(ctx, state)
	if err != nil {
		return nil, err
	}
	d.TestResults = results
	d.Review = review
	if d.Next != nil && !d.Next.InProgress {
		d.Next.Blocked = held
		d.Next.CanSkip = state.canSkipNext()
	}

	recent, err := s.recentHistory(ctx, recentLimit)
	if err != nil {
		return nil, err
	}
	d.Recent = recent
	return d, nil
}

// History returns the completed sessions, newest first, and the lifts.
func (s *Service) History(ctx context.Context) (*History, error) {
	items, err := s.recentHistory(ctx, 60)
	if err != nil {
		return nil, err
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list lifts: %w", err)
	}

	h := &History{Items: items}
	for _, l := range lifts {
		h.Lifts = append(h.Lifts, LiftTM{Slug: l.Slug, Name: l.Name})
	}
	return h, nil
}

// recentHistory summarises completed sessions. The per-workout set lookup is
// a query each, which is the right trade against a more complex join when the
// database is a local file and the list is at most a screen or two long.
func (s *Service) recentHistory(ctx context.Context, limit int64) ([]HistoryItem, error) {
	rows, err := s.q.ListRecentWorkouts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent workouts: %w", err)
	}

	items := make([]HistoryItem, 0, len(rows))
	for _, row := range rows {
		sets, err := s.q.ListSets(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("list sets for workout %d: %w", row.ID, err)
		}
		items = append(items, HistoryItem{
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
		case set.IsAmrap == best.IsAmrap && liftedWeight(set) > liftedWeight(best):
			best = set
		}
	}
	if !found {
		return ""
	}
	return fmt.Sprintf("%s x %d", FormatWeight(liftedWeight(best)), best.ActualReps.Int64)
}
