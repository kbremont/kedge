package server

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kbremont/kedge/server/internal/auth"
	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/training"
)

// harness drives the real routes, auth included, over an in-memory database.
// It reads the database directly, or through the training rules, to check
// what the routes did.
type harness struct {
	t      *testing.T
	h      http.Handler
	conn   *sql.DB
	q      *db.Queries
	svc    *training.Service
	cookie *http.Cookie
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	conn, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1) // one connection, or each gets its own empty database
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := New(conn, log, Config{Password: "test"})
	return &harness{
		t:      t,
		h:      s.Handler(),
		conn:   conn,
		q:      db.New(conn),
		svc:    training.New(conn, log),
		cookie: &http.Cookie{Name: auth.CookieName, Value: s.sessions.Sign(time.Now().Add(time.Hour))},
	}
}

func (h *harness) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	h.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.AddCookie(h.cookie)
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if rec.Code >= 500 {
		h.t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
	}
	return rec
}

func (h *harness) post(path string, form url.Values) *httptest.ResponseRecorder {
	h.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	return h.do(http.MethodPost, path, form)
}

func (h *harness) dashboard() string {
	h.t.Helper()
	return h.do(http.MethodGet, "/", nil).Body.String()
}

// setup enters training maxes and joins the block at cycleIdx with nothing
// in it done yet.
func (h *harness) setup(tms map[string]float64, cycleIdx int) {
	h.t.Helper()
	form := url.Values{}
	for slug, v := range tms {
		form.Set("tm_"+slug, strconv.FormatFloat(v, 'f', -1, 64))
	}
	h.post("/setup", form)
	h.post("/setup/history", url.Values{"cycle": {strconv.Itoa(cycleIdx)}})
}

// session starts the next session, logs every set with the reps chosen for
// it, and finishes it.
func (h *harness) session(reps func(w db.Workout, set db.Set) int) db.Workout {
	h.t.Helper()
	ctx := context.Background()

	rec := h.post("/workout/start", nil)
	loc := rec.Header().Get("Location")
	id, err := strconv.ParseInt(strings.TrimPrefix(loc, "/workout/"), 10, 64)
	if err != nil {
		h.t.Fatalf("start session redirected to %q, want a workout", loc)
	}
	workout, err := h.q.GetWorkout(ctx, id)
	if err != nil {
		h.t.Fatal(err)
	}
	sets, err := h.q.ListSets(ctx, id)
	if err != nil {
		h.t.Fatal(err)
	}
	for _, set := range sets {
		n := int(set.TargetReps)
		if reps != nil {
			n = reps(workout, set)
		}
		h.post(fmt.Sprintf("/workout/%d/set/%d", id, set.Idx), url.Values{"reps": {strconv.Itoa(n)}})
	}
	h.post(fmt.Sprintf("/workout/%d/finish", id), nil)
	return workout
}

