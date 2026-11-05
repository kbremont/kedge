package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/kbremont/kedge/server/internal/auth"
	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/training"
)

// testNow is the harness's clock: the latest date a past estimate can have,
// and the backup's filename.
var testNow = time.Date(2026, 3, 16, 12, 0, 0, 0, time.UTC)

// harness drives the API, auth included, over an in-memory database. It
// reads what it needs from the API's answers, and the database only for what
// the API does not show.
type harness struct {
	t        *testing.T
	h        http.Handler
	conn     *sql.DB
	q        *db.Queries
	sessions *auth.Sessions
	token    string
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
	sessions := auth.New("test")
	svc := training.New(conn, log).WithClock(func() time.Time { return testNow })
	return &harness{
		t:        t,
		h:        New(svc, sessions, log, "test").Routes(),
		conn:     conn,
		q:        db.New(conn),
		sessions: sessions,
		token:    sessions.Sign(time.Now().Add(time.Hour)),
	}
}

// request sends body, if any, as JSON. A string body is sent as it is. The
// answer is checked against the spec.
func (h *harness) request(method, path string, body any, token string) *httptest.ResponseRecorder {
	h.t.Helper()
	var r io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		r = strings.NewReader(b)
	default:
		encoded, err := json.Marshal(b)
		if err != nil {
			h.t.Fatal(err)
		}
		r = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, r)
	if r != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		h.t.Errorf("%s %s: Cache-Control %q, want no-store", method, path, got)
	}
	checkAnswer(h.t, req, rec)
	return rec
}

func (h *harness) call(method, path string, body any) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.request(method, path, body, h.token)
}

// send makes a request that has to succeed, and decodes its answer.
func send[T any](h *harness, method, path string, body any) T {
	h.t.Helper()
	rec := h.call(method, path, body)
	if rec.Code != http.StatusOK {
		h.t.Fatalf("%s %s: %d %s", method, path, rec.Code, rec.Body.String())
	}
	return decodeAs[T](h.t, rec)
}

func get[T any](h *harness, path string) T {
	h.t.Helper()
	return send[T](h, http.MethodGet, path, nil)
}

func decodeAs[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type %q, want application/json", ct)
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return v
}

// fails makes a request that has to fail with status, and returns its
// message.
func (h *harness) fails(status int, method, path string, body any) string {
	h.t.Helper()
	rec := h.call(method, path, body)
	if rec.Code != status {
		h.t.Fatalf("%s %s: %d %s, want %d", method, path, rec.Code, rec.Body.String(), status)
	}
	e := decodeAs[ErrorResponse](h.t, rec)
	if e.Error == "" {
		h.t.Fatalf("%s %s: no error message in %s", method, path, rec.Body.String())
	}
	return e.Error
}

func (h *harness) dashboard() Dashboard {
	h.t.Helper()
	return get[Dashboard](h, "/api/v1/dashboard")
}

// setup enters training maxes and joins the block at cycleIdx with nothing
// in it done yet.
func (h *harness) setup(tms map[string]float64, cycleIdx int) {
	h.t.Helper()
	send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: tms})
	send[Dashboard](h, http.MethodPut, fmt.Sprintf("/api/v1/setup/history/%d", cycleIdx), SetupHistoryInput{Done: []PastSession{}})
}

var startingTMs = map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 250}

// session starts the next session, logs every set with the reps chosen for
// it, and finishes it. The sets are read from the session the API returns; the
// row says which lift and round it is.
func (h *harness) session(reps func(w db.Workout, set Set) int) db.Workout {
	h.t.Helper()
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	row, err := h.q.GetWorkout(context.Background(), w.ID)
	if err != nil {
		h.t.Fatal(err)
	}
	for _, set := range w.Sets {
		n := set.TargetReps
		if reps != nil {
			n = reps(row, set)
		}
		send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, set.Idx), LogSetInput{Reps: n})
	}
	send[Dashboard](h, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/finish", w.ID), nil)
	return row
}

// testSession runs the next TM test session, taking the 100% set for the reps
// given for its lift.
func (h *harness) testSession(reps map[string]int) db.Workout {
	h.t.Helper()
	return h.session(func(w db.Workout, set Set) int {
		if w.Phase == "tm_test" && set.Tag == "Main" && set.IsAMRAP {
			return reps[w.LiftSlug]
		}
		return set.TargetReps
	})
}

// decide settles a failed TM test.
func (h *harness) decide(w db.Workout, action TMTestAction, tm float64) Dashboard {
	h.t.Helper()
	return send[Dashboard](h, http.MethodPost, fmt.Sprintf("/api/v1/tm-tests/%d", w.ID), TMTestDecision{Action: action, TM: new(tm)})
}

// startHeld checks that starting a session is refused while a training max
// decision is waiting.
func (h *harness) startHeld() {
	h.t.Helper()
	h.fails(http.StatusConflict, http.MethodPost, "/api/v1/workouts", nil)
}

func tmsOf(d Dashboard) map[string]float64 {
	out := make(map[string]float64, len(d.Tms))
	for _, tm := range d.Tms {
		out[tm.Slug] = tm.Value
	}
	return out
}

func checkTMs(t *testing.T, when string, d Dashboard, want map[string]float64) {
	t.Helper()
	got := tmsOf(d)
	for slug, v := range want {
		if got[slug] != v {
			t.Errorf("%s, %s TM = %v, want %v", when, slug, got[slug], v)
		}
	}
}

// onlyResult returns the one TM test result on the dashboard.
func onlyResult(t *testing.T, d Dashboard) TMTestResult {
	t.Helper()
	if len(d.TestResults) != 1 {
		t.Fatalf("want one TM test result, got %+v", d.TestResults)
	}
	return d.TestResults[0]
}

