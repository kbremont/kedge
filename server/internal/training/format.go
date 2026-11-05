package training

import (
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kbremont/kedge/server/internal/program"
)

// FormatWeight renders a weight without a trailing ".0", since almost every
// prescription lands on a whole number. A weight that can be put on a bar, a
// multiple of a quarter such as a 1.25 kg plate or 61.25, is written in full;
// anything else, such as an estimated max, to one decimal.
func FormatWeight(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	if q := v * 4; q == math.Trunc(q) {
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

// FormatWeightUnit renders a weight with its unit, for example "195 lb".
func FormatWeightUnit(v float64, unit string) string {
	return FormatWeight(v) + " " + unit
}

// FormatPct renders a fraction of the training max as a whole percentage.
func FormatPct(p float64) string {
	return strconv.Itoa(int(p*100+0.5)) + "%"
}

// formatPlates lists the plates for one side of the bar, heaviest first.
func formatPlates(pairs []program.PlatePair) string {
	if len(pairs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		w := FormatWeight(p.Weight)
		for i := 0; i < p.Count; i++ {
			parts = append(parts, w)
		}
	}
	return strings.Join(parts, ", ")
}

// setTag names a set for display.
func setTag(kind program.SetKind, supp program.SuppKind) string {
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

// RepTarget renders the rep target as it appears in the book: a trailing plus
// marks a set taken for as many reps as possible, and a range such as 3-5 is
// written with an en dash.
func (s Set) RepTarget() string {
	r := strconv.Itoa(s.TargetReps)
	switch {
	case s.IsAMRAP:
		return r + "+"
	case s.MaxReps > s.TargetReps:
		return r + "–" + strconv.Itoa(s.MaxReps)
	}
	return r
}

// PlateLine says how to load the bar for one side, for example
// "Bar + 45, 25 per side". When it cannot be loaded exactly, Unloadable says
// by how much it falls short.
func (s Set) PlateLine() string {
	return plateLine(s.Plates)
}

// PlateLine is the same line for a weight on its own.
func (p Plates) PlateLine() string {
	return plateLine(p.Plates)
}

func plateLine(plates string) string {
	if plates == "" {
		return "Empty bar"
	}
	return "Bar + " + plates + " per side"
}

// AsksReps reports whether a set needs its reps entered rather than ticked
// off: a PR set, or a range such as 3-5 where the number done is the point.
func (s Set) AsksReps() bool {
	return s.IsAMRAP || s.MaxReps > 0
}

// RepChoices offers ten answers for a set that asks for its reps, so they can
// be logged with one tap. An anchor PR set starts at its target, which covers
// all but the most exceptional one. A set with a window, the TM test's 100%
// AMRAP or the deload's 80% range, starts at a single rep, since falling short
// of it is exactly what those sets are there to find out.
func (s Set) RepChoices() []int {
	start := 1
	if s.IsAMRAP && s.MaxReps == 0 && s.TargetReps > 1 {
		start = s.TargetReps
	}
	out := make([]int, 10)
	for i := range out {
		out[i] = start + i
	}
	return out
}

// IsTargetRep marks the answers that meet the prescription.
func (s Set) IsTargetRep(r int) bool {
	if s.MaxReps > 0 {
		return r >= s.TargetReps && r <= s.MaxReps
	}
	return r == s.TargetReps
}

// dayLayout is how a calendar date with no time is written in the database and
// in a date input.
const dayLayout = "2006-01-02"

// formatDate renders a SQLite UTC timestamp in the server's local zone, which
// is what the TZ environment variable in the compose file is for.
func formatDate(ts sql.NullString) string {
	if !ts.Valid {
		return ""
	}
	t, err := time.Parse("2006-01-02 15:04:05", ts.String)
	if err != nil {
		return ts.String
	}
	return t.UTC().In(time.Local).Format("Jan 2")
}

// formatDay renders a calendar date such as "2026-03-03" as "Mar 3, 2026". The
// year is kept because a date from a notebook can be from any year.
func formatDay(d string) string {
	t, err := time.Parse(dayLayout, d)
	if err != nil {
		return d
	}
	return t.Format("Jan 2, 2006")
}

// formatTimestamp renders a non-null SQLite UTC timestamp in local time.
func formatTimestamp(ts string) string {
	return formatDate(sql.NullString{String: ts, Valid: true})
}
