package program

import (
	"fmt"
	"math"
)

// setSpec is one entry in a percentage table: a fraction of the training max,
// a rep target, and whether the set is taken to a rep max.
type setSpec struct {
	pct   float64
	reps  int
	amrap bool
}

// Percentage tables, indexed by week within the cycle. These are the numbers
// from the book and should not be edited casually; the tests pin them.
var (
	fivesPro = [][]setSpec{
		{{0.65, 5, false}, {0.75, 5, false}, {0.85, 5, false}},
		{{0.70, 5, false}, {0.80, 5, false}, {0.90, 5, false}},
		{{0.75, 5, false}, {0.85, 5, false}, {0.95, 5, false}},
	}

	fiveThreeOnePR = [][]setSpec{
		{{0.65, 5, false}, {0.75, 5, false}, {0.85, 5, true}},
		{{0.70, 3, false}, {0.80, 3, false}, {0.90, 3, true}},
		{{0.75, 5, false}, {0.85, 3, false}, {0.95, 1, true}},
	}

	deloadWeek = [][]setSpec{
		{{0.40, 5, false}, {0.50, 5, false}, {0.60, 5, false}},
	}

	warmupSets = []setSpec{{0.40, 5, false}, {0.50, 5, false}, {0.60, 3, false}}
)

func mainTable(s MainScheme) ([][]setSpec, error) {
	switch s {
	case SchemeFivesPro:
		return fivesPro, nil
	case Scheme531PR:
		return fiveThreeOnePR, nil
	case SchemeDeload:
		return deloadWeek, nil
	default:
		return nil, fmt.Errorf("program: unknown main scheme %q", s)
	}
}

// GenerateWorkout expands a program into the concrete sets for one training
// day: cycleIdx selects the cycle within the block, week is 1-indexed within
// that cycle, and tm is the training max for the lift being trained.
//
// The caller is responsible for supplying the training max in force at the
// time the workout is generated. Weights are absolute, so a later change to
// the training max does not rewrite completed history.
func GenerateWorkout(cfg Config, cycleIdx, week int, tm float64, s Settings) (Workout, error) {
	if cycleIdx < 0 || cycleIdx >= len(cfg.Cycles) {
		return Workout{}, fmt.Errorf("program: cycle index %d out of range (block has %d cycles)", cycleIdx, len(cfg.Cycles))
	}
	cycle := cfg.Cycles[cycleIdx]

	if week < 1 || week > cycle.Weeks {
		return Workout{}, fmt.Errorf("program: week %d out of range for cycle %q (%d weeks)", week, cycle.Label, cycle.Weeks)
	}
	if tm <= 0 {
		return Workout{}, fmt.Errorf("program: training max must be positive, got %v", tm)
	}

	table, err := mainTable(cycle.Main)
	if err != nil {
		return Workout{}, err
	}
	if week > len(table) {
		return Workout{}, fmt.Errorf("program: scheme %q has no week %d", cycle.Main, week)
	}
	mainWork := table[week-1]

	out := Workout{
		CycleLabel: cycle.Label,
		Phase:      cycle.Phase,
		Week:       week,
	}

	// The deload week's working sets are already 40/50/60, so a separate
	// warmup would just repeat them.
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
// main set of whichever week is being run.
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

// TMIncrementFor returns the increase suggested for a lift at the end of a
// cycle. Nothing in this package applies it; the user decides.
func TMIncrementFor(cfg Config, isUpper bool) float64 {
	if isUpper {
		return cfg.TMIncrementUpper
	}
	return cfg.TMIncrementLower
}
