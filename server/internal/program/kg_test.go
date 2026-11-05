package program

import "testing"

// TestKilogramPlates loads a 2.5 kg step with the 1.25 kg plates the kilogram
// defaults include, and rounds to it.
func TestKilogramPlates(t *testing.T) {
	s := DefaultSettingsFor(UnitKG)
	if got := RoundWeight(0.75*102.5, s); got != 77.5 {
		t.Errorf("RoundWeight(76.875) = %v, want 77.5", got)
	}
	plates, rem := PlatesFor(62.5, s)
	want := []PlatePair{{Weight: 20, Count: 1}, {Weight: 1.25, Count: 1}}
	if rem != 0 || len(plates) != len(want) || plates[0] != want[0] || plates[1] != want[1] {
		t.Errorf("PlatesFor(62.5) = %v, remainder %v; want %v", plates, rem, want)
	}
	// 41.125 a side: 25 and 15 leave 1.125, under the smallest plate. The
	// shortfall is all of it, not rounded to a hundredth.
	if _, rem := PlatesFor(102.25, s); rem != 2.25 {
		t.Errorf("PlatesFor(102.25) remainder = %v, want 2.25", rem)
	}
}

// TestKilogramSteps bumps by the book's kilogram steps: 2.5 for upper body
// lifts and 5 for lower, twice that with a passed TM test.
func TestKilogramSteps(t *testing.T) {
	s := DefaultSettingsFor(UnitKG)
	cfg := LeaderAnchorBBB()
	cfg.TMIncrementUpper, cfg.TMIncrementLower = TMIncrementsFor(UnitKG)
	hit := []PRSet{{Target: 5, Actual: 8}}
	if got, _ := RecommendBump(cfg, 50, true, hit, 0, false, s); got != 52.5 {
		t.Errorf("upper bump = %v, want 52.5", got)
	}
	if got, _ := RecommendBump(cfg, 140, false, hit, 0, false, s); got != 145 {
		t.Errorf("lower bump = %v, want 145", got)
	}
}
