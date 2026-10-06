package program

import (
	"fmt"
	"math"
)

// setSpec is one entry in a percentage table: a fraction of the training max,
// a rep target, and whether the set is taken to a rep max. maxReps is set only
// for a rep range, where reps is the bottom of it.
type setSpec struct {
	pct     float64
	reps    int
	amrap   bool
	maxReps int
}

// Percentage tables, indexed by round within the cycle. These are the numbers
// from the book and should not be edited casually; the tests pin them.
var (
	fivesPro = [][]setSpec{
		{{0.65, 5, false, 0}, {0.75, 5, false, 0}, {0.85, 5, false, 0}},
		{{0.70, 5, false, 0}, {0.80, 5, false, 0}, {0.90, 5, false, 0}},
		{{0.75, 5, false, 0}, {0.85, 5, false, 0}, {0.95, 5, false, 0}},
	}

	fiveThreeOnePR = [][]setSpec{
		{{0.65, 5, false, 0}, {0.75, 5, false, 0}, {0.85, 5, true, 0}},
		{{0.70, 3, false, 0}, {0.80, 3, false, 0}, {0.90, 3, true, 0}},
		{{0.75, 5, false, 0}, {0.85, 3, false, 0}, {0.95, 1, true, 0}},
	}

	deloadRound = [][]setSpec{
		{{0.40, 5, false, 0}, {0.50, 5, false, 0}, {0.60, 5, false, 0}},
	}

	// The 5/3/1 Forever 7th week protocols. The test's last set is logged as
	// performed, since the reps at 100% are the whole point of it.
	tmTestRound = [][]setSpec{
		{{0.70, 5, false, 0}, {0.80, 5, false, 0}, {0.90, 5, false, 0}, {1.00, 3, false, 5}},
	}
	seventhDeloadRound = [][]setSpec{
		{{0.70, 5, false, 0}, {0.80, 3, false, 5}, {0.90, 1, false, 0}, {1.00, 1, false, 0}},
	}

	warmupSets = []setSpec{{0.40, 5, false, 0}, {0.50, 5, false, 0}, {0.60, 3, false, 0}}
)

func mainTable(s MainScheme) ([][]setSpec, error) {
	switch s {
	case SchemeFivesPro:
		return fivesPro, nil
	case Scheme531PR:
		return fiveThreeOnePR, nil
	case SchemeDeload:
		return deloadRound, nil
	case SchemeTMTest:
		return tmTestRound, nil
	case SchemeSeventhDeload:
		return seventhDeloadRound, nil
	default:
		return nil, fmt.Errorf("program: unknown main scheme %q", s)
	}
}

// GenerateWorkout expands a program into the concrete sets for one training
// day: cycleIdx selects the cycle within the block, round is 1-indexed within
// that cycle, and tm is the training max for the lift being trained.
//
// The caller is responsible for supplying the training max in force at the
// time the workout is generated. Weights are absolute, so a later change to
// the training max does not rewrite completed history.
func GenerateWorkout(cfg Config, cycleIdx, round int, tm float64, s Settings) (Workout, error) {
	if cycleIdx < 0 || cycleIdx >= len(cfg.Cycles) {
		return Workout{}, fmt.Errorf("program: cycle index %d out of range (block has %d cycles)", cycleIdx, len(cfg.Cycles))
	}
	cycle := cfg.Cycles[cycleIdx]

	if round < 1 || round > cycle.Rounds {
		return Workout{}, fmt.Errorf("program: round %d out of range for cycle %q (%d rounds)", round, cycle.Label, cycle.Rounds)
	}
	if tm <= 0 {
		return Workout{}, fmt.Errorf("program: training max must be positive, got %v", tm)
	}

	table, err := mainTable(cycle.Main)
	if err != nil {
		return Workout{}, err
	}
	if round > len(table) {
		return Workout{}, fmt.Errorf("program: scheme %q has no round %d", cycle.Main, round)
	}
	mainWork := table[round-1]

	out := Workout{
		CycleLabel: cycle.Label,
		Phase:      cycle.Phase,
		Round:      round,
	}

	// The original deload's working sets are already 40/50/60, so a separate
	// warmup would just repeat them. The 7th week protocols start at 70% and
	// warm up as normal.
	if s.Warmups && cycle.Main != SchemeDeload {
		for _, w := range warmupSets {
			out.Sets = append(out.Sets, buildSet(KindWarmup, w, tm, s))
		}
	}

	for _, m := range mainWork {
		out.Sets = append(out.Sets, buildSet(KindMain, m, tm, s))
	}

	for _, sup := range supplementalSpecs(cycle.Supp, mainWork) {
		out.Sets = append(out.Sets, buildSet(KindSupp, sup, tm, s))
	}

	for i := range out.Sets {
		out.Sets[i].Index = i
	}
	return out, nil
}

// supplementalSpecs expands a supplemental definition into individual sets.
// First Set Last has no fixed percentage of its own; it reuses the opening
// main set of whichever round is being run.
func supplementalSpecs(sup Supplemental, mainWork []setSpec) []setSpec {
	if sup.Kind == SuppNone || sup.Sets <= 0 {
		return nil
	}

	pct := sup.Pct
	if sup.Kind == SuppFSL {
		if len(mainWork) == 0 {
			return nil
		}
		pct = mainWork[0].pct
	}
	if pct <= 0 {
		return nil
	}

	specs := make([]setSpec, sup.Sets)
	for i := range specs {
		specs[i] = setSpec{pct: pct, reps: sup.Reps}
	}
	return specs
}

func buildSet(kind SetKind, spec setSpec, tm float64, s Settings) Set {
	weight := RoundWeight(tm*spec.pct, s)
	plates, remainder := PlatesFor(weight, s)
	return Set{
		Kind:       kind,
		Weight:     weight,
		TargetReps: spec.reps,
		MaxReps:    spec.maxReps,
		IsAMRAP:    spec.amrap,
		Pct:        spec.pct,
		Plates:     plates,
		Remainder:  remainder,
	}
}

// RoundWeight rounds a prescribed weight to the nearest loadable increment,
// never going below the weight of the empty bar.
func RoundWeight(w float64, s Settings) float64 {
	inc := s.Rounding
	if inc <= 0 {
		inc = 5
	}
	rounded := math.Round(w/inc) * inc
	if rounded < s.BarWeight {
		return s.BarWeight
	}
	return rounded
}

// TMIncrementFor returns one step of training max progression for a lift.
// Nothing in this package applies it; the user decides.
func TMIncrementFor(cfg Config, isUpper bool) float64 {
	if isUpper {
		return cfg.TMIncrementUpper
	}
	return cfg.TMIncrementLower
}
