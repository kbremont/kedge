package program

import (
	"math"
	"testing"
)

const eps = 1e-9

func nearly(a, b float64) bool { return math.Abs(a-b) < eps }

func setsOfKind(w Workout, k SetKind) []Set {
	var out []Set
	for _, s := range w.Sets {
		if s.Kind == k {
			out = append(out, s)
		}
	}
	return out
}

// testSettings uses a 45 lb bar with no warmups, so that tests targeting main
// and supplemental work are not cluttered by them.
func testSettings() Settings {
	s := DefaultSettings()
	s.Warmups = false
	return s
}

type expectedSet struct {
	weight float64
	reps   int
	amrap  bool
}

// TestBlockMainWork walks every training day in the ten-round block with a
// training max of 300 and pins the prescribed main work. 300 is chosen so
// that every percentage lands on a clean multiple of five and the expected
// values can be verified by hand.
func TestBlockMainWork(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := testSettings()

	cases := []struct {
		name     string
		cycleIdx int
		round    int
		want     []expectedSet
	}{
		// Leader cycles run 5's PRO: book percentages, all fives, no AMRAP.
		{"leader1 r1", 0, 1, []expectedSet{{195, 5, false}, {225, 5, false}, {255, 5, false}}},
		{"leader1 r2", 0, 2, []expectedSet{{210, 5, false}, {240, 5, false}, {270, 5, false}}},
		{"leader1 r3", 0, 3, []expectedSet{{225, 5, false}, {255, 5, false}, {285, 5, false}}},
		{"leader2 r1", 1, 1, []expectedSet{{195, 5, false}, {225, 5, false}, {255, 5, false}}},
		{"leader2 r2", 1, 2, []expectedSet{{210, 5, false}, {240, 5, false}, {270, 5, false}}},
		{"leader2 r3", 1, 3, []expectedSet{{225, 5, false}, {255, 5, false}, {285, 5, false}}},

		{"deload", 2, 1, []expectedSet{{120, 5, false}, {150, 5, false}, {180, 5, false}}},

		// The anchor restores the rep targets and the PR set.
		{"anchor r1", 3, 1, []expectedSet{{195, 5, false}, {225, 5, false}, {255, 5, true}}},
		{"anchor r2", 3, 2, []expectedSet{{210, 3, false}, {240, 3, false}, {270, 3, true}}},
		{"anchor r3", 3, 3, []expectedSet{{225, 5, false}, {255, 3, false}, {285, 1, true}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := GenerateWorkout(cfg, tc.cycleIdx, tc.round, 300, s)
			if err != nil {
				t.Fatalf("GenerateWorkout: %v", err)
			}
			main := setsOfKind(w, KindMain)
			if len(main) != len(tc.want) {
				t.Fatalf("got %d main sets, want %d", len(main), len(tc.want))
			}
			for i, want := range tc.want {
				got := main[i]
				if !nearly(got.Weight, want.weight) {
					t.Errorf("set %d: weight = %v, want %v", i, got.Weight, want.weight)
				}
				if got.TargetReps != want.reps {
					t.Errorf("set %d: reps = %d, want %d", i, got.TargetReps, want.reps)
				}
				if got.IsAMRAP != want.amrap {
					t.Errorf("set %d: amrap = %v, want %v", i, got.IsAMRAP, want.amrap)
				}
			}
		})
	}
}

// TestOnlyAnchorHasAMRAP guards the defining property of this program: 5's
// PRO means no set is taken to failure until the anchor.
func TestOnlyAnchorHasAMRAP(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := testSettings()

	for ci, cycle := range cfg.Cycles {
		for round := 1; round <= cycle.Rounds; round++ {
			w, err := GenerateWorkout(cfg, ci, round, 300, s)
			if err != nil {
				t.Fatalf("cycle %d round %d: %v", ci, round, err)
			}
			var amraps int
			for _, set := range w.Sets {
				if set.IsAMRAP {
					amraps++
				}
			}
			want := 0
			if cycle.Phase == "anchor" {
				want = 1
			}
			if amraps != want {
				t.Errorf("%s round %d: got %d AMRAP sets, want %d", cycle.Label, round, amraps, want)
			}
		}
	}
}

