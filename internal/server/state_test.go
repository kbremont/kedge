package server

import (
	"testing"

	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/program"
)

// TestRoundOfBlock pins the "round 8 of 10" line on the dashboard. The count is
// absolute within the block, so joining partway through still reports where the
// session sits in the programme rather than where it sits in the schedule that
// was generated.
func TestRoundOfBlock(t *testing.T) {
	cfg := program.LeaderAnchorBBB()

	cases := []struct {
		pos  program.Position
		want int
	}{
		{program.Position{CycleIndex: 0, Round: 1}, 1},
		{program.Position{CycleIndex: 0, Round: 3}, 3},
		{program.Position{CycleIndex: 1, Round: 1}, 4},
		{program.Position{CycleIndex: 1, Round: 3}, 6},
		{program.Position{CycleIndex: 2, Round: 1}, 7},  // the one-round deload
		{program.Position{CycleIndex: 3, Round: 1}, 8},  // anchor opens at round 8
		{program.Position{CycleIndex: 3, Round: 3}, 10}, // and closes the block
	}
	for _, tc := range cases {
		if got := roundOfBlock(cfg, tc.pos); got != tc.want {
			t.Errorf("roundOfBlock(%v) = %d, want %d", tc.pos, got, tc.want)
		}
	}

	if got, want := cfg.RoundsPerBlock(), 10; got != want {
		t.Errorf("RoundsPerBlock() = %d, want %d", got, want)
	}
}

// TestTopMainSet checks the weight shown against each row of the onboarding
// checklist, which is what makes a past session recognisable.
func TestTopMainSet(t *testing.T) {
	cfg := program.LeaderAnchorBBB()
	settings := program.DefaultSettings()

	// Leader round 1 at a 300 training max tops out at 85%, with no PR set.
	leader, err := program.GenerateWorkout(cfg, 0, 1, 300, settings)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	weight, prTarget := topMainSet(leader)
	if weight != 255 {
		t.Errorf("leader top set = %v, want 255", weight)
	}
	if prTarget != 0 {
		t.Errorf("leader PR target = %d, want 0 (5's PRO has no rep-max set)", prTarget)
	}

	// The anchor's third round tops out at 95% and finishes on a single.
	anchor, err := program.GenerateWorkout(cfg, 3, 3, 300, settings)
	if err != nil {
		t.Fatalf("GenerateWorkout: %v", err)
	}
	weight, prTarget = topMainSet(anchor)
	if weight != 285 {
		t.Errorf("anchor top set = %v, want 285", weight)
	}
	if prTarget != 1 {
		t.Errorf("anchor PR target = %d, want 1", prTarget)
	}
}

func TestCycleDetail(t *testing.T) {
	cfg := program.LeaderAnchorBBB()

	cases := []struct {
		idx  int
		want string
	}{
		{0, "3 rounds, 5's PRO + BBB 5x10 @ 50%"},
		{1, "3 rounds, 5's PRO + BBB 5x10 @ 60%"},
		{2, "deload"}, // a single round needs no count
		{3, "3 rounds, 5/3/1 with a PR set + FSL 5x5"},
	}
	for _, tc := range cases {
		cycle, err := cfg.CycleAt(tc.idx)
		if err != nil {
			t.Fatalf("CycleAt(%d): %v", tc.idx, err)
		}
		if got := cycleDetail(cycle); got != tc.want {
			t.Errorf("cycleDetail(%q) = %q, want %q", cycle.Label, got, tc.want)
		}
	}
}

func TestFormFieldNames(t *testing.T) {
	if got, want := doneField(2, "press"), "done_r2_press"; got != want {
		t.Errorf("doneField = %q, want %q", got, want)
	}
	if got, want := repsField(3, "deadlift"), "reps_r3_deadlift"; got != want {
		t.Errorf("repsField = %q, want %q", got, want)
	}
	if got, want := bbbModeField("press"), "bbb_mode_press"; got != want {
		t.Errorf("bbbModeField = %q, want %q", got, want)
	}
	if got, want := bbbValueField("deadlift"), "bbb_value_deadlift"; got != want {
		t.Errorf("bbbValueField = %q, want %q", got, want)
	}
}

// TestBBBPctFor pins how a lift's BBB preference turns into the fraction of
// training max the engine should load.
func TestBBBPctFor(t *testing.T) {
	cases := []struct {
		name string
		lift db.Lift
		tm   float64
		want float64
	}{
		{"program mode", db.Lift{BbbMode: "program", BbbValue: 0.5}, 300, 0},
		{"percent mode", db.Lift{BbbMode: "percent", BbbValue: 0.4}, 300, 0.4},
		{"percent mode zero value", db.Lift{BbbMode: "percent", BbbValue: 0}, 300, 0},
		{"weight mode", db.Lift{BbbMode: "weight", BbbValue: 150}, 300, 0.5},
		{"weight mode zero tm", db.Lift{BbbMode: "weight", BbbValue: 150}, 0, 0},
		{"unknown mode", db.Lift{BbbMode: "bogus", BbbValue: 150}, 300, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := bbbPctFor(tc.lift, tc.tm); got != tc.want {
				t.Errorf("bbbPctFor(%+v, %v) = %v, want %v", tc.lift, tc.tm, got, tc.want)
			}
		})
	}
}

// TestRestSecondsFor pins the rest-interval lookup used when a set is logged.
func TestRestSecondsFor(t *testing.T) {
	raw := db.Setting{RestWarmupSeconds: 30, RestMainSeconds: 240, RestSuppSeconds: 60}

	cases := []struct {
		kind program.SetKind
		want int
	}{
		{program.KindWarmup, 30},
		{program.KindMain, 240},
		{program.KindSupp, 60},
		{program.SetKind("unknown"), 240},
	}
	for _, tc := range cases {
		if got := restSecondsFor(raw, tc.kind); got != tc.want {
			t.Errorf("restSecondsFor(%q) = %d, want %d", tc.kind, got, tc.want)
		}
	}
}
