package views

import (
	"strconv"
	"strings"

	"github.com/kbremont/531-tracker/server/internal/training"
)

// FmtWeight renders a weight. The rule lives in training, which writes
// weights into the text it composes.
func FmtWeight(v float64) string { return training.FormatWeight(v) }

// FmtWeightUnit renders a weight with its unit, for example "195 lb".
func FmtWeightUnit(v float64, unit string) string { return training.FormatWeightUnit(v, unit) }

// FmtPct renders a fraction of the training max as a whole percentage.
func FmtPct(p float64) string { return training.FormatPct(p) }

func FmtInt(n int) string { return strconv.Itoa(n) }

// FmtPlateList renders the plates available as the comma-separated list the
// settings form edits.
func FmtPlateList(plates []float64) string {
	parts := make([]string, len(plates))
	for i, p := range plates {
		parts[i] = strconv.FormatFloat(p, 'f', -1, 64)
	}
	return strings.Join(parts, ", ")
}