func (h *harness) currentTMs() map[string]float64 {
	h.t.Helper()
	rows, err := h.q.ListCurrentTrainingMaxes(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	out := make(map[string]float64, len(rows))
	for _, r := range rows {
		out[r.Slug] = r.Value
	}
	return out
}

func (h *harness) workoutCount() int {
	h.t.Helper()
	var n int
	if err := h.conn.QueryRow(`SELECT COUNT(*) FROM workouts`).Scan(&n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

func (h *harness) review() (key string) {
	h.t.Helper()
	d, err := h.svc.Dashboard(context.Background())
	if err != nil {
		h.t.Fatal(err)
	}
	r := d.Review
	if r == nil {
		h.t.Fatal("expected a pending training max review")
	}
	return r.Key
}

// isTopTestSet picks out the 100% AMRAP of a TM test.
func isTopTestSet(w db.Workout, set db.Set) bool {
	return w.Phase == "tm_test" && set.Kind == "main" && set.IsAmrap != 0
}

// testSession runs the next TM test session, taking the 100% set for the reps
// given for its lift.
func (h *harness) testSession(reps map[string]int) db.Workout {
	h.t.Helper()
	return h.session(func(w db.Workout, set db.Set) int {
		if isTopTestSet(w, set) {
			return reps[w.LiftSlug]
		}
		return int(set.TargetReps)
	})
}

// decide settles a failed TM test from its card.
func (h *harness) decide(w db.Workout, action, value string) {
	h.t.Helper()
	h.post(fmt.Sprintf("/tm-test/%d", w.ID), url.Values{"action": {action}, "tm": {value}})
}

func (h *harness) startBlocked() bool {
	h.t.Helper()
	before := h.workoutCount()
	h.post("/workout/start", nil)
	return h.workoutCount() == before
}

func mustContain(t *testing.T, page string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(page, want) {
			t.Errorf("page missing %q", want)
		}
	}
}

func mustNotContain(t *testing.T, page string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(page, u) {
			t.Errorf("page should not contain %q", u)
		}
	}
}

// TestTMTestThroughToNextBlock walks the 7th week end to end: each lift's TM
// test is reported as soon as it is done, only a failure offers to lower the
// training max, the anchor waits on that and runs on what was settled,
// skipping the deload ends the block, and the new block opens with a
// recommendation built from the test and the anchor's PR sets. Nothing moves
// until the user says so.
func TestTMTestThroughToNextBlock(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	page := h.dashboard()
	if !strings.Contains(page, "7th Week TM Test") || !strings.Contains(page, "round 7 of 11") {
		t.Fatalf("dashboard should offer the TM test as round 7 of 11:\n%s", page)
	}

	// One lift a day: press flies up for 7, deadlift and bench make 4, squat
	// only manages 2.
	testReps := map[string]int{"press": 7, "deadlift": 4, "bench": 4, "squat": 2}

	h.testSession(testReps)
	page = h.dashboard()
	mustContain(t, page, "Overhead Press TM test", "Strong set. This TM goes up at the end of the block.")
	mustNotContain(t, page, "Lower training max", "Settle your training maxes")
	if got := h.currentTMs()["press"]; got != 100 {
		t.Errorf("press TM moved to %v mid-block", got)
	}

	h.testSession(testReps)
	page = h.dashboard()
	mustContain(t, page, "Deadlift TM test", "Nice job, your TM is right.")
	mustNotContain(t, page, "Overhead Press TM test") // yesterday's news

	h.testSession(testReps)
	mustContain(t, h.dashboard(), "Bench Press TM test", "Nice job, your TM is right.")

	// The 100% set counts toward the best estimated max: 200 x 4 by Epley.
	best, err := h.svc.BestE1RM(context.Background(), "bench")
	if err != nil {
		t.Fatal(err)
	}
	if want := 200 * (1 + 4.0/30); math.Abs(best-want) > 0.01 {
		t.Errorf("bench best estimated max after the TM test = %v, want %v", best, want)
	}

	squat := h.testSession(testReps)
	page = h.dashboard()
	mustContain(t, page,
		"Squat TM test", "250 x 2 at 100%", "Your TM is too high.",
		`value="225"`, "Lower training max", "Keep 250 lb",
		"Settle your training maxes above to start.")
	mustNotContain(t, page, "Bench Press TM test", "Start session")

	// The anchor waits on the squat decision.
	if !h.startBlocked() {
		t.Fatal("the anchor started while a failed TM test was waiting")
	}

	h.decide(squat, "lower", "225")
	page = h.dashboard()
	mustNotContain(t, page, "Your TM is too high.")
	mustContain(t, page, "Start session")

	want := map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 225}
	got := h.currentTMs()
	for slug, v := range want {
		if got[slug] != v {
			t.Errorf("after the TM test, %s TM = %v, want %v", slug, got[slug], v)
		}
	}

	// The anchor runs on the settled training maxes. Deadlift misses its 1+ in
	// the last round; everything else makes its minimum.
	for i := range 12 {
		w := h.session(func(w db.Workout, set db.Set) int {
			if set.IsAmrap != 0 && w.LiftSlug == "deadlift" && w.Round == 3 {
				return 0
			}
			if set.IsAmrap != 0 {
				return int(set.TargetReps) + 2
			}
			return int(set.TargetReps)
		})
		if w.Phase != "anchor" {
			t.Fatalf("session %d of the anchor was %q", i, w.Phase)
		}
		if w.TrainingMax != want[w.LiftSlug] {
			t.Errorf("anchor %s ran at %v, want %v", w.LiftSlug, w.TrainingMax, want[w.LiftSlug])
		}
	}

	// The deload is offered, and skipped.
	page = h.dashboard()
	if !strings.Contains(page, "7th Week Deload") || !strings.Contains(page, "Skip it and start the next block") {
		t.Fatalf("dashboard should offer to skip the deload:\n%s", page)
	}
	h.post("/skip-cycle", nil)

	// The new block opens on its review, and nothing has moved yet.
	page = h.dashboard()
	mustContain(t, page,
		"Block 2: new training maxes",
		"PR sets and TM test: +10 lb", // press: every PR set, and 7 on the test
		"Missed a PR set: reset 10%",  // deadlift
		"Hit every PR set: +5 lb",     // bench
		"Hit every PR set: +10 lb",    // squat
		"Leader 1 - Round 1")
	if got := h.currentTMs(); got["press"] != 100 {
		t.Errorf("press TM moved to %v before the review was accepted", got["press"])
	}
	if !h.startBlocked() {
		t.Fatal("a session started while the block review was pending")
	}

	// Accept, editing squat below what was proposed.
	h.post("/tm-review", url.Values{
		"key":         {h.review()},
		"tm_press":    {"110"},
		"tm_deadlift": {"270"},
		"tm_bench":    {"205"},
		"tm_squat":    {"230"},
	})
	got = h.currentTMs()
	want = map[string]float64{"press": 110, "deadlift": 270, "bench": 205, "squat": 230}
	for slug, v := range want {
		if got[slug] != v {
			t.Errorf("after the block review, %s TM = %v, want %v", slug, got[slug], v)
		}
	}

	// And training carries on at the accepted numbers.
	w := h.session(nil)
	if w.CycleIndex != 0 || w.LiftSlug != "press" || w.TrainingMax != 110 {
		t.Errorf("first session of block 2 = cycle %d %s at %v, want cycle 0 press at 110", w.CycleIndex, w.LiftSlug, w.TrainingMax)
	}
}

