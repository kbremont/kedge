// Package program contains the 5/3/1 training logic: expanding a program
// configuration into the concrete sets performed on a given training day.
//
// Everything in this package is pure. No database, no HTTP, no clock. That
// keeps the part of the app most likely to harbour a bug -- percentages,
// rounding, and which rounds carry an AMRAP set -- exhaustively testable.
package program

// MainScheme selects the set/rep pattern used for the main work of a cycle.
type MainScheme string

const (
	// SchemeFivesPro is 5's PRO: the standard 5/3/1 percentages, but every
	// set is a flat 5 reps and nothing is taken to a rep max.
	SchemeFivesPro MainScheme = "5s_pro"

	// Scheme531PR is textbook 5/3/1, where the final set of each round is
	// taken for as many reps as possible.
	Scheme531PR MainScheme = "531_pr"

	// SchemeDeload is the original book's light round: 40/50/60 for 5s, no
	// AMRAP. The default program no longer uses it, but a stored config may.
	SchemeDeload MainScheme = "deload"

	// SchemeTMTest is the 5/3/1 Forever 7th week training max test: work up
	// to the training max itself and see whether it moves for 3-5 reps.
	SchemeTMTest MainScheme = "tm_test"

	// SchemeSeventhDeload is the 5/3/1 Forever 7th week deload: a few easy
	// sets that still touch the training max, so the bar does not feel
	// foreign when the next cycle starts.
	SchemeSeventhDeload MainScheme = "7th_deload"
)

// Phases group cycles by purpose. Workouts snapshot the phase, so history
// queries match on these values rather than on cycle positions or labels.
const (
	PhaseLeader = "leader"
	PhaseTMTest = "tm_test"
	PhaseAnchor = "anchor"
	PhaseDeload = "deload"
)

// SuppKind selects the supplemental work that follows the main sets.
type SuppKind string

const (
	SuppNone SuppKind = "none"

	// SuppBBB is Boring But Big: high-volume sets at a fixed percentage of
	// the training max, performed on the same lift as the main work.
	SuppBBB SuppKind = "bbb"

	// SuppFSL is First Set Last: repeats the round's opening main-set
	// percentage, so the load tracks the round rather than being fixed.
	SuppFSL SuppKind = "fsl"
)

// SetKind categorises a set for display and for querying history.
type SetKind string

const (
	KindWarmup SetKind = "warmup"
	KindMain   SetKind = "main"
	KindSupp   SetKind = "supplemental"
)

// Supplemental describes the volume work attached to a cycle.
type Supplemental struct {
	Kind SuppKind
	Sets int
	Reps int

	// Pct is the fraction of the training max to load. For SuppFSL it is
	// left at zero, meaning "reuse the first main set's percentage".
	Pct float64
}

// CycleSpec is one cycle within a block: a run of rounds sharing a main
// scheme and supplemental.
type CycleSpec struct {
	Label  string
	Phase  string // one of the Phase constants
	Main   MainScheme
	Supp   Supplemental
	Rounds int

	// Optional cycles are offered rather than assumed: the user may skip one
	// before its first session, and the block carries on without it.
	Optional bool
}

// Config is a complete program. The sequence of cycles repeats indefinitely;
// one pass through it is a block.
type Config struct {
	Name   string
	Cycles []CycleSpec

	// TMIncrementUpper and TMIncrementLower are the step sizes used when
	// recommending a training max at the end of a block. Nothing is ever
	// applied automatically.
	TMIncrementUpper float64
	TMIncrementLower float64
}

// Settings holds the loading preferences needed to turn a percentage into a
// weight that can actually be put on a bar.
type Settings struct {
	Unit      string    // "lb" for now; the column exists so kg can be added later
	Rounding  float64   // round prescribed weights to this increment
	BarWeight float64   // empty bar, the floor for any prescription
	Plates    []float64 // plate denominations available per side, descending
	Warmups   bool      // include 40/50/60 warmup sets before main work
}

// Set is one prescribed set.
type Set struct {
	Index      int
	Kind       SetKind
	Weight     float64
	TargetReps int
	MaxReps    int // top of a rep range such as 3-5, or 0 for a fixed count
	IsAMRAP    bool
	Pct        float64 // of training max, for display
	Plates     []PlatePair
	Remainder  float64 // weight that could not be made from available plates
}

// Workout is everything prescribed for a single training day.
type Workout struct {
	CycleLabel string
	Phase      string
	Round      int // 1-indexed within the cycle
	Sets       []Set
}

// DefaultSettings reflects a standard US gym: pounds, a 45 lb bar, and the
// usual plate selection.
func DefaultSettings() Settings {
	return Settings{
		Unit:      "lb",
		Rounding:  5,
		BarWeight: 45,
		Plates:    []float64{45, 35, 25, 10, 5, 2.5},
		Warmups:   true,
	}
}

// LeaderAnchorBBB is the default program: two leader cycles of 5's PRO with
// Boring But Big, a 7th week training max test, an anchor cycle of 5/3/1 with
// a PR set and First Set Last, then an optional 7th week deload. Eleven rounds,
// then it repeats.
func LeaderAnchorBBB() Config {
	return Config{
		Name: "5's PRO + BBB / 5-3-1 + FSL",
		Cycles: []CycleSpec{
			{
				Label:  "Leader 1",
				Phase:  PhaseLeader,
				Main:   SchemeFivesPro,
				Supp:   Supplemental{Kind: SuppBBB, Sets: 5, Reps: 10, Pct: 0.50},
				Rounds: 3,
			},
			{
				Label:  "Leader 2",
				Phase:  PhaseLeader,
				Main:   SchemeFivesPro,
				Supp:   Supplemental{Kind: SuppBBB, Sets: 5, Reps: 10, Pct: 0.60},
				Rounds: 3,
			},
			{
				Label:  "7th Week TM Test",
				Phase:  PhaseTMTest,
				Main:   SchemeTMTest,
				Supp:   Supplemental{Kind: SuppNone},
				Rounds: 1,
			},
			{
				Label:  "Anchor",
				Phase:  PhaseAnchor,
				Main:   Scheme531PR,
				Supp:   Supplemental{Kind: SuppFSL, Sets: 5, Reps: 5},
				Rounds: 3,
			},
			{
				Label:    "7th Week Deload",
				Phase:    PhaseDeload,
				Main:     SchemeSeventhDeload,
				Supp:     Supplemental{Kind: SuppNone},
				Rounds:   1,
				Optional: true,
			},
		},
		TMIncrementUpper: 5,
		TMIncrementLower: 10,
	}
}

// WithBBBPct returns a copy of the config in which every Boring But Big cycle is
// loaded at pct instead of the percentage the template prescribes. A pct of zero
// or less returns the config unchanged, which is what "use the program's own
// value" means at the call site. FSL, TM test, and deload cycles are untouched.
func (c Config) WithBBBPct(pct float64) Config {
	if pct <= 0 {
		return c
	}
	cycles := make([]CycleSpec, len(c.Cycles))
	copy(cycles, c.Cycles)
	for i, cy := range cycles {
		if cy.Supp.Kind == SuppBBB {
			cy.Supp.Pct = pct
			cycles[i] = cy
		}
	}
	c.Cycles = cycles
	return c
}

// RoundsPerBlock is the total number of training rounds in one pass through the
// program.
func (c Config) RoundsPerBlock() int {
	total := 0
	for _, cy := range c.Cycles {
		total += cy.Rounds
	}
	return total
}
