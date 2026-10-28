package program

import "fmt"

// Position identifies a single training day within a block.
type Position struct {
	CycleIndex int
	Round      int // 1-indexed within the cycle
	Day        int // 1-indexed within the round, matching a lift's day order
}

func (p Position) String() string {
	return fmt.Sprintf("c%d/r%d/d%d", p.CycleIndex, p.Round, p.Day)
}

// CycleAt returns the cycle at idx.
func (c Config) CycleAt(idx int) (CycleSpec, error) {
	if idx < 0 || idx >= len(c.Cycles) {
		return CycleSpec{}, fmt.Errorf("program: cycle index %d out of range (block has %d cycles)", idx, len(c.Cycles))
	}
	return c.Cycles[idx], nil
}

// Schedule lays out every training day in one block, in the order they are
// performed.
func (c Config) Schedule(daysPerRound int) []Position {
	return c.ScheduleFrom(daysPerRound, 0)
}

// ScheduleFrom lays out the training days in a block from startCycleIndex
// onward, for a block joined partway through.
//
// Cycles before the start are not part of this block's schedule at all, which
// is what lets someone begin mid-programme without the app inventing completed
// sessions to represent the work it was never told about.
func (c Config) ScheduleFrom(daysPerRound, startCycleIndex int) []Position {
	return c.BlockSchedule(daysPerRound, startCycleIndex, nil)
}

// BlockSchedule lays out a block that starts at startCycleIndex and leaves out
// the cycles in skipped, which is how an optional cycle the user passed on
// drops out of the block. Skipping the last remaining cycle leaves nothing
// scheduled, so the block is over and the next one begins.
func (c Config) BlockSchedule(daysPerRound, startCycleIndex int, skipped map[int]bool) []Position {
	if daysPerRound < 1 {
		return nil
	}
	if startCycleIndex < 0 {
		startCycleIndex = 0
	}
	var out []Position
	for ci, cycle := range c.Cycles {
		if ci < startCycleIndex || skipped[ci] {
			continue
		}
		for round := 1; round <= cycle.Rounds; round++ {
			for day := 1; day <= daysPerRound; day++ {
				out = append(out, Position{CycleIndex: ci, Round: round, Day: day})
			}
		}
	}
	return out
}

// NextPosition returns the earliest position in the schedule that has not been
// completed. It reports false once every day in the block is done, which is
// the signal to roll over into the next block.
//
// Deriving the current position from completed work, rather than storing a
// cursor, means there is no pointer that can drift out of step with reality.
func NextPosition(schedule []Position, done map[Position]bool) (Position, bool) {
	for _, p := range schedule {
		if !done[p] {
			return p, true
		}
	}
	return Position{}, false
}

// Describe renders a position as the label shown at the top of a session,
// for example "Leader 2 - Round 3".
func (c Config) Describe(p Position) string {
	cycle, err := c.CycleAt(p.CycleIndex)
	if err != nil {
		return p.String()
	}
	if cycle.Rounds == 1 {
		return cycle.Label
	}
	return fmt.Sprintf("%s - Round %d", cycle.Label, p.Round)
}
