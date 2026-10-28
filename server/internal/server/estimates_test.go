package server

import (
	"context"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kbremont/531-tracker/server/internal/db"
)

func (h *harness) addPast(slug, date, weight, reps string) string {
	h.t.Helper()
	rec := h.post("/lift/"+slug+"/estimates", url.Values{"recorded_on": {date}, "weight": {weight}, "reps": {reps}})
	return rec.Header().Get("Location")
}

func (h *harness) pastEstimates(slug string) []db.PastEstimate {
	h.t.Helper()
	rows, err := h.s.q.ListPastEstimates(context.Background(), slug)
	if err != nil {
		h.t.Fatal(err)
	}
	return rows
}

func (h *harness) liftPage(slug string) string {
	h.t.Helper()
	return h.do("GET", "/lift/"+slug, nil).Body.String()
}

func (h *harness) bestE1RM(slug string) float64 {
	h.t.Helper()
	best, err := h.s.bestE1RM(context.Background(), slug)
	if err != nil {
		h.t.Fatal(err)
	}
	return best
}

// TestPastEstimatesFeedTheChartAndBest enters a notebook estimate before any
// session has been logged, then a TM test on top of it: the estimate is on the
// page as a hollow dot and counts toward the best, and the test's own set
// joins it as a solid dot later in the line.
func TestPastEstimatesFeedTheChartAndBest(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	lastYear := time.Now().AddDate(-1, 0, 0)
	loc := h.addPast("press", lastYear.Format(dayLayout), "120", "1")
	if loc != "/lift/press" {
		t.Errorf("redirected to %q, want the lift page", loc)
	}

	page := h.liftPage("press")
	mustContain(t, page,
		lastYear.Format("Jan 2, 2006"), "120 x 1", "est. 120 lb",
		`class="dot past"`, "best estimated max 120 lb",
		"/lift/press/estimates/1/delete")
	if got := h.bestE1RM("press"); got != 120 {
		t.Errorf("best estimated max = %v, want 120", got)
	}

	// The 100% set of the press TM test, 100 x 9, is worth 130.
	h.testSession(map[string]int{"press": 9, "deadlift": 4, "bench": 4, "squat": 2})
	if got := h.bestE1RM("press"); math.Abs(got-130) > 0.01 {
		t.Errorf("best estimated max after the TM test = %v, want 130", got)
	}
	page = h.liftPage("press")
	mustContain(t, page, "best estimated max 130 lb")
	past, logged := strings.Index(page, `class="dot past"`), strings.Index(page, `<circle class="dot"`)
	if past < 0 || logged < 0 || past > logged {
		t.Errorf("want the hollow dot before the solid one on the chart (past at %d, logged at %d)", past, logged)
	}
}

// TestPastEstimatesConvertSets treats a notebook PR set the way a logged one
// is treated, and the best is the highest of them all.
func TestPastEstimatesConvertSets(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	h.addPast("press", "2025-01-15", "120", "1")
	h.addPast("press", "2025-06-01", "135", "5")
	if got := h.bestE1RM("press"); math.Abs(got-157.5) > 0.01 {
		t.Errorf("best = %v, want 157.5 from 135 x 5", got)
	}
	page := h.liftPage("press")
	mustContain(t, page, "Jan 15, 2025", "Jun 1, 2025", "est. 157.5 lb", "best estimated max 157.5 lb")

	// Other lifts are untouched.
	if got := h.bestE1RM("bench"); got != 0 {
		t.Errorf("bench best = %v, want 0", got)
	}
}

