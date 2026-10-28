package training

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/kbremont/531-tracker/server/internal/db"
)

func amrap(weight float64, reps int64) db.ListAMRAPSetsRow {
	return db.ListAMRAPSetsRow{Weight: weight, ActualReps: sql.NullInt64{Int64: reps, Valid: true}}
}

// loggedOn is an AMRAP set completed at midday UTC on the given date.
func loggedOn(date string, weight float64, reps int64) db.ListAMRAPSetsRow {
	row := amrap(weight, reps)
	row.CompletedAt = sql.NullString{String: date + " 12:00:00", Valid: true}
	return row
}

func pastOn(date string, weight float64, reps int64) db.PastEstimate {
	return db.PastEstimate{ID: reps, RecordedOn: date, Weight: weight, Reps: reps}
}

// logged is estimates for sets that were performed in the app.
func logged(rows ...db.ListAMRAPSetsRow) []estimate {
	return mergeEstimates(rows, nil)
}

// TestMergeEstimates interleaves the two sources by date, and on a shared day
// puts the one from before the app first.
func TestMergeEstimates(t *testing.T) {
	got := mergeEstimates(
		[]db.ListAMRAPSetsRow{
			loggedOn("2025-03-10", 100, 9),
			loggedOn("2025-06-01", 110, 5),
			loggedOn("2025-06-01", 115, 3),
		},
		[]db.PastEstimate{
			pastOn("2024-11-02", 120, 1),
			pastOn("2025-06-01", 135, 8),
			pastOn("2025-07-04", 140, 2),
		},
	)

	type row struct {
		weight float64
		past   bool
	}
	want := []row{{120, true}, {100, false}, {135, true}, {110, false}, {115, false}, {140, true}}
	if len(got) != len(want) {
		t.Fatalf("got %d estimates, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Weight != w.weight || got[i].Past != w.past {
			t.Errorf("estimate %d = %v past=%v, want %v past=%v", i, got[i].Weight, got[i].Past, w.weight, w.past)
		}
	}
	if got[0].ID != 1 || got[0].Reps != 1 {
		t.Errorf("past estimate lost its id or reps: %+v", got[0])
	}
}

func TestMergeEstimatesOneSideEmpty(t *testing.T) {
	if got := mergeEstimates(nil, nil); len(got) != 0 {
		t.Errorf("nothing merged to %+v", got)
	}
	if got := mergeEstimates(nil, []db.PastEstimate{pastOn("2025-01-01", 100, 1)}); len(got) != 1 || !got[0].Past {
		t.Errorf("past only = %+v, want one past estimate", got)
	}
	if got := mergeEstimates([]db.ListAMRAPSetsRow{loggedOn("2025-01-01", 100, 1)}, nil); len(got) != 1 || got[0].Past {
		t.Errorf("logged only = %+v, want one logged estimate", got)
	}
}

// TestBuildChartPastEstimates plots entries from before the app among logged
// sets, flags them, and counts them toward the best.
func TestBuildChartPastEstimates(t *testing.T) {
	ests := mergeEstimates(
		[]db.ListAMRAPSetsRow{loggedOn("2025-06-01", 100, 9)}, // 130
		[]db.PastEstimate{pastOn("2025-01-15", 135, 8)},       // 171, beats every logged set
	)
	points, _, _, _, best := buildChart(ests)

	if best != 171 {
		t.Errorf("best = %v, want the past 135 x 8 at 171", best)
	}
	if len(points) != 2 || !points[0].Past || points[1].Past {
		t.Errorf("points = %+v, want the past one first and flagged, the logged one second", points)
	}
	if points[0].Label != "171" || points[1].Label != "130" {
		t.Errorf("labels %q and %q, want 171 and 130", points[0].Label, points[1].Label)
	}
}

// TestBuildChartSinglePoint draws the first estimate on its own rather than
// waiting for a second: a centred dot, inside the axis range, labelled.
func TestBuildChartSinglePoint(t *testing.T) {
	points, path, yMin, yMax, best := buildChart(logged(amrap(100, 9)))

	if len(points) != 1 {
		t.Fatalf("got %d points, want 1", len(points))
	}
	p := points[0]
	if want := (plotLeft + plotRight) / 2; p.X != want {
		t.Errorf("lone point at x=%v, want centred at %v", p.X, want)
	}
	if p.Y <= plotTop || p.Y >= plotBottom {
		t.Errorf("lone point at y=%v, want inside the plot (%v to %v)", p.Y, plotTop, plotBottom)
	}
	if p.Label != "130" || best != 130 {
		t.Errorf("label %q, best %v; want 130 for 100 x 9", p.Label, best)
	}
	if !(yMin < 130 && 130 < yMax) {
		t.Errorf("axis %v to %v does not contain 130", yMin, yMax)
	}
	if !strings.HasPrefix(path, "M ") || strings.Contains(path, "L") {
		t.Errorf("path %q, want a single move with no line", path)
	}
}

// TestBuildChartLine spans the full width once there is something to join.
func TestBuildChartLine(t *testing.T) {
	points, path, _, _, _ := buildChart(logged(amrap(100, 9), amrap(90, 8)))
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}
	if points[0].X != plotLeft || points[1].X != plotRight {
		t.Errorf("points at x=%v and %v, want %v and %v", points[0].X, points[1].X, plotLeft, plotRight)
	}
	if !strings.Contains(path, " L ") {
		t.Errorf("path %q, want a line between the points", path)
	}
}

func TestBuildChartEmpty(t *testing.T) {
	points, path, _, _, best := buildChart(nil)
	if points != nil || path != "" || best != 0 {
		t.Errorf("empty chart = %v, %q, %v; want nothing", points, path, best)
	}
}