// TestBBBIsFixedPercentage checks that Boring But Big holds its load across
// the cycle: 50% in the first leader, 60% in the second, five sets of ten.
func TestBBBIsFixedPercentage(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := testSettings()

	cases := []struct {
		cycleIdx   int
		wantWeight float64
	}{
		{0, 150}, // 50% of 300
		{1, 180}, // 60% of 300
	}

	for _, tc := range cases {
		for round := 1; round <= 3; round++ {
			w, err := GenerateWorkout(cfg, tc.cycleIdx, round, 300, s)
			if err != nil {
				t.Fatalf("cycle %d round %d: %v", tc.cycleIdx, round, err)
			}
			supp := setsOfKind(w, KindSupp)
			if len(supp) != 5 {
				t.Fatalf("cycle %d round %d: got %d supplemental sets, want 5", tc.cycleIdx, round, len(supp))
			}
			for i, set := range supp {
				if !nearly(set.Weight, tc.wantWeight) {
					t.Errorf("cycle %d round %d set %d: weight = %v, want %v", tc.cycleIdx, round, i, set.Weight, tc.wantWeight)
				}
				if set.TargetReps != 10 {
					t.Errorf("cycle %d round %d set %d: reps = %d, want 10", tc.cycleIdx, round, i, set.TargetReps)
				}
				if set.IsAMRAP {
					t.Errorf("cycle %d round %d set %d: BBB should never be an AMRAP", tc.cycleIdx, round, i)
				}
			}
		}
	}
}

// TestWithBBBPct checks that the override replaces BBB's load on both leader
// cycles, leaves FSL and deload alone, and does not mutate the receiver.
func TestWithBBBPct(t *testing.T) {
	cfg := LeaderAnchorBBB()
	overridden := cfg.WithBBBPct(0.45)

	if overridden.Cycles[0].Supp.Pct != 0.45 {
		t.Errorf("leader 1 BBB pct = %v, want 0.45", overridden.Cycles[0].Supp.Pct)
	}
	if overridden.Cycles[1].Supp.Pct != 0.45 {
		t.Errorf("leader 2 BBB pct = %v, want 0.45", overridden.Cycles[1].Supp.Pct)
	}
	if overridden.Cycles[3].Supp.Kind != SuppFSL || overridden.Cycles[3].Supp.Pct != 0 {
		t.Errorf("anchor FSL cycle changed: %+v", overridden.Cycles[3].Supp)
	}

	// The receiver's backing array must be untouched.
	if cfg.Cycles[0].Supp.Pct != 0.50 {
		t.Errorf("receiver mutated: leader 1 BBB pct = %v, want 0.50", cfg.Cycles[0].Supp.Pct)
	}
	if cfg.Cycles[1].Supp.Pct != 0.60 {
		t.Errorf("receiver mutated: leader 2 BBB pct = %v, want 0.60", cfg.Cycles[1].Supp.Pct)
	}

	// A non-positive pct means "use the program's own value".
	unchanged := cfg.WithBBBPct(0)
	if unchanged.Cycles[0].Supp.Pct != 0.50 {
		t.Errorf("WithBBBPct(0) changed leader 1 pct to %v, want unchanged 0.50", unchanged.Cycles[0].Supp.Pct)
	}
	if cfg.WithBBBPct(-1).Cycles[0].Supp.Pct != 0.50 {
		t.Errorf("WithBBBPct(-1) should leave the config unchanged")
	}
}

// TestGenerateWorkoutWithBBBOverride checks that an overridden config produces
// BBB sets at the new load, without disturbing main or warmup sets.
func TestGenerateWorkoutWithBBBOverride(t *testing.T) {
	cfg := LeaderAnchorBBB().WithBBBPct(0.40)
	s := testSettings()

	w, err := GenerateWorkout(cfg, 0, 1, 300, s)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	supp := setsOfKind(w, KindSupp)
	if len(supp) != 5 {
		t.Fatalf("got %d supplemental sets, want 5", len(supp))
	}
	for i, set := range supp {
		if !nearly(set.Weight, 120) { // 40% of 300
			t.Errorf("set %d: weight = %v, want 120", i, set.Weight)
		}
	}
	main := setsOfKind(w, KindMain)
	if !nearly(main[0].Weight, 195) || !nearly(main[1].Weight, 225) || !nearly(main[2].Weight, 255) {
		t.Errorf("main sets changed by BBB override: %+v", main)
	}
}

// TestFSLTracksRoundOpener checks that First Set Last follows the round's
// opening percentage rather than sitting at a fixed load like BBB.
func TestFSLTracksRoundOpener(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := testSettings()

	want := map[int]float64{1: 195, 2: 210, 3: 225} // 65%, 70%, 75% of 300

	for round, wantWeight := range want {
		w, err := GenerateWorkout(cfg, 3, round, 300, s)
		if err != nil {
			t.Fatalf("anchor round %d: %v", round, err)
		}
		supp := setsOfKind(w, KindSupp)
		if len(supp) != 5 {
			t.Fatalf("anchor round %d: got %d FSL sets, want 5", round, len(supp))
		}
		main := setsOfKind(w, KindMain)
		for i, set := range supp {
			if !nearly(set.Weight, wantWeight) {
				t.Errorf("anchor round %d set %d: weight = %v, want %v", round, i, set.Weight, wantWeight)
			}
			if !nearly(set.Weight, main[0].Weight) {
				t.Errorf("anchor round %d set %d: FSL weight %v does not match first main set %v", round, i, set.Weight, main[0].Weight)
			}
			if set.TargetReps != 5 {
				t.Errorf("anchor round %d set %d: reps = %d, want 5", round, i, set.TargetReps)
			}
		}
	}
}

