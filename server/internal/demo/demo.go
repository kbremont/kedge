// Package demo fills an empty database with a realistic block of training, for
// the public demo server App Review signs in to and for trying the app
// locally. It drives the real rules through package training, so the data is
// exactly what the app would have produced; only the dates are moved into the
// past afterwards.
package demo

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/kbremont/kedge/server/internal/training"
)

// The lifter: training maxes from a couple of years of 5/3/1.
var trainingMaxes = map[string]float64{"press": 115, "deadlift": 365, "bench": 205, "squat": 305}

// The block so far, in session order: both leaders, the TM test with every lift
// passing, and the anchor's first six sessions. The seventh, bench's second
// anchor session, is next.
const (
	leaderSessions = 24
	anchorSessions = 6
)

// The demo lifter is getting stronger: on every lift's chart each estimated
// max beats the last, from the notebook through the TM test into the anchor.
// TestSeed holds it to that.

// testReps is each lift's 100% set on the TM test: all passes, so nothing is
// held, and press strong enough to earn a bump at the end of the block.
var testReps = map[string]int{"Overhead Press": 6, "Deadlift": 3, "Bench Press": 4, "Squat": 3}

// prReps is what each lift does on its anchor PR sets, by the set's target:
// 5+ in the first round, 3+ in the second, 1+ in the third.
var prReps = map[string]map[int]int{
	"Overhead Press": {5: 12, 3: 11, 1: 10},
	"Deadlift":       {5: 10, 3: 8, 1: 7},
	"Bench Press":    {5: 11, 3: 9, 1: 8},
	"Squat":          {5: 10, 3: 8, 1: 7},
}

// pastEstimates are notebook numbers from the year before the app.
var pastEstimates = []struct {
	slug         string
	monthsBefore int // before a year ago
	weight       float64
	reps         int
}{
	{"press", 4, 105, 6},
	{"press", 1, 115, 4},
	{"deadlift", 5, 335, 3},
	{"deadlift", 2, 345, 4},
	{"deadlift", 0, 350, 4},
	{"bench", 4, 185, 6},
	{"bench", 1, 195, 5},
	{"squat", 3, 275, 5},
	{"squat", 0, 285, 5},
}

// Seed fills the database with the demo lifter's block, unless training maxes
// have already been entered, in which case it does nothing and reports false.
func Seed(ctx context.Context, conn *sql.DB, log *slog.Logger) (seeded bool, err error) {
	var tms int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM training_maxes`).Scan(&tms); err != nil {
		return false, fmt.Errorf("count training maxes: %w", err)
	}
	if tms > 0 {
		return false, nil
	}

	svc := training.New(conn, log)
	if err := svc.SaveSetup(ctx, trainingMaxes); err != nil {
		return false, fmt.Errorf("enter training maxes: %w", err)
	}
	if err := svc.SaveSetupHistory(ctx, 0, nil); err != nil {
		return false, fmt.Errorf("join the block: %w", err)
	}
	if err := svc.SaveSettings(ctx, training.SettingsInput{
		Plates:            []float64{45, 35, 25, 10, 5, 2.5},
		Warmups:           true,
		RestWarmupSeconds: 60,
		RestMainSeconds:   180,
		RestSuppSeconds:   90,
	}); err != nil {
		return false, fmt.Errorf("save settings: %w", err)
	}

	var sessions []int64
	for range leaderSessions + 4 + anchorSessions {
		id, err := session(ctx, svc)
		if err != nil {
			return false, err
		}
		sessions = append(sessions, id)
	}

	now := time.Now()
	for _, e := range pastEstimates {
		date := now.AddDate(-1, -e.monthsBefore, 0).Format("2006-01-02")
		if err := svc.AddPastEstimate(ctx, e.slug, date, e.weight, e.reps); err != nil {
			return false, fmt.Errorf("add past estimate for %s: %w", e.slug, err)
		}
	}

	if err := backdate(ctx, conn, sessions, now); err != nil {
		return false, err
	}
	return true, nil
}

// session starts the next session and logs every set: the prescription,
// except the AMRAP sets, which the demo lifter takes past their target.
func session(ctx context.Context, svc *training.Service) (int64, error) {
	id, err := svc.StartWorkout(ctx)
	if err != nil {
		return 0, fmt.Errorf("start session: %w", err)
	}
	w, err := svc.Workout(ctx, id)
	if err != nil {
		return 0, err
	}
	for _, set := range w.Sets {
		reps := set.TargetReps
		switch {
		case set.IsAMRAP && w.Phase == "tm_test":
			reps = testReps[w.LiftName]
		case set.IsAMRAP && prReps[w.LiftName][set.TargetReps] > 0:
			reps = prReps[w.LiftName][set.TargetReps]
		}
		if err := svc.LogSet(ctx, id, int64(set.Idx), reps); err != nil {
			return 0, fmt.Errorf("log set %d of session %d: %w", set.Idx, id, err)
		}
	}
	if err := svc.FinishWorkout(ctx, id); err != nil {
		return 0, err
	}
	return id, nil
}

// backdate spreads the sessions over the weeks before now, about three a week
// with the last one yesterday evening, and starts the block and its training
// maxes the day before the first. Timestamps are stored as UTC, as SQLite's
// datetime('now') writes them.
func backdate(ctx context.Context, conn *sql.DB, sessions []int64, now time.Time) error {
	const layout = "2006-01-02 15:04:05"
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, time.Local).AddDate(0, 0, -1).UTC()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var first time.Time
	for i, id := range sessions {
		// Sessions two or three days apart: three a week.
		daysBefore := (len(sessions) - 1 - i) * 7 / 3
		start := yesterday.AddDate(0, 0, -daysBefore)
		if i == 0 {
			first = start
		}
		if _, err := tx.ExecContext(ctx, `UPDATE workouts SET started_at = ?, completed_at = ? WHERE id = ?`,
			start.Format(layout), start.Add(70*time.Minute).Format(layout), id); err != nil {
			return fmt.Errorf("backdate session %d: %w", id, err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE sets SET completed_at = datetime(?, '+' || (idx * 3) || ' minutes') WHERE workout_id = ?`,
			start.Format(layout), id); err != nil {
			return fmt.Errorf("backdate sets of session %d: %w", id, err)
		}
	}

	blockStart := first.AddDate(0, 0, -1).Format(layout)
	if _, err := tx.ExecContext(ctx, `UPDATE training_maxes SET effective_from = ?`, blockStart); err != nil {
		return fmt.Errorf("backdate training maxes: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE blocks SET started_at = ?`, blockStart); err != nil {
		return fmt.Errorf("backdate the block: %w", err)
	}
	return tx.Commit()
}
