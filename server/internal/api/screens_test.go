package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kbremont/531-tracker/server/internal/db"
)

// The golden responses: one per screen state the app's gallery shows, kept in
// testdata/ and copied into the app as fixtures by `make generate`. They pin
// the wire format, so a change to it shows up as a diff here and in the app.
//
//	go test ./internal/api -update
//
// rewrites them.

var update = flag.Bool("update", false, "rewrite the golden responses in testdata/")

func TestMain(m *testing.M) {
	// Dates are formatted in the server's zone; pin it so the goldens do not
	// depend on the machine.
	time.Local = time.UTC
	os.Exit(m.Run())
}

// pinTimestamps rewrites every timestamp the database wrote with
// datetime('now') to a fixed date plus an offset from the row id: sessions two
// days apart, their sets a few minutes apart, training maxes and blocks a day
// apart. Rows keep the order they were written in, and the screens get dates
// that look like a training log.
func (h *harness) pinTimestamps() {
	h.t.Helper()
	const base = "2026-01-05 18:00:00"
	for _, stmt := range []string{
		`UPDATE workouts SET
			started_at = datetime('` + base + `', '+' || (id * 2) || ' days'),
			completed_at = CASE WHEN completed_at IS NULL THEN NULL
				ELSE datetime('` + base + `', '+' || (id * 2) || ' days', '+1 hour') END`,
		`UPDATE sets SET completed_at = (
			SELECT datetime(w.started_at, '+' || (sets.idx * 3) || ' minutes')
			FROM workouts w WHERE w.id = sets.workout_id
		) WHERE completed_at IS NOT NULL`,
		`UPDATE training_maxes SET effective_from = datetime('` + base + `', '+' || id || ' days', '-6 hours')`,
		`UPDATE blocks SET started_at = datetime('` + base + `', '+' || id || ' days', '-6 hours')`,
	} {
		if _, err := h.conn.Exec(stmt); err != nil {
			h.t.Fatalf("pin timestamps: %v", err)
		}
	}
}

// setupDone enters training maxes and joins the block at cycleIdx with the
// sessions given already done.
func (h *harness) setupDone(cycleIdx int, done ...PastSession) {
	h.t.Helper()
	send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
	send[Dashboard](h, http.MethodPut, fmt.Sprintf("/api/v1/setup/history/%d", cycleIdx), SetupHistoryInput{Done: append([]PastSession{}, done...)})
}

// rounds lists every session of the first n rounds as done, with prReps on
// any PR set.
func rounds(n, prReps int) []PastSession {
	var out []PastSession
	for round := 1; round <= n; round++ {
		for _, slug := range []string{"press", "deadlift", "bench", "squat"} {
			out = append(out, PastSession{Round: round, Slug: slug, PRReps: new(prReps)})
		}
	}
	return out
}

// start starts the next session and logs its sets with their target reps until
// stop says to leave one as the current set.
func (h *harness) start(stop func(Set) bool) Workout {
	h.t.Helper()
	w := send[Workout](h, http.MethodPost, "/api/v1/workouts", nil)
	for _, s := range w.Sets {
		if stop != nil && stop(s) {
			break
		}
		w = send[Workout](h, http.MethodPut, fmt.Sprintf("/api/v1/workouts/%d/sets/%d", w.ID, s.Idx), LogSetInput{Reps: s.TargetReps})
	}
	return w
}

func (h *harness) addEstimate(slug, date string, weight float64, reps int) {
	h.t.Helper()
	send[Lift](h, http.MethodPost, "/api/v1/lifts/"+slug+"/estimates", PastEstimateInput{Date: date, Weight: weight, Reps: reps})
}

func firstMain(s Set) bool { return s.Tag == "Main" }

