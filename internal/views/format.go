package views

import (
	"strconv"
	"strings"

	"github.com/kbremont/531-tracker/internal/program"
)

// FmtWeight renders a weight without a trailing ".0", since almost every
// prescription lands on a whole number.
func FmtWeight(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// FmtWeightUnit renders a weight with its unit, for example "195 lb".
func FmtWeightUnit(v float64, unit string) string {
	return FmtWeight(v) + " " + unit
}

// FmtPct renders a fraction of the training max as a whole percentage.
func FmtPct(p float64) string {
	return strconv.Itoa(int(p*100+0.5)) + "%"
}

func FmtInt(n int) string { return strconv.Itoa(n) }

// FmtPlates lists the plates for one side of the bar, heaviest first.
func FmtPlates(pairs []program.PlatePair) string {
	if len(pairs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		w := FmtWeight(p.Weight)
		for i := 0; i < p.Count; i++ {
			parts = append(parts, w)
		}
	}
	return strings.Join(parts, ", ")
}

// RepTarget renders the rep target as it appears in the book: a trailing plus
// marks a set taken for as many reps as possible.
func RepTarget(reps int, amrap bool) string {
	s := strconv.Itoa(reps)
	if amrap {
		return s + "+"
	}
	return s
}

// SetTag names a set for display.
func SetTag(kind program.SetKind, supp program.SuppKind) string {
	switch kind {
	case program.KindWarmup:
		return "Warmup"
	case program.KindMain:
		return "Main"
	case program.KindSupp:
		switch supp {
		case program.SuppBBB:
			return "BBB"
		case program.SuppFSL:
			return "FSL"
		}
		return "Supplemental"
	}
	return string(kind)
}
