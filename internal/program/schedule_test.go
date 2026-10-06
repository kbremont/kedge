package program

import "testing"

func TestScheduleCoversBlock(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.Schedule(4)

	if got, want := len(sched), 11*4; got != want {
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

	if first, want := sched[0], (Position{CycleIndex: 0, Round: 1, Day: 1}); first != want {
		t.Errorf("first day = %v, want %v", first, want)
	}
	if last, want := sched[len(sched)-1], (Position{CycleIndex: 4, Round: 1, Day: 4}); last != want {
		t.Errorf("last day = %v, want %v", last, want)
	}
}

// TestScheduleOrdersSeventhWeeks pins the shape of the block: six leader
// rounds, the TM test, three anchor rounds, then the deload.
func TestScheduleOrdersSeventhWeeks(t *testing.T) {
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
	if counts[PhaseLeader] != 24 { // 6 rounds x 4 days
		t.Errorf("leader days = %d, want 24", counts[PhaseLeader])
	}
	if counts[PhaseTMTest] != 4 {
		t.Errorf("TM test days = %d, want 4", counts[PhaseTMTest])
	}
	if counts[PhaseAnchor] != 12 {
		t.Errorf("anchor days = %d, want 12", counts[PhaseAnchor])
	}
	if counts[PhaseDeload] != 4 {
		t.Errorf("deload days = %d, want 4", counts[PhaseDeload])
	}

	// The TM test sits between the leaders and the anchor; the deload closes
	// the block.
	if phases[23] != PhaseLeader || phases[24] != PhaseTMTest || phases[28] != PhaseAnchor {
		t.Errorf("phase boundaries wrong: ...%s | %s ... %s...", phases[23], phases[24], phases[28])
	}
	if phases[39] != PhaseAnchor || phases[40] != PhaseDeload {
		t.Errorf("deload boundary wrong: ...%s | %s...", phases[39], phases[40])
	}
}

// TestBlockScheduleSkipsCycles covers passing on the optional deload: it drops
// out of the block, which then ends with the anchor.
func TestBlockScheduleSkipsCycles(t *testing.T) {
	cfg := LeaderAnchorBBB()

	sched := cfg.BlockSchedule(4, 0, map[int]bool{4: true})
	if got, want := len(sched), 10*4; got != want {
		t.Fatalf("schedule without the deload has %d days, want %d", got, want)
	}
	if last, want := sched[len(sched)-1], (Position{CycleIndex: 3, Round: 3, Day: 4}); last != want {
		t.Errorf("last day = %v, want %v", last, want)
	}
	for _, p := range sched {
		if p.CycleIndex == 4 {
			t.Fatalf("skipped cycle still scheduled at %v", p)
		}
	}

	// With everything before it done, skipping the deload ends the block.
	done := map[Position]bool{}
	for _, p := range sched {
		done[p] = true
	}
	if _, ok := NextPosition(sched, done); ok {
		t.Error("block should be over once the deload is skipped")
	}
}

// TestOnlyTheDeloadIsOptional pins which cycle may be skipped: the TM test
// feeds the end-of-block recommendation, so it is never optional.
func TestOnlyTheDeloadIsOptional(t *testing.T) {
	for i, c := range LeaderAnchorBBB().Cycles {
		if want := c.Phase == PhaseDeload; c.Optional != want {
			t.Errorf("cycle %d (%s): optional = %v, want %v", i, c.Label, c.Optional, want)
		}
	}
}

func TestNextPosition(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.Schedule(4)

	// Nothing done: start at the beginning.
	got, ok := NextPosition(sched, map[Position]bool{})
	if !ok || got != (Position{0, 1, 1}) {
		t.Errorf("empty block: got %v (ok=%v), want c0/r1/d1", got, ok)
	}

	// Three days done: pick up on the fourth.
	done := map[Position]bool{{0, 1, 1}: true, {0, 1, 2}: true, {0, 1, 3}: true}
	got, ok = NextPosition(sched, done)
	if !ok || got != (Position{0, 1, 4}) {
		t.Errorf("partial round: got %v (ok=%v), want c0/r1/d4", got, ok)
	}

	// A skipped day is picked up before later work, so the block cannot be
	// silently left with a hole in it.
	done = map[Position]bool{{0, 1, 1}: true, {0, 1, 3}: true, {0, 1, 4}: true}
	got, ok = NextPosition(sched, done)
	if !ok || got != (Position{0, 1, 2}) {
		t.Errorf("skipped day: got %v (ok=%v), want c0/r1/d2", got, ok)
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
		{Position{0, 1, 1}, "Leader 1 - Round 1"},
		{Position{1, 3, 2}, "Leader 2 - Round 3"},
		{Position{2, 1, 1}, "7th Week TM Test"}, // a one-round cycle needs no round number
		{Position{3, 2, 4}, "Anchor - Round 2"},
		{Position{4, 1, 3}, "7th Week Deload"},
	}
	for _, tc := range cases {
		if got := cfg.Describe(tc.pos); got != tc.want {
			t.Errorf("Describe(%v) = %q, want %q", tc.pos, got, tc.want)
		}
	}
}

func TestCycleAtRejectsOutOfRange(t *testing.T) {
	cfg := LeaderAnchorBBB()
	for _, idx := range []int{-1, 5, 99} {
		if _, err := cfg.CycleAt(idx); err == nil {
			t.Errorf("CycleAt(%d) should have failed", idx)
		}
	}
}

// TestScheduleFromTrimsEarlierCycles covers joining a block partway through:
// the cycles before the start are not part of the schedule at all, so nothing
// has to be fabricated to represent them.
func TestScheduleFromTrimsEarlierCycles(t *testing.T) {
	cfg := LeaderAnchorBBB()

	sched := cfg.ScheduleFrom(4, 3) // start at the anchor
	if got, want := len(sched), 3*4+4; got != want {
		t.Fatalf("schedule from the anchor has %d days, want %d", got, want)
	}
	if first, want := sched[0], (Position{CycleIndex: 3, Round: 1, Day: 1}); first != want {
		t.Errorf("first day = %v, want %v", first, want)
	}
	if last, want := sched[len(sched)-1], (Position{CycleIndex: 4, Round: 1, Day: 4}); last != want {
		t.Errorf("last day = %v, want %v", last, want)
	}
	for _, p := range sched {
		if p.CycleIndex < 3 {
			t.Errorf("position %v is before the anchor", p)
		}
	}
}

func TestScheduleFromStartingAtZeroIsTheWholeBlock(t *testing.T) {
	cfg := LeaderAnchorBBB()

	full := cfg.Schedule(4)
	from := cfg.ScheduleFrom(4, 0)
	if len(full) != len(from) {
		t.Fatalf("ScheduleFrom(4, 0) has %d days, Schedule(4) has %d", len(from), len(full))
	}
	for i := range full {
		if full[i] != from[i] {
			t.Fatalf("day %d: ScheduleFrom gave %v, Schedule gave %v", i, from[i], full[i])
		}
	}
}

// TestScheduleFromTMTestIsOneRound guards the single-round cycle, which is the
// one most likely to be off by a round.
func TestScheduleFromTMTestIsOneRound(t *testing.T) {
	cfg := LeaderAnchorBBB()

	sched := cfg.ScheduleFrom(4, 2) // TM test, then the anchor and the deload
	if got, want := len(sched), 4+(3*4)+4; got != want {
		t.Fatalf("schedule from the TM test has %d days, want %d", got, want)
	}
	if first, want := sched[0], (Position{CycleIndex: 2, Round: 1, Day: 1}); first != want {
		t.Errorf("first day = %v, want %v", first, want)
	}
	// The TM test is one round, so the fifth day must already be the anchor.
	if fifth, want := sched[4], (Position{CycleIndex: 3, Round: 1, Day: 1}); fifth != want {
		t.Errorf("day after the TM test = %v, want %v", fifth, want)
	}
}

// TestNextPositionOnTrimmedSchedule is the onboarding case end to end: the
// days recorded during setup are marked done, and the app resumes at the
// first that is not.
func TestNextPositionOnTrimmedSchedule(t *testing.T) {
	cfg := LeaderAnchorBBB()
	sched := cfg.ScheduleFrom(4, 3)

	// Anchor round 1 complete, plus press and deadlift of round 2.
	done := map[Position]bool{
		{3, 1, 1}: true, {3, 1, 2}: true, {3, 1, 3}: true, {3, 1, 4}: true,
		{3, 2, 1}: true, {3, 2, 2}: true,
	}
	got, ok := NextPosition(sched, done)
	if !ok {
		t.Fatal("expected a next position")
	}
	if want := (Position{CycleIndex: 3, Round: 2, Day: 3}); got != want {
		t.Errorf("next = %v, want %v", got, want)
	}
}
