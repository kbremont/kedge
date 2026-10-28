package web

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/kbremont/531-tracker/server/internal/training"
)

func renderDashboard(t *testing.T, d training.Dashboard) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Dashboard(d).Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

// TestTMTestResultCards covers the two kinds of TM test result: a pass is
// news with nothing to fill in, and a failure offers to lower the training max
// with the proposal prefilled, capped at the current value, and a way to keep
// it that skips the input's validation.
func TestTMTestResultCards(t *testing.T) {
	d := training.Dashboard{
		Unit: "lb",
		Next: &training.NextUp{LiftName: "Squat", CycleLabel: "Anchor - Round 1", Blocked: true},
		TestResults: []training.TMTestResult{
			{WorkoutID: 7, LiftName: "Bench Press", Unit: "lb", Tested: 200, Reps: 4, Headline: "Nice job, your TM is right.", Tone: "hold", Current: 200},
			{WorkoutID: 8, LiftName: "Squat", Unit: "lb", Tested: 250, Reps: 2, Headline: "Your TM is too high.", Tone: "down", NeedsDecision: true, Current: 250, Proposed: 225},
		},
	}

	html := renderDashboard(t, d)

	for _, want := range []string{
		"Bench Press TM test", "200 x 4 at 100%", "Nice job, your TM is right.", "verdict verdict-hold",
		"Squat TM test", "250 x 2 at 100%", "Your TM is too high.", "verdict verdict-down",
		`action="/tm-test/8"`, `name="tm"`, `max="250"`, `value="225"`,
		`name="action" value="lower"`, `name="action" value="keep" formnovalidate`, "Keep 250 lb",
		"Settle your training maxes above to start.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("result cards missing %q", want)
		}
	}
	if strings.Contains(html, `action="/tm-test/7"`) {
		t.Error("a passing test should not ask for anything")
	}
	if strings.Contains(html, "Start session") {
		t.Error("start offered while a decision is pending")
	}
}

// TestBlockReviewCard lays out one editable input per lift, prefilled with the
// proposal and left uncapped: the start of a block is the one place a training
// max may rise.
func TestBlockReviewCard(t *testing.T) {
	d := training.Dashboard{
		Unit: "lb",
		Review: &training.TMReview{
			Title: "Block 2: new training maxes",
			Key:   "b2:c0",
			Unit:  "lb",
			Rows: []training.TMReviewRow{
				{Slug: "squat", Name: "Squat", Verdict: "Hit every PR set: +10 lb", Tone: "up", Current: 250, Proposed: 260},
			},
		},
	}
	html := renderDashboard(t, d)
	for _, want := range []string{`name="key" value="b2:c0"`, `name="tm_squat"`, `value="260"`, "verdict verdict-up", "Accept training maxes"} {
		if !strings.Contains(html, want) {
			t.Errorf("block review missing %q", want)
		}
	}
	if strings.Contains(html, `max="`) {
		t.Error("block review input should not be capped")
	}
}

// TestSkipOffered shows the skip button only when the cycle may be skipped.
func TestSkipOffered(t *testing.T) {
	d := training.Dashboard{Unit: "lb", Next: &training.NextUp{LiftName: "Press", CycleLabel: "7th Week Deload", CanSkip: true}}
	html := renderDashboard(t, d)
	if !strings.Contains(html, `action="/skip-cycle"`) || !strings.Contains(html, "Start session") {
		t.Error("expected both start and skip")
	}

	d.Next.CanSkip = false
	if strings.Contains(renderDashboard(t, d), "/skip-cycle") {
		t.Error("skip offered for a cycle that cannot be skipped")
	}
}