// TestTMTestThroughToNextBlock walks the 7th week end to end through the API,
// as flow_test does through the pages: each TM test is reported as soon as it
// is done, only a failure offers to lower the training max, the anchor waits
// on that and runs on what was settled, skipping the deload ends the block,
// and the new block opens with a recommendation that nothing moves until
// accepted.
func TestTMTestThroughToNextBlock(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 2)

	d := h.dashboard()
	if d.Next == nil || d.Next.CycleLabel != "7th Week TM Test" || d.Next.Phase != "tm_test" || d.Next.RoundOfBlock != 7 || d.Next.RoundsInBlock != 11 {
		t.Fatalf("dashboard should offer the TM test as round 7 of 11: %+v", d.Next)
	}

	testReps := map[string]int{"press": 7, "deadlift": 4, "bench": 4, "squat": 2}

	h.testSession(testReps)
	d = h.dashboard()
	r := onlyResult(t, d)
	if r.LiftName != "Overhead Press" || r.Headline != "Strong set. This TM goes up at the end of the block." || r.Tone != "up" || r.NeedsDecision {
		t.Errorf("press result = %+v", r)
	}
	if d.Review != nil || d.Next.Blocked {
		t.Errorf("a passed test should hold nothing: review %+v, next %+v", d.Review, d.Next)
	}
	checkTMs(t, "after the press test", d, map[string]float64{"press": 100})

	h.testSession(testReps)
	r = onlyResult(t, h.dashboard()) // the press result is yesterday's news
	if r.LiftName != "Deadlift" || r.Headline != "Nice job, your TM is right." || r.Tone != "hold" {
		t.Errorf("deadlift result = %+v", r)
	}

	h.testSession(testReps)
	if r = onlyResult(t, h.dashboard()); r.LiftName != "Bench Press" || r.Tone != "hold" {
		t.Errorf("bench result = %+v", r)
	}

	// The 100% set counts toward the best estimated max: 200 x 4 by Epley.
	bench := get[Lift](h, "/api/v1/lifts/bench")
	if want := 200 * (1 + 4.0/30); math.Abs(bench.BestE1RM-want) > 0.01 {
		t.Errorf("bench best estimated max after the TM test = %v, want %v", bench.BestE1RM, want)
	}

	squat := h.testSession(testReps)
	d = h.dashboard()
	r = onlyResult(t, d)
	if r.LiftName != "Squat" || r.Tested != 250 || r.Reps != 2 || r.Headline != "Your TM is too high." ||
		r.Tone != "down" || !r.NeedsDecision || r.Current != 250 || r.Proposed != 225 {
		t.Errorf("squat result = %+v", r)
	}
	if !d.Next.Blocked {
		t.Error("the anchor should be blocked on the squat decision")
	}
	h.startHeld()

	d = h.decide(squat, "lower", 225)
	if len(d.TestResults) != 0 || d.Next.Blocked {
		t.Errorf("after lowering, results %+v and next %+v", d.TestResults, d.Next)
	}
	want := map[string]float64{"press": 100, "deadlift": 300, "bench": 200, "squat": 225}
	checkTMs(t, "after the TM test", d, want)

	// The anchor runs on the settled training maxes. Deadlift misses its 1+ in
	// the last round; everything else makes its minimum.
	for i := range 12 {
		w := h.session(func(w db.Workout, set Set) int {
			if set.IsAMRAP && w.LiftSlug == "deadlift" && w.Round == 3 {
				return 0
			}
			if set.IsAMRAP {
				return set.TargetReps + 2
			}
			return set.TargetReps
		})
		if w.Phase != "anchor" {
			t.Fatalf("session %d of the anchor was %q", i, w.Phase)
		}
		if w.TrainingMax != want[w.LiftSlug] {
			t.Errorf("anchor %s ran at %v, want %v", w.LiftSlug, w.TrainingMax, want[w.LiftSlug])
		}
	}

	// The deload is offered, and skipped.
	d = h.dashboard()
	if d.Next == nil || d.Next.CycleLabel != "7th Week Deload" || !d.Next.CanSkip {
		t.Fatalf("dashboard should offer to skip the deload: %+v", d.Next)
	}
	d = send[Dashboard](h, http.MethodPost, "/api/v1/skip-cycle", nil)

	// The new block opens on its review, and nothing has moved yet.
	if d.Review == nil || d.Review.Title != "Block 2: new training maxes" {
		t.Fatalf("want the block 2 review, got %+v", d.Review)
	}
	verdicts := map[string]string{}
	for _, row := range d.Review.Rows {
		verdicts[row.Slug] = row.Verdict
	}
	wantVerdicts := map[string]string{
		"press":    "PR sets and TM test: +10 lb", // every PR set, and 7 on the test
		"deadlift": "Missed a PR set: reset 10%",
		"bench":    "Hit every PR set: +5 lb",
		"squat":    "Hit every PR set: +10 lb",
	}
	for slug, v := range wantVerdicts {
		if verdicts[slug] != v {
			t.Errorf("%s verdict = %q, want %q", slug, verdicts[slug], v)
		}
	}
	if d.Next.CycleLabel != "Leader 1 - Round 1" || !d.Next.Blocked {
		t.Errorf("block 2 should open on a blocked Leader 1 - Round 1: %+v", d.Next)
	}
	checkTMs(t, "before the review was accepted", d, map[string]float64{"press": 100})
	h.startHeld()

	// Accept, editing squat below what was proposed.
	want = map[string]float64{"press": 110, "deadlift": 270, "bench": 205, "squat": 230}
	d = send[Dashboard](h, http.MethodPost, "/api/v1/tm-review", TMReviewInput{Key: d.Review.Key, Tms: want})
	if d.Review != nil {
		t.Errorf("review still pending after accepting it: %+v", d.Review)
	}
	checkTMs(t, "after the block review", d, want)

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
	h.setup(startingTMs, 2)

	testReps := map[string]int{"press": 1, "deadlift": 4, "bench": 4, "squat": 2}
	press := h.testSession(testReps)
	d := h.dashboard()
	if r := onlyResult(t, d); r.LiftName != "Overhead Press" || !r.NeedsDecision {
		t.Errorf("press result = %+v", r)
	}
	if d.Next.Blocked {
		t.Fatal("a failed press test should not hold the rest of the TM test week")
	}
	// Starting begins deadlift's test; abandon it, then run the rest.
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	if w.LiftName != "Deadlift" || w.Phase != "tm_test" {
		t.Fatalf("started %s %s, want deadlift's TM test", w.LiftName, w.Phase)
	}
	d = send[Dashboard](h, http.MethodDelete, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil)
	if d.Next == nil || d.Next.InProgress {
		t.Errorf("abandoned session still in progress: %+v", d.Next)
	}
	h.testSession(testReps)
	h.testSession(testReps)
	squat := h.testSession(testReps)

	// Both failures stay up until settled, and the anchor waits on them.
	if d = h.dashboard(); len(d.TestResults) != 2 || !d.Next.Blocked {
		t.Fatalf("want press and squat waiting and the anchor blocked: %+v, %+v", d.TestResults, d.Next)
	}
	h.startHeld()

	h.decide(press, "keep", 0)
	h.startHeld()

	// A value above the training max is not a raise; it keeps it.
	d = h.decide(squat, "lower", 400)
	checkTMs(t, "after keeping both", d, map[string]float64{"press": 100, "squat": 250})
	var notes int
	if err := h.conn.QueryRow(`SELECT COUNT(*) FROM training_maxes WHERE note = 'tm test'`).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if notes != 0 {
		t.Errorf("%d training max entries written for kept TMs, want 0", notes)
	}
	if len(d.TestResults) != 0 || d.Next.Blocked {
		t.Errorf("after both decisions, results %+v and next %+v", d.TestResults, d.Next)
	}

	// A decision already made is no longer on offer.
	h.fails(http.StatusConflict, http.MethodPost, fmt.Sprintf("/api/v1/tm-tests/%d", press.ID), TMTestDecision{Action: "keep"})

	got := h.session(nil)
	if got.Phase != "anchor" || got.TrainingMax != 100 {
		t.Errorf("first anchor session = %s at %v, want anchor press at 100", got.Phase, got.TrainingMax)
	}
}