// screens builds one database per screen state and names the request whose
// answer is kept.
var screens = []struct {
	name string
	// prepare fills the database and returns the request to capture.
	prepare func(h *harness) (method, path string, body any)
	status  int // 200 if not given
}{
	// Connect.
	{name: "info", prepare: func(h *harness) (string, string, any) {
		return http.MethodGet, "/api/v1/info", nil
	}},
	{name: "error-input", status: http.StatusBadRequest, prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		return http.MethodPost, "/api/v1/lifts/press/estimates", PastEstimateInput{Date: "2027-01-01", Weight: 120, Reps: 1}
	}},

	// Today.
	{name: "dashboard-setup", prepare: func(h *harness) (string, string, any) {
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-fresh", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-next", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		for range 6 {
			h.session(nil)
		}
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-in-progress", prepare: func(h *harness) (string, string, any) {
		h.setupDone(1, rounds(2, 0)...)
		h.start(firstMain)
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-tm-test-passed", prepare: func(h *harness) (string, string, any) {
		h.setupDone(2)
		h.testSession(map[string]int{"press": 7})
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-tm-test-failed", prepare: func(h *harness) (string, string, any) {
		h.setupDone(2)
		for range 4 {
			h.testSession(map[string]int{"press": 5, "deadlift": 4, "bench": 3, "squat": 2})
		}
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-block-review", prepare: func(h *harness) (string, string, any) {
		h.setupDone(2)
		press := h.testSession(map[string]int{"press": 2, "deadlift": 4, "bench": 7, "squat": 5})
		for range 3 {
			h.testSession(map[string]int{"press": 2, "deadlift": 4, "bench": 7, "squat": 5})
		}
		h.decide(press, "lower", 90)
		for range 12 {
			h.session(func(w db.Workout, set Set) int {
				if set.IsAMRAP && w.LiftSlug == "deadlift" && w.Round == 3 {
					return 0
				}
				if set.IsAMRAP {
					return set.TargetReps + 2
				}
				return set.TargetReps
			})
		}
		send[Dashboard](h, http.MethodPost, "/api/v1/skip-cycle", nil)
		return http.MethodGet, "/api/v1/dashboard", nil
	}},
	{name: "dashboard-skip-deload", prepare: func(h *harness) (string, string, any) {
		h.setupDone(3, rounds(3, 7)...)
		return http.MethodGet, "/api/v1/dashboard", nil
	}},

	// Workout.
	{name: "workout-current", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		w := h.start(firstMain)
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},
	{name: "workout-amrap", prepare: func(h *harness) (string, string, any) {
		h.setupDone(3)
		h.addEstimate("press", "2025-06-02", 95, 8)
		w := h.start(func(s Set) bool { return s.IsAMRAP })
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},
	{name: "workout-range", prepare: func(h *harness) (string, string, any) {
		h.setupDone(4)
		w := h.start(func(s Set) bool { return s.MaxReps > 0 })
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},
	{name: "workout-plates-short", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		send[Settings](h, http.MethodPut, "/api/v1/settings", SettingsInput{Plates: []float64{45, 25, 10}, Warmups: true, BBB: map[string]BBBInput{}})
		w := h.start(firstMain)
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},
	{name: "workout-finish", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		w := h.start(nil)
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},
	{name: "workout-completed", prepare: func(h *harness) (string, string, any) {
		h.setupDone(3)
		w := h.session(func(_ db.Workout, set Set) int {
			if set.IsAMRAP {
				return 9
			}
			return set.TargetReps
		})
		return http.MethodGet, fmt.Sprintf("/api/v1/workouts/%d", w.ID), nil
	}},

	// History.
	{name: "history", prepare: func(h *harness) (string, string, any) {
		h.setupDone(2)
		for range 4 {
			h.testSession(map[string]int{"press": 6, "deadlift": 4, "bench": 5, "squat": 4})
		}
		for range 3 {
			h.session(nil)
		}
		return http.MethodGet, "/api/v1/history", nil
	}},
	{name: "history-empty", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		return http.MethodGet, "/api/v1/history", nil
	}},

	// Lift.
	{name: "lift", prepare: func(h *harness) (string, string, any) {
		h.setupDone(2)
		h.addEstimate("press", "2025-02-10", 105, 1)
		h.addEstimate("press", "2025-09-15", 95, 6)
		for range 4 {
			h.testSession(map[string]int{"press": 6, "deadlift": 4, "bench": 5, "squat": 4})
		}
		for range 8 {
			h.session(func(_ db.Workout, set Set) int {
				if set.IsAMRAP {
					return set.TargetReps + 3
				}
				return set.TargetReps
			})
		}
		send[Settings](h, http.MethodPost, "/api/v1/training-maxes", TrainingMaxInput{Lift: "press", Value: 105, Note: new("felt light")})
		return http.MethodGet, "/api/v1/lifts/press", nil
	}},
	{name: "lift-single", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		h.addEstimate("press", "2025-11-03", 115, 1)
		return http.MethodGet, "/api/v1/lifts/press", nil
	}},
	{name: "lift-empty", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		return http.MethodGet, "/api/v1/lifts/press", nil
	}},

	// Settings.
	{name: "settings", prepare: func(h *harness) (string, string, any) {
		h.setupDone(0)
		send[Settings](h, http.MethodPut, "/api/v1/settings", SettingsInput{
			Plates:            []float64{45, 35, 25, 10, 5, 2.5},
			Warmups:           true,
			RestWarmupSeconds: 60,
			RestMainSeconds:   180,
			RestSuppSeconds:   90,
			BBB: map[string]BBBInput{
				"deadlift": {Mode: "weight", Value: 185},
				"squat":    {Mode: "percent", Value: 55},
			},
		})
		return http.MethodGet, "/api/v1/settings", nil
	}},

	// Setup.
	{name: "setup", prepare: func(h *harness) (string, string, any) {
		return http.MethodGet, "/api/v1/setup", nil
	}},
	{name: "setup-position", prepare: func(h *harness) (string, string, any) {
		send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
		return http.MethodGet, "/api/v1/setup/position", nil
	}},
	{name: "setup-history-leader", prepare: func(h *harness) (string, string, any) {
		send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
		return http.MethodGet, "/api/v1/setup/history/0", nil
	}},
	{name: "setup-history-anchor", prepare: func(h *harness) (string, string, any) {
		send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
		return http.MethodGet, "/api/v1/setup/history/3", nil
	}},
	{name: "setup-history-tm-test", prepare: func(h *harness) (string, string, any) {
		send[SetupPosition](h, http.MethodPut, "/api/v1/setup", SetupInput{Tms: startingTMs})
		return http.MethodGet, "/api/v1/setup/history/2", nil
	}},
}

