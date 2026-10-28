package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kbremont/531-tracker/server/internal/db"
	"github.com/kbremont/531-tracker/server/internal/program"
)

// blockState is everything the dashboard and the session routes need to know
// about where the user currently is in their programming.
type blockState struct {
	Block    db.Block
	Config   program.Config
	Settings program.Settings
	Raw      db.Setting

	Schedule []program.Position
	Done     map[program.Position]bool
	Next     program.Position
	HasNext  bool

	// Skipped holds the optional cycles the user passed on in this block.
	Skipped map[int]bool

	// InProgress is a session that was started but never finished.
	InProgress *db.Workout

	Lifts     []db.Lift
	LiftByDay map[int]db.Lift
	TMs       map[string]float64
	TMOrder   []db.ListCurrentTrainingMaxesRow
}

// loadSettings reads the settings row and converts it into the loading
// preferences the program engine works with.
func (s *Server) loadSettings(ctx context.Context) (db.Setting, program.Settings, error) {
	raw, err := s.q.GetSettings(ctx)
	if err != nil {
		return db.Setting{}, program.Settings{}, fmt.Errorf("load settings: %w", err)
	}

	var plates []float64
	if err := json.Unmarshal([]byte(raw.Plates), &plates); err != nil {
		// A malformed plate list should not take the whole app down; fall
		// back to a standard set and carry on.
		s.log.WarnContext(ctx, "settings.plates is not valid JSON, using defaults", "value", raw.Plates)
		plates = program.DefaultSettings().Plates
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(plates)))

	return raw, program.Settings{
		Unit:      raw.Unit,
		Rounding:  raw.Rounding,
		BarWeight: raw.BarWeight,
		Plates:    plates,
		Warmups:   raw.Warmups != 0,
	}, nil
}

// activeProgram returns the configured program, seeding the default one the
// first time the app runs.
func (s *Server) activeProgram(ctx context.Context) (db.Program, program.Config, error) {
	row, err := s.q.GetActiveProgram(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		cfg := program.LeaderAnchorBBB()
		encoded, mErr := json.Marshal(cfg)
		if mErr != nil {
			return db.Program{}, program.Config{}, fmt.Errorf("encode default program: %w", mErr)
		}
		row, err = s.q.InsertProgram(ctx, db.InsertProgramParams{
			Name:     cfg.Name,
			Config:   string(encoded),
			IsActive: 1,
		})
		if err != nil {
			return db.Program{}, program.Config{}, fmt.Errorf("seed default program: %w", err)
		}
		return row, cfg, nil
	}
	if err != nil {
		return db.Program{}, program.Config{}, fmt.Errorf("load active program: %w", err)
	}

	var cfg program.Config
	if err := json.Unmarshal([]byte(row.Config), &cfg); err != nil {
		return db.Program{}, program.Config{}, fmt.Errorf("decode program %d: %w", row.ID, err)
	}
	return row, cfg, nil
}

// currentBlock returns the open block, starting the first one if needed.
func (s *Server) currentBlock(ctx context.Context, programID int64) (db.Block, error) {
	block, err := s.q.GetCurrentBlock(ctx)
	if err == nil {
		return block, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.Block{}, fmt.Errorf("load current block: %w", err)
	}

	count, err := s.q.CountBlocks(ctx)
	if err != nil {
		return db.Block{}, fmt.Errorf("count blocks: %w", err)
	}
	block, err = s.q.InsertBlock(ctx, db.InsertBlockParams{
		ProgramID: programID,
		Number:    count + 1,
	})
	if err != nil {
		return db.Block{}, fmt.Errorf("start block: %w", err)
	}
	return block, nil
}

