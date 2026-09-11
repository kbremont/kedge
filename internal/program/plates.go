package program

import "math"

// PlatePair is a plate denomination and how many of them go on each side of
// the bar.
type PlatePair struct {
	Weight float64
	Count  int
}

// PlatesFor works out how to load a bar, returning the plates needed per side
// and any weight that could not be made from the available denominations.
//
// A non-zero remainder means the prescription is not loadable as written --
// worth surfacing in the UI rather than silently lying about the weight.
func PlatesFor(target float64, s Settings) ([]PlatePair, float64) {
	if target <= s.BarWeight {
		return nil, 0
	}

	perSide := (target - s.BarWeight) / 2
	var out []PlatePair

	for _, p := range s.Plates {
		if p <= 0 || perSide < p {
			continue
		}
		n := int(math.Floor(roundTo(perSide/p, 1e-6)))
		if n == 0 {
			continue
		}
		out = append(out, PlatePair{Weight: p, Count: n})
		perSide -= float64(n) * p
	}

	// Floating point noise accumulates across subtractions of 2.5s; anything
	// under a hundredth of a pound is not a real remainder.
	perSide = roundTo(perSide, 0.01)
	if math.Abs(perSide) < 0.01 {
		perSide = 0
	}
	return out, perSide * 2
}

// Epley1RM estimates a one-rep max from a set taken to failure. This is the
// formula 5/3/1 uses to score AMRAP sets against each other.
func Epley1RM(weight float64, reps int) float64 {
	if reps <= 0 || weight <= 0 {
		return 0
	}
	if reps == 1 {
		return weight
	}
	return weight * (1 + float64(reps)/30)
}

func roundTo(v, step float64) float64 {
	return math.Round(v/step) * step
}
