package server

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/kbremont/531-tracker/internal/db"
)

func amrap(weight float64, reps int64) db.ListAMRAPSetsRow {
	return db.ListAMRAPSetsRow{Weight: weight, ActualReps: sql.NullInt64{Int64: reps, Valid: true}}
}

// TestBuildChartSinglePoint draws the first estimate on its own rather than
// waiting for a second: a centred dot, inside the axis range, labelled.
func TestBuildChartSinglePoint(t *testing.T) {
	points, path, yMin, yMax, best := buildChart([]db.ListAMRAPSetsRow{amrap(100, 9)})

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
	points, path, _, _, _ := buildChart([]db.ListAMRAPSetsRow{amrap(100, 9), amrap(90, 8)})
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