// TestDeloadHasNoWarmupsOrSupplemental keeps the light round light: its working
// sets are already 40/50/60, so warmups would duplicate them.
func TestDeloadHasNoWarmupsOrSupplemental(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings() // warmups deliberately enabled

	w, err := GenerateWorkout(cfg, 2, 1, 300, s)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	if n := len(setsOfKind(w, KindWarmup)); n != 0 {
		t.Errorf("got %d warmup sets on the deload, want 0", n)
	}
	if n := len(setsOfKind(w, KindSupp)); n != 0 {
		t.Errorf("got %d supplemental sets on the deload, want 0", n)
	}
	if n := len(w.Sets); n != 3 {
		t.Errorf("got %d total sets on the deload, want 3", n)
	}
}

// TestWarmupSets pins the 40/50/60 ramp that precedes main work.
func TestWarmupSets(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings()

	w, err := GenerateWorkout(cfg, 0, 1, 300, s)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	want := []expectedSet{{120, 5, false}, {150, 5, false}, {180, 3, false}}
	got := setsOfKind(w, KindWarmup)
	if len(got) != len(want) {
		t.Fatalf("got %d warmup sets, want %d", len(got), len(want))
	}
	for i, wantSet := range want {
		if !nearly(got[i].Weight, wantSet.weight) || got[i].TargetReps != wantSet.reps {
			t.Errorf("warmup %d: got %v x %d, want %v x %d", i, got[i].Weight, got[i].TargetReps, wantSet.weight, wantSet.reps)
		}
	}
}

// TestSetOrderAndIndexing checks that a session reads top to bottom in the
// order it is performed, with contiguous indices for stable URLs.
func TestSetOrderAndIndexing(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings()

	w, err := GenerateWorkout(cfg, 0, 1, 300, s)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	if len(w.Sets) != 11 { // 3 warmup + 3 main + 5 BBB
		t.Fatalf("got %d sets, want 11", len(w.Sets))
	}
	wantOrder := []SetKind{
		KindWarmup, KindWarmup, KindWarmup,
		KindMain, KindMain, KindMain,
		KindSupp, KindSupp, KindSupp, KindSupp, KindSupp,
	}
	for i, set := range w.Sets {
		if set.Kind != wantOrder[i] {
			t.Errorf("set %d: kind = %q, want %q", i, set.Kind, wantOrder[i])
		}
		if set.Index != i {
			t.Errorf("set %d: index = %d, want %d", i, set.Index, i)
		}
	}
}

func TestRoundWeight(t *testing.T) {
	s := DefaultSettings()

	cases := []struct {
		in, want float64
	}{
		{195, 195},
		{87.75, 90},   // 65% of a 135 training max
		{101.25, 100}, // 75% of 135
		{114.75, 115}, // 85% of 135
		{67.5, 70},    // exact half rounds away from zero
		{62.5, 65},    //
		{26, 45},      // below the bar, clamped up
		{0.5, 45},     //
		{47.4, 45},    // rounds to 45, which is the bar
	}
	for _, tc := range cases {
		if got := RoundWeight(tc.in, s); !nearly(got, tc.want) {
			t.Errorf("RoundWeight(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestLightTrainingMaxClampsToBar covers a beginner pressing less than an
// empty bar at 40%: the prescription can never drop below the bar itself.
func TestLightTrainingMaxClampsToBar(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings()

	w, err := GenerateWorkout(cfg, 0, 1, 65, s)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	for _, set := range w.Sets {
		if set.Weight < s.BarWeight {
			t.Errorf("%s set at %v is below the bar weight %v", set.Kind, set.Weight, s.BarWeight)
		}
	}
}

func TestGenerateWorkoutErrors(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings()

	cases := []struct {
		name     string
		cycleIdx int
		round    int
		tm       float64
	}{
		{"negative cycle", -1, 1, 300},
		{"cycle past end of block", 4, 1, 300},
		{"round zero", 0, 0, 300},
		{"round past end of cycle", 0, 4, 300},
		{"deload has only one round", 2, 2, 300},
		{"zero training max", 0, 1, 0},
		{"negative training max", 0, 1, -100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := GenerateWorkout(cfg, tc.cycleIdx, tc.round, tc.tm, s); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestRoundsPerBlock(t *testing.T) {
	if got := LeaderAnchorBBB().RoundsPerBlock(); got != 10 {
		t.Errorf("RoundsPerBlock() = %d, want 10", got)
	}
}

func TestTMIncrementFor(t *testing.T) {
	cfg := LeaderAnchorBBB()
	if got := TMIncrementFor(cfg, true); !nearly(got, 5) {
		t.Errorf("upper increment = %v, want 5", got)
	}
	if got := TMIncrementFor(cfg, false); !nearly(got, 10) {
		t.Errorf("lower increment = %v, want 10", got)
	}
}
