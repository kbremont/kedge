package program

import "testing"

func TestScheduleCoversBlock(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.Schedule(4)

	if got, want := len(sched), 10*4; got != want {
		t.Fatalf("schedule has %d days, want %d", got, want)
	}

	// The block must run in order and never revisit a day.
	seen := map[Position]bool{}
	for _, p := range sched {
		if seen[p] {
			t.Errorf("duplicate position %v", p)
		}
		seen[p] = true
	}

	if first, want := sched[0], (Position{CycleIndex: 0, Week: 1, Day: 1}); first != want {
		t.Errorf("first day = %v, want %v", first, want)
	}
	if last, want := sched[len(sched)-1], (Position{CycleIndex: 3, Week: 3, Day: 4}); last != want {
		t.Errorf("last day = %v, want %v", last, want)
	}
}

// TestScheduleOrdersDeloadBetweenLeadersAndAnchor pins the shape of the
// block: six leader weeks, one deload, three anchor weeks.
func TestScheduleOrdersDeloadBetweenLeadersAndAnchor(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.Schedule(4)

	phases := make([]string, 0, len(sched))
	for _, p := range sched {
		cycle, err := cfg.CycleAt(p.CycleIndex)
		if err != nil {
			t.Fatalf("CycleAt(%d): %v", p.CycleIndex, err)
		}
		phases = append(phases, cycle.Phase)
	}

	counts := map[string]int{}
	for _, ph := range phases {
		counts[ph]++
	}
	if counts["leader"] != 24 { // 6 weeks x 4 days
		t.Errorf("leader days = %d, want 24", counts["leader"])
	}
	if counts["deload"] != 4 {
		t.Errorf("deload days = %d, want 4", counts["deload"])
	}
	if counts["anchor"] != 12 {
		t.Errorf("anchor days = %d, want 12", counts["anchor"])
	}

	// The deload must sit between the leaders and the anchor.
	if phases[23] != "leader" || phases[24] != "deload" || phases[28] != "anchor" {
		t.Errorf("phase boundaries wrong: ...%s | %s ... %s...", phases[23], phases[24], phases[28])
	}
}

func TestNextPosition(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.Schedule(4)

	// Nothing done: start at the beginning.
	got, ok := NextPosition(sched, map[Position]bool{})
	if !ok || got != (Position{0, 1, 1}) {
		t.Errorf("empty block: got %v (ok=%v), want c0/w1/d1", got, ok)
	}

	// Three days done: pick up on the fourth.
	done := map[Position]bool{{0, 1, 1}: true, {0, 1, 2}: true, {0, 1, 3}: true}
	got, ok = NextPosition(sched, done)
	if !ok || got != (Position{0, 1, 4}) {
		t.Errorf("partial week: got %v (ok=%v), want c0/w1/d4", got, ok)
	}

	// A skipped day is picked up before later work, so the block cannot be
	// silently left with a hole in it.
	done = map[Position]bool{{0, 1, 1}: true, {0, 1, 3}: true, {0, 1, 4}: true}
	got, ok = NextPosition(sched, done)
	if !ok || got != (Position{0, 1, 2}) {
		t.Errorf("skipped day: got %v (ok=%v), want c0/w1/d2", got, ok)
	}

	// Everything done: the block is over.
	all := map[Position]bool{}
	for _, p := range sched {
		all[p] = true
	}
	if _, ok := NextPosition(sched, all); ok {
		t.Error("completed block should report no next position")
	}
}

func TestDescribe(t *testing.T) {
	cfg := LeaderAnchorBBB()

	cases := []struct {
		pos  Position
		want string
	}{
		{Position{0, 1, 1}, "Leader 1 - Week 1"},
		{Position{1, 3, 2}, "Leader 2 - Week 3"},
		{Position{2, 1, 1}, "Deload"}, // a one-week cycle needs no week number
		{Position{3, 2, 4}, "Anchor - Week 2"},
	}
	for _, tc := range cases {
		if got := cfg.Describe(tc.pos); got != tc.want {
			t.Errorf("Describe(%v) = %q, want %q", tc.pos, got, tc.want)
		}
	}
}

func TestCycleAtRejectsOutOfRange(t *testing.T) {
	cfg := LeaderAnchorBBB()
	for _, idx := range []int{-1, 4, 99} {
		if _, err := cfg.CycleAt(idx); err == nil {
			t.Errorf("CycleAt(%d) should have failed", idx)
		}
	}
}