// loadState assembles the current position in the program. When the open
// block turns out to be finished it is closed and the next one started, so
// the user never has to do anything to roll over.
func (s *Server) loadState(ctx context.Context) (*blockState, error) {
	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	prog, cfg, err := s.activeProgram(ctx)
	if err != nil {
		return nil, err
	}

	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list lifts: %w", err)
	}
	liftByDay := make(map[int]db.Lift, len(lifts))
	dayBySlug := make(map[string]int, len(lifts))
	for _, l := range lifts {
		liftByDay[int(l.DayOrder)] = l
		dayBySlug[l.Slug] = int(l.DayOrder)
	}

	tmRows, err := s.q.ListCurrentTrainingMaxes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list training maxes: %w", err)
	}
	tms := make(map[string]float64, len(tmRows))
	for _, r := range tmRows {
		tms[r.Slug] = r.Value
	}

	// Two passes at most: if the open block is already complete, close it,
	// open the next, and look again.
	var state *blockState
	for attempt := 0; attempt < 2; attempt++ {
		block, err := s.currentBlock(ctx, prog.ID)
		if err != nil {
			return nil, err
		}

		skippedRows, err := s.q.ListSkippedCycles(ctx, block.ID)
		if err != nil {
			return nil, fmt.Errorf("list skipped cycles in block %d: %w", block.ID, err)
		}
		skipped := make(map[int]bool, len(skippedRows))
		for _, idx := range skippedRows {
			skipped[int(idx)] = true
		}

		// The schedule depends on the block, not just the program: a block
		// joined partway through during setup starts at its own cycle, and an
		// optional cycle the user skipped is left out.
		schedule := cfg.BlockSchedule(len(lifts), int(block.StartCycleIndex), skipped)

		workouts, err := s.q.ListWorkoutsInBlock(ctx, block.ID)
		if err != nil {
			return nil, fmt.Errorf("list workouts in block %d: %w", block.ID, err)
		}

		done := make(map[program.Position]bool, len(workouts))
		var inProgress *db.Workout
		for i, w := range workouts {
			pos := program.Position{
				CycleIndex: int(w.CycleIndex),
				Round:      int(w.Round),
				Day:        dayBySlug[w.LiftSlug],
			}
			if w.CompletedAt.Valid {
				done[pos] = true
			} else if inProgress == nil {
				inProgress = &workouts[i]
			}
		}

		next, hasNext := program.NextPosition(schedule, done)
		state = &blockState{
			Block:      block,
			Config:     cfg,
			Settings:   settings,
			Raw:        raw,
			Schedule:   schedule,
			Done:       done,
			Next:       next,
			HasNext:    hasNext,
			Skipped:    skipped,
			InProgress: inProgress,
			Lifts:      lifts,
			LiftByDay:  liftByDay,
			TMs:        tms,
			TMOrder:    tmRows,
		}

		if hasNext || inProgress != nil {
			return state, nil
		}
		if err := s.q.CompleteBlock(ctx, block.ID); err != nil {
			return nil, fmt.Errorf("complete block %d: %w", block.ID, err)
		}
	}
	return state, nil
}

// cycleStarted reports whether any session of a cycle has been started in the
// current block, finished or not.
func (st *blockState) cycleStarted(cycleIdx int) bool {
	if st.InProgress != nil && int(st.InProgress.CycleIndex) == cycleIdx {
		return true
	}
	for p := range st.Done {
		if p.CycleIndex == cycleIdx {
			return true
		}
	}
	return false
}

// canSkipNext reports whether the next scheduled cycle is optional and still
// untouched, which is the only time skipping it is offered.
func (st *blockState) canSkipNext() bool {
	if st.InProgress != nil || !st.HasNext {
		return false
	}
	cycle, err := st.Config.CycleAt(st.Next.CycleIndex)
	if err != nil {
		return false
	}
	return cycle.Optional && !st.cycleStarted(st.Next.CycleIndex)
}

// bbbPctFor turns a lift's BBB preference into the fraction of its training max
// the engine should load, or 0 to leave the program's own percentage alone.
// An absolute weight becomes a fraction here, where the training max is known.
func bbbPctFor(lift db.Lift, tm float64) float64 {
	switch lift.BbbMode {
	case "percent":
		if lift.BbbValue > 0 {
			return lift.BbbValue
		}
	case "weight":
		if lift.BbbValue > 0 && tm > 0 {
			return lift.BbbValue / tm
		}
	}
	return 0
}

// restSecondsFor returns the rest interval for the kind of set just completed.
func restSecondsFor(raw db.Setting, kind program.SetKind) int {
	switch kind {
	case program.KindWarmup:
		return int(raw.RestWarmupSeconds)
	case program.KindSupp:
		return int(raw.RestSuppSeconds)
	default:
		return int(raw.RestMainSeconds)
	}
}

// roundOfBlock converts a position into its 1-indexed round within the whole
// block, for the "round 7 of 10" line on the dashboard.
func roundOfBlock(cfg program.Config, p program.Position) int {
	round := 0
	for i, cycle := range cfg.Cycles {
		if i == p.CycleIndex {
			return round + p.Round
		}
		round += cycle.Rounds
	}
	return round
}

// parsePlates accepts the comma-separated list shown in settings and returns
// it sorted heaviest first.
func parsePlates(in string) ([]float64, error) {
	fields := strings.FieldsFunc(in, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]float64, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", f)
		}
		if v > 0 {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("list at least one plate")
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(out)))
	return out, nil
}

func formatPlates(plates []float64) string {
	parts := make([]string, len(plates))
	for i, p := range plates {
		parts[i] = strconv.FormatFloat(p, 'f', -1, 64)
	}
	return strings.Join(parts, ", ")
}