// TestBlockReviewProposals checks the numbers the block review puts in front
// of the user, before anything is accepted.
func TestBlockReviewProposals(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 2)

	testReps := map[string]int{"press": 2, "deadlift": 4, "bench": 7, "squat": 5}
	press := h.testSession(testReps)
	for range 3 {
		h.testSession(testReps)
	}
	h.decide(press, "lower", 90)
	for range 12 {
		h.session(func(w db.Workout, set Set) int {
			if set.IsAMRAP && w.LiftSlug == "deadlift" && w.Round == 3 {
				return 0
			}
			return set.TargetReps
		})
	}
	d := send[Dashboard](h, http.MethodPost, "/api/v1/skip-cycle", nil)

	if d.Review == nil {
		t.Fatal("want a block review")
	}
	proposed := map[string]float64{}
	for _, row := range d.Review.Rows {
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

	// Leaving a lift out accepts nothing.
	msg := h.fails(http.StatusBadRequest, http.MethodPost, "/api/v1/tm-review",
		TMReviewInput{Key: d.Review.Key, Tms: map[string]float64{"press": 95, "deadlift": 270, "bench": 210}})
	if msg != "Enter a positive training max for Squat." {
		t.Errorf("message = %q", msg)
	}
	if h.dashboard().Review == nil {
		t.Error("a rejected review should still be pending")
	}
	// Nor does a key for some other review.
	h.fails(http.StatusConflict, http.MethodPost, "/api/v1/tm-review", TMReviewInput{Key: "b1:c0", Tms: want})
}

// TestSeventhWeekDeload runs the optional deload rather than skipping it: the
// Forever percentages, a 3-5 range on the 80% set, and the block review after.
func TestSeventhWeekDeload(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 4)

	d := h.dashboard()
	if d.Next == nil || d.Next.CycleLabel != "7th Week Deload" || d.Next.Phase != "deload" {
		t.Fatalf("expected the deload next: %+v", d.Next)
	}
	if !d.Next.CanSkip {
		t.Error("the deload should be skippable before it starts")
	}

	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	type main struct {
		weight             float64
		reps, maxReps      int
		repTarget          string
		asksReps           bool
		choices, targetsAt string
	}
	var got []main
	for _, s := range w.Sets {
		if s.Tag != "Main" {
			continue
		}
		var choices, targets []string
		for _, c := range s.RepChoices {
			choices = append(choices, fmt.Sprint(c.Reps))
			if c.Target {
				targets = append(targets, fmt.Sprint(c.Reps))
			}
		}
		got = append(got, main{s.Weight, s.TargetReps, s.MaxReps, s.RepTarget, s.AsksReps, strings.Join(choices, " "), strings.Join(targets, " ")})
	}
	want := []main{ // press at 100
		{70, 5, 0, "5", false, "", ""},
		{80, 3, 5, "3–5", true, "1 2 3 4 5 6 7 8 9 10", "3 4 5"},
		{90, 1, 0, "1", false, "", ""},
		{100, 1, 0, "1", false, "", ""},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("deload main sets =\n%v\nwant\n%v", got, want)
	}

	// Once started, it can no longer be skipped.
	if d = h.dashboard(); d.Next.CanSkip {
		t.Error("skip offered after the deload started")
	}
	h.fails(http.StatusConflict, http.MethodPost, "/api/v1/skip-cycle", nil)
	for _, s := range w.Sets {
		send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, s.Idx), LogSetInput{Reps: s.TargetReps})
	}
	send[Dashboard](h, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/finish", w.ID), nil)
	if d = h.dashboard(); d.Next.CanSkip {
		t.Error("skip offered after the first deload session")
	}
	h.fails(http.StatusConflict, http.MethodPost, "/api/v1/skip-cycle", nil)
	for range 3 {
		h.session(nil)
	}

	// No anchor in this block, so the review holds every training max.
	d = h.dashboard()
	if d.Review == nil || d.Review.Title != "Block 2: new training maxes" {
		t.Fatalf("expected the block 2 review: %+v", d.Review)
	}
	for _, row := range d.Review.Rows {
		if row.Evidence != "No anchor PR sets recorded" || row.Tone != "hold" || row.Proposed != row.Current {
			t.Errorf("%s review row = %+v, want a hold with no anchor evidence", row.Slug, row)
		}
	}
}

