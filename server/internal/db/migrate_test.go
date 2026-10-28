package db

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/kbremont/531-tracker/server/internal/program"
)

// oldDefaultProgram is the default program as it was stored before the 7th
// week: Leader 1, Leader 2, a 40/50/60 deload, then the anchor.
const oldDefaultProgram = `{"Name":"5's PRO + BBB / 5-3-1 + FSL","Cycles":[` +
	`{"Label":"Leader 1","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"bbb","Sets":5,"Reps":10,"Pct":0.5},"Rounds":3},` +
	`{"Label":"Leader 2","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"bbb","Sets":5,"Reps":10,"Pct":0.6},"Rounds":3},` +
	`{"Label":"Deload","Phase":"deload","Main":"deload","Supp":{"Kind":"none","Sets":0,"Reps":0,"Pct":0},"Rounds":1},` +
	`{"Label":"Anchor","Phase":"anchor","Main":"531_pr","Supp":{"Kind":"fsl","Sets":5,"Reps":5,"Pct":0},"Rounds":3}` +
	`],"TMIncrementUpper":5,"TMIncrementLower":10}`

// openBefore returns an in-memory database migrated up to, but not including,
// the migration named stop.
func openBefore(t *testing.T, stop string) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1) // one connection, or each gets its own empty database
	t.Cleanup(func() { conn.Close() })

	if err := createMigrationsTable(conn); err != nil {
		t.Fatal(err)
	}
	names, err := migrationNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if name == stop {
			return conn
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMigration(conn, name, string(body)); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("no migration named %s", stop)
	return nil
}

func mustExec(t *testing.T, conn *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(query, args...); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}

