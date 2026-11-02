package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/program"
)

// Onboarding lets someone who is already mid-programme start using the app
// without keying in their whole training history. They name the cycle they are
// on and tick off the sessions of that cycle they have already done; earlier
// cycles are excluded from the block rather than fabricated.

// Setup returns the first-run screen: every lift, with its training max if
// one has been entered.
func (s *Service) Setup(ctx context.Context) (*Setup, error) {
	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list lifts: %w", err)
	}

	setup := &Setup{Unit: raw.Unit}
	for _, l := range lifts {
		item := LiftTM{Slug: l.Slug, Name: l.Name}
		if tm, err := s.q.GetCurrentTrainingMax(ctx, l.Slug); err == nil {
			item.Value = tm.Value
		}
		setup.Lifts = append(setup.Lifts, item)
	}
	return setup, nil
}

// SaveSetup records the first training maxes, from values by lift slug. Every
// lift needs a positive value, or nothing is written.
func (s *Service) SaveSetup(ctx context.Context, values map[string]float64) error {
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return fmt.Errorf("list lifts: %w", err)
	}
	for _, l := range lifts {
		if values[l.Slug] <= 0 {
			return inputError(fmt.Sprintf("Enter a positive training max for %s.", l.Name))
		}
	}

	return s.tx(ctx, func(q *db.Queries) error {
		for _, l := range lifts {
			if _, err := q.SetTrainingMax(ctx, db.SetTrainingMaxParams{
				LiftSlug: l.Slug,
				Value:    values[l.Slug],
				Note:     "initial setup",
			}); err != nil {
				return fmt.Errorf("set training max for %s: %w", l.Slug, err)
			}
		}
		return nil
	})
}

// SetupPosition returns the cycles a new user can say they are on.
func (s *Service) SetupPosition(ctx context.Context) (*SetupPosition, error) {
	_, cfg, err := s.activeProgram(ctx)
	if err != nil {
		return nil, err
	}

	pos := &SetupPosition{}
	for i, c := range cfg.Cycles {
		pos.Cycles = append(pos.Cycles, SetupCycle{
			Index:  i,
			Label:  c.Label,
			Detail: cycleDetail(c),
		})
	}
	return pos, nil
}

// PastSession is one session of the chosen cycle that the user says they have
// already done. PRReps is the reps on its AMRAP set, or 0 if not given.
type PastSession struct {
	Round  int
	Slug   string
	PRReps int
}

// SetupHistory returns the checklist for a cycle. It returns ErrUnknownCycle
// for a cycle the program does not have, and ErrNeedsSetup before training
// maxes are entered.
func (s *Service) SetupHistory(ctx context.Context, cycleIdx int) (*SetupHistory, error) {
	hist, _, err := s.buildSetupHistory(ctx, cycleIdx)
	return hist, err
}

// SaveSetupHistory joins the current block at a cycle and records the sessions
// of it already done, with the same errors as SetupHistory. Sessions not in
// the cycle are ignored, and so is a PR rep count for a cycle that does not ask
// for one or one outside 1 to 500.
func (s *Service) SaveSetupHistory(ctx context.Context, cycleIdx int, done []PastSession) error {
	hist, data, err := s.buildSetupHistory(ctx, cycleIdx)
	if err != nil {
		return err
	}
	type session struct {
		round int
		slug  string
	}
	prReps := make(map[session]int, len(done))
	for _, d := range done {
		prReps[session{d.Round, d.Slug}] = d.PRReps
	}

	prog, cfg, err := s.activeProgram(ctx)
	if err != nil {
		return err
	}
	block, err := s.currentBlock(ctx, prog.ID)
	if err != nil {
		return err
	}

	return s.tx(ctx, func(q *db.Queries) error {
		if err := q.SetBlockStartCycle(ctx, db.SetBlockStartCycleParams{
			StartCycleIndex: int64(hist.CycleIndex),
			ID:              block.ID,
		}); err != nil {
			return fmt.Errorf("set block start cycle: %w", err)
		}

		for _, rd := range hist.Rounds {
			for _, day := range rd.Days {
				n, ok := prReps[session{rd.Round, day.Slug}]
				if !ok {
					continue
				}
				reps := 0
				if hist.AsksReps && n > 0 && n <= maxLoggedReps {
					reps = n
				}
				if err := s.recordPastSession(ctx, q, cfg, data, block.ID, hist.CycleIndex, rd.Round, day.Slug, reps); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// recordPastSession writes a session the user says they already performed. The
// sets come from the same pure generator the live flow uses, so the weights
// match exactly what the app would have prescribed at this training max.
func (s *Service) recordPastSession(
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
			MaxReps:    int64(set.MaxReps),
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

// buildSetupHistory assembles the checklist for a cycle. Each row carries the
// weight of its top main set, which is what makes a session recognisable.
func (s *Service) buildSetupHistory(ctx context.Context, idx int) (*SetupHistory, *setupData, error) {
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
		return nil, nil, ErrUnknownCycle
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
		return nil, nil, ErrNeedsSetup
	}
	tms := make(map[string]float64, len(tmRows))
	for _, row := range tmRows {
		tms[row.Slug] = row.Value
	}
	bbbPct := make(map[string]float64, len(lifts))
	for _, lift := range lifts {
		bbbPct[lift.Slug] = bbbPctFor(lift, tms[lift.Slug])
	}

	hist := &SetupHistory{
		CycleIndex: idx,
		CycleLabel: cycle.Label,
		Unit:       raw.Unit,
	}
	switch cycle.Main {
	case program.Scheme531PR:
		hist.AsksReps, hist.RepsLabel = true, "PR set reps"
	case program.SchemeTMTest:
		hist.AsksReps, hist.RepsLabel = true, "100% set reps"
	}
	for round := 1; round <= cycle.Rounds; round++ {
		rd := SetupRound{Round: round}
		for _, lift := range lifts {
			generated, err := program.GenerateWorkout(cfg.WithBBBPct(bbbPct[lift.Slug]), idx, round, tms[lift.Slug], settings)
			if err != nil {
				return nil, nil, err
			}
			top, prTarget := topMainSet(generated)
			rd.Days = append(rd.Days, SetupDay{
				Slug:      lift.Slug,
				Name:      lift.Name,
				TopWeight: top,
				PRTarget:  prTarget,
			})
		}
		hist.Rounds = append(hist.Rounds, rd)
	}
	return hist, &setupData{Settings: settings, TMs: tms, BBBPct: bbbPct}, nil
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
	case program.SchemeDeload, program.SchemeSeventhDeload:
		main = "deload"
	case program.SchemeTMTest:
		main = "TM test"
	default:
		main = string(c.Main)
	}
	if c.Optional {
		main += " (optional)"
	}

	var supp string
	switch c.Supp.Kind {
	case program.SuppBBB:
		supp = fmt.Sprintf(" + BBB %dx%d @ %s", c.Supp.Sets, c.Supp.Reps, FormatPct(c.Supp.Pct))
	case program.SuppFSL:
		supp = fmt.Sprintf(" + FSL %dx%d", c.Supp.Sets, c.Supp.Reps)
	}

	if c.Rounds == 1 {
		return main + supp
	}
	return fmt.Sprintf("%d rounds, %s%s", c.Rounds, main, supp)
}