// TestSetupHistoryRepsLabel names the set whose reps onboarding asks for: the
// PR set on the anchor, the 100% set on the TM test, and nothing on a leader.
func TestSetupHistoryRepsLabel(t *testing.T) {
	h := newHarness(t)
	send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})

	cases := []struct {
		cycle    int
		asksReps bool
		label    string
	}{
		{2, true, "100% set reps"},
		{3, true, "PR set reps"},
		{0, false, ""},
	}
	for _, tc := range cases {
		hist := get[SetupHistory](h, fmt.Sprintf("/api/v1/setup/history/%d", tc.cycle))
		if hist.AsksReps != tc.asksReps || hist.RepsLabel != tc.label {
			t.Errorf("cycle %d asks reps %v labelled %q, want %v %q", tc.cycle, hist.AsksReps, hist.RepsLabel, tc.asksReps, tc.label)
		}
	}
}

// TestSetupJoinsPartway records two anchor sessions as done, with the PR set
// reps for one, and picks up at the third.
func TestSetupJoinsPartway(t *testing.T) {
	h := newHarness(t)
	if d := h.dashboard(); !d.NeedsSetup || d.Next != nil || d.Tms == nil || d.Recent == nil || d.TestResults == nil {
		t.Fatalf("a new database should need setup, with empty lists: %+v", d)
	}
	pos := send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
	if len(pos.Cycles) != 5 || pos.Cycles[3].Label != "Anchor" {
		t.Fatalf("position = %+v", pos)
	}
	d := send[Dashboard](h, http.MethodPut, "/api/v1/setup/history/3", SetupHistoryInput{Done: []PastSession{
		{Round: 1, Slug: "press", PRReps: new(9)},
		{Round: 1, Slug: "deadlift"},
	}})
	if d.NeedsSetup || d.Next == nil || d.Next.LiftSlug != "bench" || d.Next.CycleLabel != "Anchor - Round 1" {
		t.Fatalf("want bench's first anchor session next: %+v", d.Next)
	}
	if len(d.Recent) != 2 || d.Recent[1].Summary != "85 x 9" {
		t.Errorf("recent = %+v, want the press PR set as 85 x 9", d.Recent)
	}
}

func TestAuth(t *testing.T) {
	h := newHarness(t)

	info := decodeAs[Info](t, h.request(http.MethodGet, "/api/v1/info", nil, ""))
	if info.APIVersion != 1 || info.Version != "test" {
		t.Errorf("info = %+v", info)
	}

	for name, token := range map[string]string{"no token": "", "a bad token": "1.nope", "an expired token": h.sessions.Sign(time.Now().Add(-time.Minute))} {
		rec := h.request(http.MethodGet, "/api/v1/dashboard", nil, token)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: %d, want 401", name, rec.Code)
			continue
		}
		if e := decodeAs[ErrorResponse](t, rec); e.Error == "" {
			t.Errorf("%s: no error message", name)
		}
	}
	// An unknown path is a 401 too until signed in, rather than a hint at
	// what exists.
	if rec := h.request(http.MethodGet, "/api/v1/nope", nil, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("unknown path without a token: %d, want 401", rec.Code)
	}

	rec := h.request(http.MethodPost, "/api/v1/login", LoginRequest{Password: "wrong"}, "")
	if rec.Code != http.StatusUnauthorized || decodeAs[ErrorResponse](t, rec).Error != "Incorrect password." {
		t.Errorf("wrong password: %d %s", rec.Code, rec.Body.String())
	}

	rec = h.request(http.MethodPost, "/api/v1/login", LoginRequest{Password: "test"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	session := decodeAs[LoginResponse](t, rec)
	expires, err := time.Parse(time.RFC3339, session.ExpiresAt)
	if err != nil || expires.Before(time.Now().Add(auth.TTL-time.Hour)) {
		t.Errorf("expiresAt %q: %v", session.ExpiresAt, err)
	}
	if rec := h.request(http.MethodGet, "/api/v1/dashboard", nil, session.Token); rec.Code != http.StatusOK {
		t.Errorf("dashboard with the token from login: %d", rec.Code)
	}

	// The browser's cookie carries the same token.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session.Token})
	rec = httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("dashboard with the session cookie: %d", rec.Code)
	}
}