// TestFailedTMTestHoldsOnlyTheAnchor covers the choices a failed test offers.
// The rest of the TM test week carries on regardless; keeping the training
// max is as good a decision as lowering it; and the offer can never raise.
func TestFailedTMTestHoldsOnlyTheAnchor(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	testReps := map[string]int{"press": 1, "deadlift": 4, "bench": 4, "squat": 2}
	press := h.testSession(testReps)
	mustContain(t, h.dashboard(), "Overhead Press TM test", "Your TM is too high.", "Start session")
	if h.startBlocked() {
		t.Fatal("a failed press test should not hold the rest of the TM test week")
	}
	// That start began deadlift's test; finish it, then the rest.
	d, err := h.svc.Dashboard(context.Background())
	if err != nil || d.Next == nil || !d.Next.InProgress {
		t.Fatalf("expected deadlift's test in progress: %v", err)
	}
	h.post(fmt.Sprintf("/workout/%d/abandon", d.Next.WorkoutID), nil)
	h.testSession(testReps)
	h.testSession(testReps)
	squat := h.testSession(testReps)

	// Both failures stay up until settled, and the anchor waits on them.
	mustContain(t, h.dashboard(), "Overhead Press TM test", "Squat TM test")
	if !h.startBlocked() {
		t.Fatal("the anchor started with two failed TM tests waiting")
	}

	h.decide(press, "keep", "")
	if !h.startBlocked() {
		t.Fatal("the anchor started with squat's failed TM test still waiting")
	}

	// A value above the training max is not a raise; it keeps it.
	h.decide(squat, "lower", "400")

	got := h.currentTMs()
	if got["press"] != 100 || got["squat"] != 250 {
		t.Errorf("TMs = press %v, squat %v; want both kept at 100 and 250", got["press"], got["squat"])
	}
	var notes int
	if err := h.conn.QueryRow(`SELECT COUNT(*) FROM training_maxes WHERE note = 'tm test'`).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Errorf("%d training max entries written for kept TMs, want 0", notes)
	}

	mustNotContain(t, h.dashboard(), "Your TM is too high.")
	w := h.session(nil)
	if w.Phase != "anchor" || w.TrainingMax != 100 {
		t.Errorf("first anchor session = %s at %v, want anchor press at 100", w.Phase, w.TrainingMax)
	}
}

