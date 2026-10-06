package views

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func renderDashboard(t *testing.T, d DashboardVM) string {
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
	d := DashboardVM{
		Unit: "lb",
		Next: &NextUp{LiftName: "Squat", CycleLabel: "Anchor - Round 1", Blocked: true},
		TestResults: []TMTestResultVM{
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
	d := DashboardVM{
		Unit: "lb",
		Review: &TMReviewVM{
			Title: "Block 2: new training maxes",
			Key:   "b2:c0",
			Unit:  "lb",
			Rows: []TMReviewRow{
				{Slug: "squat", Name: "Squat", Verdict: "Hit every PR set: +10 lb", Tone: "up", Current: 250, Proposed: 260, Field: "tm_squat"},
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
	d := DashboardVM{Unit: "lb", Next: &NextUp{LiftName: "Press", CycleLabel: "7th Week Deload", CanSkip: true}}
	html := renderDashboard(t, d)
	if !strings.Contains(html, `action="/skip-cycle"`) || !strings.Contains(html, "Start session") {
		t.Error("expected both start and skip")
	}

	d.Next.CanSkip = false
	if strings.Contains(renderDashboard(t, d), "/skip-cycle") {
		t.Error("skip offered for a cycle that cannot be skipped")
	}
}