func TestScreens(t *testing.T) {
	names := map[string]bool{}
	for _, tc := range screens {
		if names[tc.name] {
			t.Fatalf("two screens named %s", tc.name)
		}
		names[tc.name] = true

		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			method, path, body := tc.prepare(h)
			h.pinTimestamps()

			rec := h.call(method, path, body)
			want := tc.status
			if want == 0 {
				want = http.StatusOK
			}
			if rec.Code != want {
				t.Fatalf("%s %s: %d %s, want %d", method, path, rec.Code, rec.Body.String(), want)
			}
			var got bytes.Buffer
			if err := json.Indent(&got, rec.Body.Bytes(), "", "  "); err != nil {
				t.Fatalf("indent: %v", err)
			}

			golden := filepath.Join("testdata", tc.name+".json")
			if *update {
				if err := os.WriteFile(golden, got.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			saved, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v; run go test ./internal/api -update", err)
			}
			if !bytes.Equal(saved, got.Bytes()) {
				t.Errorf("%s %s no longer matches %s; if the change is intended, run go test ./internal/api -update and review the diff.\ngot:\n%s", method, path, golden, got.String())
			}
		})
	}

	// A golden file nothing writes would go on being copied into the app.
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		name := filepath.Base(f)
		name = name[:len(name)-len(".json")]
		if names[name] {
			continue
		}
		if *update {
			os.Remove(f)
			continue
		}
		t.Errorf("%s has no screen; run go test ./internal/api -update to remove it", f)
	}
}