// TestBlockReviewProposals checks the numbers the block review puts in front
// of the user, before anything is accepted.
func TestBlockReviewProposals(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 2)

	testReps := map[string]int{"press": 2, "deadlift": 4, "bench": 7, "squat": 5}
	press := h.testSession(testReps)
	for range 3 {
		h.testSession(testReps)
	}
	h.decide(press, "lower", "90")
	for range 12 {
		h.session(func(w db.Workout, set db.Set) int {
			if set.IsAmrap != 0 && w.LiftSlug == "deadlift" && w.Round == 3 {
				return 0
			}
			return int(set.TargetReps)
		})
	}
	h.post("/skip-cycle", nil)

	d, err := h.svc.Dashboard(context.Background())
	if err != nil || d.Review == nil {
		t.Fatalf("pending review = %v, %v; want a block review", d, err)
	}
	r := d.Review
	proposed := map[string]float64{}
	for _, row := range r.Rows {
		proposed[row.Slug] = row.Proposed
	}
	want := map[string]float64{
		"press":    95,  // made every PR set: +5
		"deadlift": 270, // missed one: reset 10%
		"bench":    210, // made them all and went 7 on the test: +10
		"squat":    260, // made them all: +10
	}
	for slug, v := range want {
		if proposed[slug] != v {
			t.Errorf("%s proposed %v, want %v", slug, proposed[slug], v)
		}
	}
}

// TestSeventhWeekDeload runs the optional deload rather than skipping it: the
// Forever percentages, a 3-5 range on the 80% set, and the block review after.
func TestSeventhWeekDeload(t *testing.T) {
	h := newHarness(t)
	h.setup(map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}, 4)

	page := h.dashboard()
	if !strings.Contains(page, "7th Week Deload") {
		t.Fatalf("expected the deload next:\n%s", page)
	}
	if !strings.Contains(page, "Skip it and start the next block") {
		t.Error("the deload should be skippable before it starts")
	}

	w := h.session(nil)
	sets, err := h.q.ListSets(context.Background(), w.ID)
	if err != nil {
		t.Fatal(err)
	}
	type main struct {
		weight        float64
		reps, maxReps int64
	}
	var got []main
	for _, s := range sets {
		if s.Kind == "main" {
			got = append(got, main{s.Weight, s.TargetReps, s.MaxReps})
		}
	}
	want := []main{{70, 5, 0}, {80, 3, 5}, {90, 1, 0}, {100, 1, 0}} // press at 100
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("deload main sets = %v, want %v", got, want)
	}

	// Once started, it can no longer be skipped.
	if strings.Contains(h.dashboard(), "Skip it and start the next block") {
		t.Error("skip offered after the deload started")
	}
	h.post("/skip-cycle", nil)
	for range 3 {
		h.session(nil)
	}

	// No anchor in this block, so the review holds every training max.
	page = h.dashboard()
	if !strings.Contains(page, "Block 2: new training maxes") || !strings.Contains(page, "No anchor PR sets recorded") {
		t.Errorf("expected a holding block review:\n%s", page)
	}
}

// TestSetupHistoryRepsLabel names the set whose reps onboarding asks for: the
// PR set on the anchor, the 100% set on the TM test, and nothing on a leader.
func TestSetupHistoryRepsLabel(t *testing.T) {
	h := newHarness(t)
	h.post("/setup", url.Values{"tm_press": {"100"}, "tm_deadlift": {"300"}, "tm_bench": {"200"}, "tm_squat": {"250"}})

	cases := []struct {
		cycle    string
		want     string
		unwanted []string
	}{
		{"2", "100% set reps", []string{"PR set reps"}},
		{"3", "PR set reps", []string{"100% set reps"}},
		{"0", "", []string{"PR set reps", "100% set reps"}},
	}
	for _, tc := range cases {
		page := h.do(http.MethodGet, "/setup/history?cycle="+tc.cycle, nil).Body.String()
		if tc.want != "" {
			mustContain(t, page, tc.want)
		}
		mustNotContain(t, page, tc.unwanted...)
	}
}