// TestInputErrors sends what the forms would refuse: each is a 400 with
// training's message, and nothing is written.
func TestInputErrors(t *testing.T) {
	h := newHarness(t)

	msg := h.fails(http.StatusBadRequest, http.MethodPut, "/api/v1/setup", SetupInput{Tms: map[string]float64{"press": 100, "deadlift": 300, "bench": 200}})
	if msg != "Enter a positive training max for Squat." {
		t.Errorf("setup with squat missing: %q", msg)
	}
	if !h.dashboard().NeedsSetup {
		t.Error("a rejected setup wrote training maxes")
	}

	h.setup(startingTMs, 2)
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	msg = h.fails(http.StatusBadRequest, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/0", w.ID), LogSetInput{Reps: 501})
	if msg != "Reps must be between 0 and 500." {
		t.Errorf("501 reps: %q", msg)
	}
	if w = get[Workout](h, fmt.Sprintf("/api/v1/workouts/%d", w.ID)); w.Sets[0].Done {
		t.Error("501 reps logged the set")
	}

	// Reps left out, or sent under the wrong name, are not a failed set: 0
	// would count as a missed PR. The spec requires them.
	for _, body := range []any{struct{}{}, map[string]int{"rep": 5}, map[string]any{"reps": nil}} {
		h.fails(http.StatusBadRequest, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/0", w.ID), body)
	}
	if w = get[Workout](h, fmt.Sprintf("/api/v1/workouts/%d", w.ID)); w.Sets[0].Done {
		t.Error("a log without reps logged the set")
	}

	msg = h.fails(http.StatusBadRequest, http.MethodPost, "/api/v1/lifts/press/estimates", PastEstimateInput{Date: "2026-03-17", Weight: 120, Reps: 1})
	if msg != "That estimate was not saved. Check the date, weight and reps." {
		t.Errorf("estimate dated tomorrow: %q", msg)
	}
	if l := get[Lift](h, "/api/v1/lifts/press"); len(l.PastEstimates) != 0 || l.Today != "2026-03-16" {
		t.Errorf("after a rejected estimate, past estimates %+v and today %q", l.PastEstimates, l.Today)
	}

	msg = h.fails(http.StatusBadRequest, http.MethodPut, "/api/v1/settings", SettingsInput{Plates: []float64{0, -5}, Rounding: 2.5, BBB: map[string]BBBInput{}})
	if msg != "List at least one plate." {
		t.Errorf("plates with none positive: %q", msg)
	}
	// A BBB mode the spec does not list is refused, not read as the default.
	h.fails(http.StatusBadRequest, http.MethodPut, "/api/v1/settings", SettingsInput{
		Plates:   []float64{45, 25},
		Rounding: 2.5,
		BBB:      map[string]BBBInput{"press": {Mode: "heavy", Value: 55}},
	})
	if s := get[Settings](h, "/api/v1/settings"); s.Rounding != 5 || len(s.Plates) != 6 {
		t.Errorf("a rejected settings change wrote rounding %v and plates %v", s.Rounding, s.Plates)
	}

	msg = h.fails(http.StatusBadRequest, http.MethodPost, "/api/v1/training-maxes", TrainingMaxInput{Lift: "press", Value: 0})
	if msg != "Enter a positive training max." {
		t.Errorf("a zero training max: %q", msg)
	}

	// Lowering a failed test's training max to nothing.
	send[Dashboard](h, http.MethodDelete, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil)
	press := h.testSession(map[string]int{"press": 1})
	msg = h.fails(http.StatusBadRequest, http.MethodPost, fmt.Sprintf("/api/v1/tm-tests/%d", press.ID), TMTestDecision{Action: "lower", TM: new(0.0)})
	if msg != "Enter a positive training max." {
		t.Errorf("lowering to 0: %q", msg)
	}
	// Nor is an action that is neither keep nor lower read as lowering.
	h.fails(http.StatusBadRequest, http.MethodPost, fmt.Sprintf("/api/v1/tm-tests/%d", press.ID), TMTestDecision{Action: "drop", TM: new(90.0)})
	if d := h.dashboard(); len(d.TestResults) != 1 || !d.TestResults[0].NeedsDecision || tmsOf(d)["press"] != 100 {
		t.Errorf("a rejected decision changed something: %+v", d.TestResults)
	}
}

func TestNotFound(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/workouts/99"},
		{http.MethodGet, "/api/v1/workouts/abc"},
		{http.MethodPut, "/api/v1/workouts/99/sets/0"},
		{http.MethodGet, "/api/v1/lifts/curl"},
		{http.MethodPost, "/api/v1/lifts/curl/estimates"},
		{http.MethodDelete, "/api/v1/lifts/press/estimates/abc"},
		{http.MethodGet, "/api/v1/setup/history/9"},
		{http.MethodGet, "/api/v1/setup/history/x"},
		{http.MethodPost, "/api/v1/tm-tests/abc"},
		{http.MethodGet, "/api/v1/nope"},
		{http.MethodGet, "/api/v2/dashboard"},
	} {
		var body any
		switch tc.method {
		case http.MethodPut:
			body = LogSetInput{Reps: 5}
		case http.MethodPost:
			body = PastEstimateInput{Date: "2025-01-01", Weight: 100, Reps: 1}
		}
		if msg := h.fails(http.StatusNotFound, tc.method, tc.path, body); msg != "Not found." {
			t.Errorf("%s %s: %q", tc.method, tc.path, msg)
		}
	}

	// An unknown lift's training max is a 404, not a foreign key failure.
	h.fails(http.StatusNotFound, http.MethodPost, "/api/v1/training-maxes", TrainingMaxInput{Lift: "curl", Value: 100})

	// A route asked for with a method it does not have is a 405, which says
	// what it does have.
	rec := h.call(http.MethodPost, "/api/v1/dashboard", nil)
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET" {
		t.Errorf("POST dashboard: %d allowing %q, want 405 allowing GET", rec.Code, rec.Header().Get("Allow"))
	}
	if rec = h.call(http.MethodPatch, "/api/v1/settings", nil); rec.Header().Get("Allow") != "GET, PUT" {
		t.Errorf("PATCH settings: %d allowing %q, want 405 allowing GET, PUT", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestMalformedJSON(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)

	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPut, "/api/v1/settings", "{"},
		{http.MethodPut, "/api/v1/setup", `{"tms": {"press": "heavy"}}`},
		{http.MethodPost, "/api/v1/lifts/press/estimates", ""},
		{http.MethodPost, "/api/v1/training-maxes", "[1, 2]"},
	} {
		h.fails(http.StatusBadRequest, tc.method, tc.path, tc.body)
	}
	rec := h.request(http.MethodPost, "/api/v1/login", "password=test", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("a form-encoded login: %d, want 400", rec.Code)
	}

	// Fields the server does not know are ignored, for an app newer than it.
	if s := send[Settings](h, http.MethodPost, "/api/v1/training-maxes", `{"lift": "press", "value": 105, "note": "", "unit": "kg"}`); s.Tms[0].Value != 105 {
		t.Errorf("press TM = %v, want 105", s.Tms[0].Value)
	}
}

