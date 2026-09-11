package program

import (
	"testing"
)

func TestPlatesFor(t *testing.T) {
	s := DefaultSettings()

	cases := []struct {
		target float64
		want   []PlatePair
	}{
		{45, nil}, // the empty bar
		{40, nil}, // nothing lighter than the bar is loadable
		{95, []PlatePair{{25, 1}}},
		{135, []PlatePair{{45, 1}}},
		{185, []PlatePair{{45, 1}, {25, 1}}},
		{225, []PlatePair{{45, 2}}},
		{255, []PlatePair{{45, 2}, {10, 1}, {5, 1}}},
		{285, []PlatePair{{45, 2}, {25, 1}, {5, 1}}},
		{120, []PlatePair{{35, 1}, {2.5, 1}}},
		{150, []PlatePair{{45, 1}, {5, 1}, {2.5, 1}}},
		{180, []PlatePair{{45, 1}, {10, 2}, {2.5, 1}}},
		{195, []PlatePair{{45, 1}, {25, 1}, {5, 1}}},
		{210, []PlatePair{{45, 1}, {35, 1}, {2.5, 1}}},
		{240, []PlatePair{{45, 2}, {5, 1}, {2.5, 1}}},
		{270, []PlatePair{{45, 2}, {10, 2}, {2.5, 1}}},
	}

	for _, tc := range cases {
		got, remainder := PlatesFor(tc.target, s)
		if remainder != 0 {
			t.Errorf("PlatesFor(%v): remainder = %v, want 0", tc.target, remainder)
		}
		if len(got) != len(tc.want) {
			t.Errorf("PlatesFor(%v) = %v, want %v", tc.target, got, tc.want)
			continue
		}
		for i := range got {
			if !nearly(got[i].Weight, tc.want[i].Weight) || got[i].Count != tc.want[i].Count {
				t.Errorf("PlatesFor(%v) = %v, want %v", tc.target, got, tc.want)
				break
			}
		}
	}
}

// TestPlatesForAlwaysBalances is the invariant that matters at the rack: the
// plates the app names, doubled and added to the bar, plus whatever it admits
// it cannot load, must equal the weight it prescribed.
func TestPlatesForAlwaysBalances(t *testing.T) {
	s := DefaultSettings()

	for target := 45.0; target <= 700; target += 2.5 {
		plates, remainder := PlatesFor(target, s)
		total := s.BarWeight + remainder
		for _, p := range plates {
			total += p.Weight * float64(p.Count) * 2
		}
		if !nearly(total, target) {
			t.Errorf("PlatesFor(%v) accounts for %v (plates %v, remainder %v)", target, total, plates, remainder)
		}
	}
}

// TestPlatesForUnloadableWeight covers a sparse plate set, where the app has
// to admit the prescription cannot be made exactly rather than quietly
// rounding it away.
func TestPlatesForUnloadableWeight(t *testing.T) {
	s := DefaultSettings()
	s.Plates = []float64{45, 25, 10}

	plates, remainder := PlatesFor(50, s)
	if len(plates) != 0 {
		t.Errorf("got plates %v, want none", plates)
	}
	if !nearly(remainder, 5) {
		t.Errorf("remainder = %v, want 5", remainder)
	}

	plates, remainder = PlatesFor(100, s)
	if len(plates) != 1 || !nearly(plates[0].Weight, 25) || plates[0].Count != 1 {
		t.Errorf("got plates %v, want one 25", plates)
	}
	if !nearly(remainder, 5) { // 45 bar + 50 in plates leaves 5
		t.Errorf("remainder = %v, want 5", remainder)
	}
}

func TestEpley1RM(t *testing.T) {
	cases := []struct {
		weight float64
		reps   int
		want   float64
	}{
		{300, 1, 300},   // a single is its own max
		{255, 5, 297.5}, // 255 * (1 + 5/30)
		{135, 10, 180},
		{225, 3, 247.5},
		{0, 5, 0},   // no weight, no estimate
		{200, 0, 0}, // a set that was not performed
		{200, -1, 0},
	}
	for _, tc := range cases {
		if got := Epley1RM(tc.weight, tc.reps); !nearly(got, tc.want) {
			t.Errorf("Epley1RM(%v, %d) = %v, want %v", tc.weight, tc.reps, got, tc.want)
		}
	}
}

// TestEpleyRanksAMRAPSets is what the estimate is actually for: comparing PR
// sets across cycles, where more reps at a lighter load can still be a better
// performance.
func TestEpleyRanksAMRAPSets(t *testing.T) {
	lighterButMoreReps := Epley1RM(255, 8) // 323.0
	heavierFewerReps := Epley1RM(285, 3)   // 313.5

	if lighterButMoreReps <= heavierFewerReps {
		t.Errorf("255x8 (%v) should outrank 285x3 (%v)", lighterButMoreReps, heavierFewerReps)
	}
}
