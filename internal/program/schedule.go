package program

import "fmt"

// Position identifies a single training day within a block.
type Position struct {
	CycleIndex int
	Week       int // 1-indexed within the cycle
	Day        int // 1-indexed within the week, matching a lift's day order
}

func (p Position) String() string {
	return fmt.Sprintf("c%d/w%d/d%d", p.CycleIndex, p.Week, p.Day)
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
func (c Config) Schedule(daysPerWeek int) []Position {
	if daysPerWeek < 1 {
		return nil
	}
	var out []Position
	for ci, cycle := range c.Cycles {
		for week := 1; week <= cycle.Weeks; week++ {
			for day := 1; day <= daysPerWeek; day++ {
				out = append(out, Position{CycleIndex: ci, Week: week, Day: day})
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
// for example "Leader 2 - Week 3".
func (c Config) Describe(p Position) string {
	cycle, err := c.CycleAt(p.CycleIndex)
	if err != nil {
		return p.String()
	}
	if cycle.Weeks == 1 {
		return cycle.Label
	}
	return fmt.Sprintf("%s - Week %d", cycle.Label, p.Week)
}