func TestBackup(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)

	rec := h.call(http.MethodGet, "/api/v1/backup", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("backup: %d %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="531-2026-03-16.db"` {
		t.Errorf("Content-Disposition %q", cd)
	}
	if !strings.HasPrefix(rec.Body.String(), "SQLite format 3\x00") {
		t.Errorf("backup does not start with the SQLite header: %q", rec.Body.String()[:min(16, rec.Body.Len())])
	}
}

// TestLogSetAnswers checks what logging a set answers with: the set done, the
// next one current, and the rest for the kind of set just logged, which the
// app starts its timer from.
func TestLogSetAnswers(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)
	send[Settings](h, http.MethodPut, "/api/v1/settings", SettingsInput{
		Plates:            []float64{45, 25, 10, 5, 2.5},
		Warmups:           true,
		RestWarmupSeconds: 60,
		RestMainSeconds:   180,
		RestSuppSeconds:   120,
		BBB:               map[string]BBBInput{},
	})

	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	if w.CurrentIdx != 0 || w.Completed {
		t.Fatalf("a new session: current %d, completed %v", w.CurrentIdx, w.Completed)
	}
	if again := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil); again.ID != w.ID {
		t.Errorf("starting again began session %d, want %d resumed", again.ID, w.ID)
	}

	first := map[SetTag]int{}
	for i, s := range w.Sets {
		if _, ok := first[s.Tag]; !ok {
			first[s.Tag] = i
		}
	}
	for _, tc := range []struct {
		tag  SetTag
		rest int
	}{
		{"Warmup", 60},
		{"Main", 180},
		{"BBB", 120},
	} {
		i, ok := first[tc.tag]
		if !ok {
			t.Fatalf("no %s set in %+v", tc.tag, w.Sets)
		}
		idx := w.Sets[i].Idx
		got := send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, idx), LogSetInput{Reps: 4})
		set := got.Sets[i]
		if !set.Done || set.ActualReps != 4 {
			t.Errorf("%s set after logging: %+v", tc.tag, set)
		}
		if got.RestSeconds != tc.rest {
			t.Errorf("rest after a %s set = %d, want %d", tc.tag, got.RestSeconds, tc.rest)
		}
		if tc.tag == "Warmup" && got.CurrentIdx != idx+1 {
			t.Errorf("current set after the first = %d, want %d", got.CurrentIdx, idx+1)
		}
	}

	got := send[Workout](h, http.MethodDelete, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, w.Sets[0].Idx), nil)
	if got.Sets[0].Done || got.CurrentIdx != w.Sets[0].Idx {
		t.Errorf("after undo: set %+v, current %d", got.Sets[0], got.CurrentIdx)
	}

	d := send[Dashboard](h, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/finish", w.ID), nil)
	if len(d.Recent) != 1 || d.Recent[0].WorkoutID != w.ID {
		t.Errorf("recent after finishing = %+v", d.Recent)
	}
	if got = get[Workout](h, fmt.Sprintf("/api/v1/workouts/%d", w.ID)); !got.Completed {
		t.Error("a finished session is not marked completed")
	}
}

// actualWeight reads what a set stored as its changed weight, NULL if none.
func (h *harness) actualWeight(workout int64, idx int) sql.NullFloat64 {
	h.t.Helper()
	var got sql.NullFloat64
	if err := h.conn.QueryRow(`SELECT actual_weight FROM sets WHERE workout_id = ? AND idx = ?`, workout, idx).Scan(&got); err != nil {
		h.t.Fatalf("read actual_weight: %v", err)
	}
	return got
}