func TestPastEstimateRejectsBadInput(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	tomorrow := time.Now().AddDate(0, 0, 1).Format(dayLayout)
	for name, form := range map[string][3]string{
		"future date":    {tomorrow, "120", "1"},
		"zero weight":    {"2025-01-15", "0", "1"},
		"negative":       {"2025-01-15", "-5", "1"},
		"under a pound":  {"2025-01-15", "0.5", "1"},
		"not a number":   {"2025-01-15", "heavy", "1"},
		"infinite":       {"2025-01-15", "Inf", "1"},
		"nan":            {"2025-01-15", "NaN", "1"},
		"zero reps":      {"2025-01-15", "120", "0"},
		"too many reps":  {"2025-01-15", "120", "31"},
		"fractional rep": {"2025-01-15", "120", "1.5"},
		"no date":        {"", "120", "1"},
		"bad date":       {"15/01/2025", "120", "1"},
		"no reps":        {"2025-01-15", "120", ""},
	} {
		if loc := h.addPast("press", form[0], form[1], form[2]); loc != "/lift/press?error=1" {
			t.Errorf("%s: redirected to %q, want the lift page with an error", name, loc)
		}
	}
	if n := len(h.pastEstimates("press")); n != 0 {
		t.Errorf("%d estimates written from invalid input, want 0", n)
	}
	mustContain(t, h.do("GET", "/lift/press?error=1", nil).Body.String(), "That estimate was not saved")
	mustNotContain(t, h.liftPage("press"), "That estimate was not saved")

	rec := h.post("/lift/curl/estimates", url.Values{"recorded_on": {"2025-01-15"}, "weight": {"120"}, "reps": {"1"}})
	if rec.Code != 404 {
		t.Errorf("unknown lift = %d, want 404", rec.Code)
	}
	var n int
	if err := h.s.conn.QueryRow(`SELECT COUNT(*) FROM past_estimates`).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d estimates in the database after invalid input (%v), want 0", n, err)
	}

	// Today and the bounds are fine.
	h.addPast("press", time.Now().Format(dayLayout), "1", "30")
	if n := len(h.pastEstimates("press")); n != 1 {
		t.Errorf("%d estimates after a valid one, want 1", n)
	}
}

func TestDeletePastEstimate(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	h.addPast("press", "2025-01-15", "120", "1")
	h.addPast("bench", "2025-02-15", "200", "1")
	press := h.pastEstimates("press")[0]
	bench := h.pastEstimates("bench")[0]

	// A stale page aimed at another lift's slug does nothing.
	h.post("/lift/press/estimates/"+strconv.FormatInt(bench.ID, 10)+"/delete", nil)
	if len(h.pastEstimates("bench")) != 1 {
		t.Fatal("deleting through the wrong lift removed bench's estimate")
	}

	rec := h.post("/lift/press/estimates/"+strconv.FormatInt(press.ID, 10)+"/delete", nil)
	if loc := rec.Header().Get("Location"); loc != "/lift/press" {
		t.Errorf("redirected to %q, want the lift page", loc)
	}
	if n := len(h.pastEstimates("press")); n != 0 {
		t.Errorf("%d press estimates after deleting, want 0", n)
	}
	mustNotContain(t, h.liftPage("press"), "Jan 15, 2025", `class="dot past"`)
	if got := h.bestE1RM("press"); got != 0 {
		t.Errorf("best after deleting = %v, want 0", got)
	}
	if len(h.pastEstimates("bench")) != 1 {
		t.Error("deleting press's estimate removed bench's")
	}

	if rec := h.post("/lift/press/estimates/nope/delete", nil); rec.Code != 404 {
		t.Errorf("a bad id = %d, want 404", rec.Code)
	}
}

// TestPastEstimatesAreNotEvidence runs the TM test with a huge notebook
// estimate on file: the test's top sets, the anchor's PR sets and the history
// are what they would be without it.
func TestPastEstimatesAreNotEvidence(t *testing.T) {
	ctx := context.Background()

	type snapshot struct {
		tops    []db.ListTMTestTopSetsRow
		anchors []db.ListAnchorPRSetsRow
		started int
	}
	run := func(withPast bool) snapshot {
		h := newHarness(t)
		h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)
		if withPast {
			h.addPast("press", "2020-01-01", "400", "10")
		}
		reps := map[string]int{"press": 2, "deadlift": 4, "bench": 4, "squat": 4}
		for range 4 {
			h.testSession(reps)
		}
		st, err := h.s.loadState(ctx)
		if err != nil {
			t.Fatal(err)
		}
		tops, err := h.s.q.ListTMTestTopSets(ctx, st.Block.ID)
		if err != nil {
			t.Fatal(err)
		}
		anchors, err := h.s.q.ListAnchorPRSets(ctx, st.Block.ID)
		if err != nil {
			t.Fatal(err)
		}
		// The 2-rep press test fails, whatever the notebook says.
		mustContain(t, h.dashboard(), "Overhead Press TM test", "Your TM is too high.")
		return snapshot{tops: tops, anchors: anchors, started: h.workoutCount()}
	}

	without, with := run(false), run(true)
	if len(without.tops) != 4 {
		t.Fatalf("%d TM test sets, want one per lift", len(without.tops))
	}
	// Row ids differ between databases only if the sessions do, and they do not.
	if !reflect.DeepEqual(without, with) {
		t.Errorf("a past estimate changed the evidence\nwithout: %+v\n   with: %+v", without, with)
	}
}