func renderLift(t *testing.T, l LiftPage) string {
	t.Helper()
	var buf bytes.Buffer
	if err := LiftDetail(l).Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

// TestLiftChartFromFirstPoint shows the chart as soon as there is one
// estimate, labelled with its value, and keeps the empty state for none.
func TestLiftChartFromFirstPoint(t *testing.T) {
	one := LiftPage{Lift: training.Lift{Name: "Overhead Press", Unit: "lb", CurrentTM: 100, BestE1RM: 130,
		Points: []training.ChartPoint{{X: 160, Y: 70, Label: "130"}}, Path: "M 160.0 70.0", YMin: 120, YMax: 140}}
	html := renderLift(t, one)
	for _, want := range []string{`<svg class="chart"`, `<circle class="dot" cx="160" cy="70"`, `text-anchor="middle">130</text>`} {
		if !strings.Contains(html, want) {
			t.Errorf("one-point chart missing %q", want)
		}
	}
	if strings.Contains(html, "Not enough data yet") {
		t.Error("empty state shown alongside a point")
	}

	two := one
	two.Points = []training.ChartPoint{{X: 10, Y: 90, Label: "120"}, {X: 310, Y: 40, Label: "130"}}
	if html := renderLift(t, two); strings.Contains(html, `text-anchor="middle"`) {
		t.Error("a line should not carry the lone-point label")
	}

	none := LiftPage{Lift: training.Lift{Name: "Overhead Press", Unit: "lb", CurrentTM: 100}}
	if html := renderLift(t, none); !strings.Contains(html, "Not enough data yet") || strings.Contains(html, "<svg") {
		t.Error("expected the empty state and no chart with no points")
	}
}

// TestLiftPastEstimates lists the estimates from before the app with their
// year, a delete button each, and the form to add another, and draws their
// points hollow.
func TestLiftPastEstimates(t *testing.T) {
	l := LiftPage{Lift: training.Lift{Name: "Overhead Press", Slug: "press", Unit: "lb", CurrentTM: 100, BestE1RM: 171,
		Points: []training.ChartPoint{{X: 10, Y: 40, Label: "171", Past: true}, {X: 310, Y: 90, Label: "130"}},
		Path:   "M 10.0 40.0 L 310.0 90.0", YMin: 120, YMax: 180,
		PastEstimates: []training.PastEstimate{
			{ID: 7, Date: "Mar 3, 2025", Weight: 135, Reps: 8, Estimate: 171},
			{ID: 9, Date: "Jun 1, 2025", Weight: 120, Reps: 1, Estimate: 120},
		},
		Today: "2026-10-06",
	}}
	html := renderLift(t, l)
	for _, want := range []string{
		"Past estimates", "Mar 3, 2025", "135 x 8", "est. 171 lb", "Jun 1, 2025", "120 x 1",
		`action="/lift/press/estimates/7/delete"`, `action="/lift/press/estimates/9/delete"`,
		`action="/lift/press/estimates"`, `type="date"`, `max="2026-10-06"`, `name="reps" value="1"`,
		`<circle class="dot past" cx="10" cy="40"`, `<circle class="dot" cx="310" cy="90"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("lift page missing %q", want)
		}
	}
	if n := strings.Count(html, "/estimates/"); n != 2 {
		t.Errorf("%d delete forms, want one per estimate", n)
	}
	if strings.Contains(html, `class="error"`) {
		t.Error("error shown without one")
	}
	// The form waits, closed, behind a plus in the heading.
	if !strings.Contains(html, `<details class="past-estimates"><summary><h2>Past estimates</h2><span class="plus" aria-hidden="true"><span>+</span></span></summary>`) {
		t.Error("expected the add form closed behind a plus")
	}

	// A rejected entry reopens it with the error in view.
	l.Error = true
	if html := renderLift(t, l); !strings.Contains(html, `<details class="past-estimates" open>`) || !strings.Contains(html, `class="error"`) {
		t.Error("a rejected estimate should reopen the form with an error")
	}

	// With none yet there is still a form to add the first.
	l = LiftPage{Lift: training.Lift{Name: "Overhead Press", Slug: "press", Unit: "lb", Today: "2026-10-06"}}
	html = renderLift(t, l)
	if !strings.Contains(html, `action="/lift/press/estimates"`) || strings.Contains(html, "/delete") {
		t.Error("expected the add form and no delete buttons with no estimates")
	}
	if !strings.Contains(html, "estimates from before the app") {
		t.Error("the empty state does not mention past estimates")
	}
}