// TestLogSetWeight logs a set at a weight other than the one suggested. The
// session answers with that weight in every figure, and undoing the set goes
// back to the suggestion.
func TestLogSetWeight(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	i := slices.IndexFunc(w.Sets, firstMain)
	suggested := w.Sets[i]
	path := fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, suggested.Idx)
	if !suggested.WeightEditable {
		t.Fatalf("a plain set's weight isn't editable: %+v", suggested)
	}

	changed := suggested.Weight + 10
	got := send[Workout](h, http.MethodPut, path, LogSetInput{Reps: 4, Weight: &changed}).Sets[i]
	if got.Weight != changed || got.ActualReps != 4 || !got.Done {
		t.Errorf("logged at %v: %+v", changed, got)
	}
	if want := changed / w.TrainingMax; got.Pct != want {
		t.Errorf("pct = %v, want %v", got.Pct, want)
	}
	if got.Plates == suggested.Plates {
		t.Errorf("plates stayed %q for %v", got.Plates, changed)
	}
	if stored := h.actualWeight(w.ID, suggested.Idx); stored.Float64 != changed {
		t.Errorf("actual_weight = %+v", stored)
	}

	back := send[Workout](h, http.MethodDelete, path, nil).Sets[i]
	if back.Weight != suggested.Weight || back.Pct != suggested.Pct || back.Plates != suggested.Plates {
		t.Errorf("undone: %+v, want the suggestion %+v", back, suggested)
	}
	if stored := h.actualWeight(w.ID, suggested.Idx); stored.Valid {
		t.Errorf("undo left actual_weight = %+v", stored)
	}

	// The suggestion sent back is no change, and neither is leaving it out.
	same := suggested.Weight
	for _, body := range []LogSetInput{{Reps: 5, Weight: &same}, {Reps: 5}} {
		if got := send[Workout](h, http.MethodPut, path, body).Sets[i]; got.Weight != suggested.Weight {
			t.Errorf("%+v: weight %v", body, got.Weight)
		}
		if stored := h.actualWeight(w.ID, suggested.Idx); stored.Valid {
			t.Errorf("%+v stored actual_weight = %+v", body, stored)
		}
	}
	send[Workout](h, http.MethodDelete, path, nil)

	for _, bad := range []float64{0, -5, 0.5, 2000.5} {
		msg := h.fails(http.StatusBadRequest, http.MethodPut, path, LogSetInput{Reps: 5, Weight: &bad})
		if msg != "Weight must be between 1 and 2000." {
			t.Errorf("weight %v: %q", bad, msg)
		}
	}
	if got := get[Workout](h, fmt.Sprintf("/api/v1/workouts/%d", w.ID)).Sets[i]; got.Done {
		t.Error("a refused weight logged the set")
	}

	h.fails(http.StatusNotFound, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/99", w.ID), LogSetInput{Reps: 5, Weight: &changed})
}

// TestPRSetWeightIsLocked keeps a PR set or TM test at its suggested weight,
// since the verdicts read its reps as done at that weight. Its reps still log.
func TestPRSetWeightIsLocked(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cycle int
	}{
		{"TM test", 2},
		{"anchor PR set", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.setup(startingTMs, tc.cycle)
			w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
			i := slices.IndexFunc(w.Sets, func(s Set) bool { return s.IsAMRAP })
			if i < 0 {
				t.Fatalf("no PR set in %+v", w.Sets)
			}
			set := w.Sets[i]
			if set.WeightEditable {
				t.Errorf("weight is editable: %+v", set)
			}
			path := fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, set.Idx)

			lighter := set.Weight - 10
			msg := h.fails(http.StatusBadRequest, http.MethodPut, path, LogSetInput{Reps: 8, Weight: &lighter})
			if msg != "The weight of a PR set or TM test can't be changed." {
				t.Errorf("refusal: %q", msg)
			}
			if got := get[Workout](h, fmt.Sprintf("/api/v1/workouts/%d", w.ID)).Sets[i]; got.Done {
				t.Error("a refused weight logged the set")
			}

			same := set.Weight
			if got := send[Workout](h, http.MethodPut, path, LogSetInput{Reps: 8, Weight: &same}).Sets[i]; !got.Done || got.ActualReps != 8 {
				t.Errorf("logged at its own weight: %+v", got)
			}
		})
	}
}

// TestRangeSetWeightIsEditable: the deload's 3-5 set asks for its reps but no
// verdict reads it, so its weight can change.
func TestRangeSetWeightIsEditable(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 4)
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	i := slices.IndexFunc(w.Sets, func(s Set) bool { return s.AsksReps })
	if i < 0 {
		t.Fatalf("no set asks for its reps in %+v", w.Sets)
	}
	if !w.Sets[i].WeightEditable || w.Sets[i].IsAMRAP {
		t.Fatalf("range set: %+v", w.Sets[i])
	}
	changed := w.Sets[i].Weight - 5
	got := send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, w.Sets[i].Idx), LogSetInput{Reps: 3, Weight: &changed})
	if got.Sets[i].Weight != changed {
		t.Errorf("logged at %v: %+v", changed, got.Sets[i])
	}
}

// TestRecentSummaryUsesLiftedWeight: a session is remembered by what was
// lifted, in the heaviest main set's place and its figures.
func TestRecentSummaryUsesLiftedWeight(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	var mains []Set
	for _, s := range w.Sets {
		if s.Tag == "Main" {
			mains = append(mains, s)
		}
	}
	if len(mains) < 2 {
		t.Fatalf("main sets: %+v", mains)
	}
	first := mains[0]
	heaviest := mains[len(mains)-1].Weight + 15
	for _, s := range w.Sets {
		body := LogSetInput{Reps: s.TargetReps}
		if s.Idx == first.Idx {
			body = LogSetInput{Reps: 4, Weight: &heaviest}
		}
		send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, s.Idx), body)
	}
	d := send[Dashboard](h, http.MethodPost, fmt.Sprintf("/api/v1/workouts/%d/finish", w.ID), nil)
	want := fmt.Sprintf("%s x 4", training.FormatWeight(heaviest))
	if len(d.Recent) != 1 || d.Recent[0].Summary != want {
		t.Errorf("recent = %+v, want summary %q", d.Recent, want)
	}
}

// TestPlates loads the bar for a weight being typed, by the same math and
// settings as a suggestion, and writes nothing.
func TestPlates(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)
	send[Settings](h, http.MethodPut, "/api/v1/settings", SettingsInput{Plates: []float64{45, 25, 10}, Warmups: true, BBB: map[string]BBBInput{}})

	got := get[Plates](h, "/api/v1/plates?weight=185")
	if got.Plates != "45, 25" || got.PlateLine != "Bar + 45, 25 per side" || got.Unloadable != 0 {
		t.Errorf("185: %+v", got)
	}
	if got = get[Plates](h, "/api/v1/plates?weight=45"); got.Plates != "" || got.PlateLine != "Empty bar" {
		t.Errorf("empty bar: %+v", got)
	}
	if got = get[Plates](h, "/api/v1/plates?weight=190"); got.Unloadable != 5 {
		t.Errorf("190 with no 2.5s: %+v", got)
	}

	if msg := h.fails(http.StatusBadRequest, http.MethodGet, "/api/v1/plates?weight=2001", nil); msg != "Weight must be between 1 and 2000." {
		t.Errorf("too heavy: %q", msg)
	}
	h.fails(http.StatusBadRequest, http.MethodGet, "/api/v1/plates", nil)
	h.fails(http.StatusBadRequest, http.MethodGet, "/api/v1/plates?weight=lots", nil)
}