func count(t *testing.T, conn *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := conn.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

const seventhWeek = "0003_seventh_week.sql"

// TestSeventhWeekUpgradesDefaultProgram holds the JSON in the migration and
// program.LeaderAnchorBBB together: an upgraded install must end up with
// exactly the program a fresh one is seeded with.
func TestSeventhWeekUpgradesDefaultProgram(t *testing.T) {
	conn := openBefore(t, seventhWeek)
	mustExec(t, conn, `INSERT INTO programs (name, config, is_active) VALUES (?, ?, 1)`, "default", oldDefaultProgram)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var raw string
	if err := conn.QueryRow(`SELECT config FROM programs`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var got program.Config
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("decode upgraded config: %v", err)
	}
	if want := program.LeaderAnchorBBB(); !reflect.DeepEqual(got, want) {
		t.Errorf("upgraded config differs from the default\n got: %+v\nwant: %+v", got, want)
	}
}

// TestSeventhWeekLeavesCustomProgramsAlone covers a program that is not the
// old default: it is someone's own, and is not rewritten.
func TestSeventhWeekLeavesCustomProgramsAlone(t *testing.T) {
	conn := openBefore(t, seventhWeek)
	custom := `{"Name":"mine","Cycles":[` +
		`{"Label":"A","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"none"},"Rounds":3},` +
		`{"Label":"B","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"none"},"Rounds":3},` +
		`{"Label":"C","Phase":"leader","Main":"5s_pro","Supp":{"Kind":"none"},"Rounds":3},` +
		`{"Label":"D","Phase":"anchor","Main":"531_pr","Supp":{"Kind":"none"},"Rounds":3}` +
		`],"TMIncrementUpper":5,"TMIncrementLower":10}`
	mustExec(t, conn, `INSERT INTO programs (name, config, is_active) VALUES (?, ?, 1)`, "mine", custom)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var raw string
	if err := conn.QueryRow(`SELECT config FROM programs`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw != custom {
		t.Errorf("custom program was rewritten:\n%s", raw)
	}
}

// seedOldDeloadBlock writes an open block that has finished Leader 2 and
// started the old 40/50/60 deload, plus whatever extra workouts the case needs.
func seedOldDeloadBlock(t *testing.T, conn *sql.DB, extraCycleIndex int) {
	t.Helper()
	mustExec(t, conn, `INSERT INTO programs (id, name, config, is_active) VALUES (1, 'default', ?, 1)`, oldDefaultProgram)
	mustExec(t, conn, `INSERT INTO blocks (id, program_id, number) VALUES (1, 1, 1)`)

	// A finished Leader 2 session, and an old deload session with a set.
	mustExec(t, conn, `INSERT INTO workouts (id, block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max, completed_at)
		VALUES (1, 1, 'squat', 1, 'Leader 2 - Round 3', 'leader', 3, 300, datetime('now'))`)
	mustExec(t, conn, `INSERT INTO workouts (id, block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max, completed_at)
		VALUES (2, 1, 'press', 2, 'Deload', 'deload', 1, 150, datetime('now'))`)
	mustExec(t, conn, `INSERT INTO sets (workout_id, idx, kind, weight, pct, target_reps, is_amrap) VALUES (2, 0, 'main', 60, 0.4, 5, 0)`)

	if extraCycleIndex >= 0 {
		mustExec(t, conn, `INSERT INTO workouts (id, block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max)
			VALUES (3, 1, 'press', ?, 'Anchor - Round 1', 'anchor', 1, 150)`, extraCycleIndex)
	}
}

// TestSeventhWeekSetsTheOpenBlockBack puts a block that has started the old
// deload back at the start of the 7th week, so it runs the TM test instead.
func TestSeventhWeekSetsTheOpenBlockBack(t *testing.T) {
	conn := openBefore(t, seventhWeek)
	seedOldDeloadBlock(t, conn, -1)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if n := count(t, conn, `SELECT COUNT(*) FROM workouts WHERE cycle_index = 2`); n != 0 {
		t.Errorf("%d old deload sessions remain, want 0", n)
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM sets WHERE workout_id = 2`); n != 0 {
		t.Errorf("%d sets of the removed session remain, want 0", n)
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM workouts WHERE cycle_index = 1`); n != 1 {
		t.Errorf("leader sessions = %d, want 1 (untouched)", n)
	}
}

// TestSeventhWeekKeepsDeloadOnceAnchorBegan leaves the old deload alone when
// the anchor has already started: by then it is history.
func TestSeventhWeekKeepsDeloadOnceAnchorBegan(t *testing.T) {
	conn := openBefore(t, seventhWeek)
	seedOldDeloadBlock(t, conn, 3)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if n := count(t, conn, `SELECT COUNT(*) FROM workouts WHERE cycle_index = 2`); n != 1 {
		t.Errorf("old deload sessions = %d, want 1 (kept)", n)
	}
}

// TestSeventhWeekOnFreshInstall is the common case: no program row yet, so
// there is nothing to upgrade and nothing to remove.
func TestSeventhWeekOnFreshInstall(t *testing.T) {
	conn := openBefore(t, seventhWeek)
	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM programs`); n != 0 {
		t.Errorf("programs = %d, want 0", n)
	}
	// The new column defaults to a fixed rep count.
	mustExec(t, conn, `INSERT INTO programs (id, name, config, is_active) VALUES (1, 'default', '{}', 1)`)
	mustExec(t, conn, `INSERT INTO blocks (id, program_id, number) VALUES (1, 1, 1)`)
	mustExec(t, conn, `INSERT INTO workouts (id, block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max)
		VALUES (1, 1, 'press', 0, 'Leader 1 - Round 1', 'leader', 1, 150)`)
	mustExec(t, conn, `INSERT INTO sets (workout_id, idx, kind, weight, pct, target_reps, is_amrap) VALUES (1, 0, 'main', 100, 0.65, 5, 0)`)
	if n := count(t, conn, `SELECT max_reps FROM sets WHERE workout_id = 1`); n != 0 {
		t.Errorf("max_reps default = %d, want 0", n)
	}
}

const pastEstimates = "0004_past_estimates.sql"

// TestPastEstimatesApplyToAnExistingDatabase adds the table on top of an
// install that already has workouts, and leaves them as they were.
func TestPastEstimatesApplyToAnExistingDatabase(t *testing.T) {
	conn := openBefore(t, pastEstimates)
	mustExec(t, conn, `INSERT INTO programs (name, config, is_active) VALUES ('p', '{}', 1)`)
	mustExec(t, conn, `INSERT INTO blocks (program_id, number) VALUES (1, 1)`)
	mustExec(t, conn, `INSERT INTO workouts (block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max)
		VALUES (1, 'press', 0, 'Leader 1', 'leader', 1, 100)`)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM workouts`); n != 1 {
		t.Errorf("%d workouts after migrating, want the 1 that was there", n)
	}
	mustExec(t, conn, `INSERT INTO past_estimates (lift_slug, recorded_on, weight, reps) VALUES ('press', '2025-01-15', 120, 1)`)
	if n := count(t, conn, `SELECT COUNT(*) FROM past_estimates`); n != 1 {
		t.Errorf("%d past estimates, want 1", n)
	}
}

func TestPastEstimateConstraints(t *testing.T) {
	conn := openBefore(t, pastEstimates)
	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for name, args := range map[string][]any{
		"zero weight":     {"press", "2025-01-15", 0, 1},
		"negative weight": {"press", "2025-01-15", -5, 1},
		"zero reps":       {"press", "2025-01-15", 120, 0},
		"negative reps":   {"press", "2025-01-15", 120, -1},
		"unknown lift":    {"curl", "2025-01-15", 120, 1},
	} {
		_, err := conn.Exec(`INSERT INTO past_estimates (lift_slug, recorded_on, weight, reps) VALUES (?, ?, ?, ?)`, args...)
		if err == nil {
			t.Errorf("%s was accepted, want it rejected", name)
		}
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM past_estimates`); n != 0 {
		t.Errorf("%d rows after rejected inserts, want 0", n)
	}
}

const tmTestAMRAP = "0005_tm_test_amrap.sql"

// TestTMTestAMRAPUpgradesEarlierTests marks the 100% set of a TM test logged
// before it became an AMRAP as one, and touches nothing else: not the test's
// other sets, not the deload's 3-5 range, and not the reps that were logged.
func TestTMTestAMRAPUpgradesEarlierTests(t *testing.T) {
	conn := openBefore(t, tmTestAMRAP)
	mustExec(t, conn, `INSERT INTO programs (id, name, config, is_active) VALUES (1, 'p', '{}', 1)`)
	mustExec(t, conn, `INSERT INTO blocks (id, program_id, number) VALUES (1, 1, 1)`)
	mustExec(t, conn, `INSERT INTO workouts (id, block_id, lift_slug, cycle_index, cycle_label, phase, round_number, training_max, completed_at) VALUES
		(1, 1, 'press',    2, '7th Week TM Test', 'tm_test', 1, 100, datetime('now')),
		(2, 1, 'deadlift', 2, '7th Week TM Test', 'tm_test', 1, 200, NULL),
		(3, 1, 'press',    3, 'Anchor - Round 1', 'anchor',  1, 100, datetime('now')),
		(4, 1, 'press',    4, '7th Week Deload',  'deload',  1, 100, datetime('now'))`)
	mustExec(t, conn, `INSERT INTO sets (workout_id, idx, kind, weight, pct, target_reps, max_reps, is_amrap, actual_reps) VALUES
		(1, 0, 'warmup', 45,  0.4,  5, 0, 0, 5),
		(1, 3, 'main',   90,  0.9,  5, 0, 0, 5),
		(1, 4, 'main',   100, 1.0,  3, 5, 0, 9),
		(2, 4, 'main',   200, 1.0,  3, 5, 0, NULL),
		(3, 2, 'main',   85,  0.85, 5, 0, 1, 8),
		(4, 1, 'main',   80,  0.8,  3, 5, 0, 4)`)

	if err := Migrate(conn); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	amrap := func(workout, idx int) int {
		return count(t, conn, `SELECT is_amrap FROM sets WHERE workout_id = ? AND idx = ?`, workout, idx)
	}
	cases := []struct {
		name         string
		workout, idx int
		want         int
	}{
		{"logged TM test 100% set", 1, 4, 1},
		{"in-progress TM test 100% set", 2, 4, 1},
		{"TM test 90% set", 1, 3, 0},
		{"TM test warmup", 1, 0, 0},
		{"anchor PR set", 3, 2, 1},
		{"deload 80% range", 4, 1, 0},
	}
	for _, tc := range cases {
		if got := amrap(tc.workout, tc.idx); got != tc.want {
			t.Errorf("%s: is_amrap = %d, want %d", tc.name, got, tc.want)
		}
	}
	if n := count(t, conn, `SELECT COUNT(*) FROM sets WHERE workout_id = 1 AND idx = 4
		AND weight = 100 AND target_reps = 3 AND max_reps = 5 AND actual_reps = 9`); n != 1 {
		t.Error("the upgraded set's weight or reps changed")
	}
}