// TestPastEstimates adds an estimate and deletes it, each answering with the
// lift as it now is.
func TestPastEstimates(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)

	l := send[Lift](h, http.MethodPost, "/api/v1/lifts/press/estimates", PastEstimateInput{Date: "2025-03-03", Weight: 100, Reps: 6})
	if len(l.PastEstimates) != 1 || l.PastEstimates[0].Date != "Mar 3, 2025" || l.PastEstimates[0].Estimate != 120 {
		t.Fatalf("past estimates = %+v", l.PastEstimates)
	}
	if len(l.Chart.Points) != 1 || !l.Chart.Points[0].Past || l.Chart.Points[0].Label != "120" || l.BestE1RM != 120 {
		t.Errorf("chart = %+v, best %v", l.Chart, l.BestE1RM)
	}
	if l.Chart.Width != 320 || l.Chart.Height != 160 {
		t.Errorf("chart size = %vx%v, want 320x160", l.Chart.Width, l.Chart.Height)
	}

	l = send[Lift](h, http.MethodDelete, fmt.Sprintf("/api/v1/lifts/press/estimates/%d", l.PastEstimates[0].ID), nil)
	if len(l.PastEstimates) != 0 || len(l.Chart.Points) != 0 || l.Chart.Points == nil {
		t.Errorf("after deleting, past estimates %+v and points %#v", l.PastEstimates, l.Chart.Points)
	}
}

// TestSettingsRoundTrip saves every setting, including each BBB mode, and
// leaves one lift out so it goes back to the program's default.
func TestSettingsRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.setup(startingTMs, 0)

	s := send[Settings](h, http.MethodPut, "/api/v1/settings", SettingsInput{
		Rounding:          2.5,
		BarWeight:         35,
		Plates:            []float64{5, 45, 0, 25},
		Warmups:           false,
		RestWarmupSeconds: 30,
		RestMainSeconds:   0, // keeps the current 90
		RestSuppSeconds:   75,
		BBB: map[string]BBBInput{
			"press":    {Mode: "percent", Value: 55},
			"deadlift": {Mode: "weight", Value: 185},
			"bench":    {Mode: "percent", Value: 0}, // not positive: the program's own
		},
	})
	if s.Rounding != 2.5 || s.BarWeight != 35 || fmt.Sprint(s.Plates) != "[45 25 5]" || s.Warmups ||
		s.RestWarmupSeconds != 30 || s.RestMainSeconds != 90 || s.RestSuppSeconds != 75 {
		t.Errorf("settings = %+v", s)
	}
	got := map[string]string{}
	for _, b := range s.BBB {
		got[b.Slug] = fmt.Sprintf("%s %v", b.Mode, b.Value)
	}
	want := map[string]string{"press": "percent 55", "deadlift": "weight 185", "bench": "program 0", "squat": "program 0"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("BBB = %v, want %v", got, want)
	}

	s = send[Settings](h, http.MethodPost, "/api/v1/training-maxes", TrainingMaxInput{Lift: "squat", Value: 255, Note: new("felt good")})
	if s.Tms[3].Slug != "squat" || s.Tms[3].Value != 255 {
		t.Errorf("TMs = %+v", s.Tms)
	}
	l := get[Lift](h, "/api/v1/lifts/squat")
	if len(l.TMHistory) != 2 || l.TMHistory[0].Value != 255 || l.TMHistory[0].Note != "felt good" {
		t.Errorf("squat TM history = %+v", l.TMHistory)
	}
}

// TestSetupUnit takes a unit in setup until anything is logged in it, and
// refuses one it doesn't know.
func TestSetupUnit(t *testing.T) {
	h := newHarness(t)
	if got := get[Setup](h, "/api/v1/setup"); got.Unit != "lb" || got.CanChangeUnit == nil || !*got.CanChangeUnit {
		t.Fatalf("fresh setup = %+v, want lb that can change", got)
	}
	h.fails(http.StatusBadRequest, http.MethodPut, "/api/v1/setup", map[string]any{"tms": startingTMs, "unit": "stone"})

	kg := SetupInputUnitKg
	send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: kgTMs, Unit: &kg})
	if got := get[Settings](h, "/api/v1/settings"); got.Unit != "kg" || got.BarWeight != 20 {
		t.Errorf("settings = %s with a %v bar, want kg and 20", got.Unit, got.BarWeight)
	}

	send[Dashboard](h, http.MethodPut, "/api/v1/setup/history/0", SetupHistoryInput{Done: []PastSession{{Round: 1, Slug: "press", PRReps: new(8)}}})
	if got := get[Setup](h, "/api/v1/setup"); got.Unit != "kg" || *got.CanChangeUnit {
		t.Errorf("setup after a session = %+v, want kg that can't change", got)
	}
	lb := SetupInputUnitLb
	if msg := h.fails(http.StatusBadRequest, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs, Unit: &lb}); msg != "The unit can't change once training is logged in kg." {
		t.Errorf("change after a session: %q", msg)
	}
	// An app from before kilograms sends no unit, and keeps the one set.
	send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: kgTMs})
}
