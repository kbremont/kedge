package demo

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/training"
)

func TestSeed(t *testing.T) {
	conn, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1) // one connection, or each gets its own empty database
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	seeded, err := Seed(ctx, conn, log)
	if err != nil || !seeded {
		t.Fatalf("Seed = %v, %v; want it to seed an empty database", seeded, err)
	}
	svc := training.New(conn, log)

	// The next anchor session is ready to start, with the block behind it.
	d, err := svc.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.NeedsSetup || d.Next == nil || d.Next.InProgress || d.Next.Blocked || d.Next.Phase != "anchor" || d.Review != nil {
		t.Fatalf("dashboard next = %+v, review %+v; want an anchor session ready to start", d.Next, d.Review)
	}
	if d.Next.LiftSlug != "bench" || d.Next.CycleLabel != "Anchor - Round 2" {
		t.Errorf("next = %s %s, want bench's Anchor - Round 2", d.Next.LiftSlug, d.Next.CycleLabel)
	}
	if len(d.Recent) == 0 || len(d.TestResults) != 0 {
		t.Errorf("recent %d sessions, %d TM test results; want sessions and nothing to decide", len(d.Recent), len(d.TestResults))
	}

	// Every lift has a trend: the TM test, the anchor, and the notebook.
	for _, slug := range []string{"press", "deadlift", "bench", "squat"} {
		l, err := svc.Lift(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		var past int
		for _, p := range l.Points {
			if p.Past {
				past++
			}
		}
		if len(l.PastEstimates) < 2 || past != len(l.PastEstimates) || len(l.Points) <= past {
			t.Errorf("%s: %d points, %d past estimates; want both, and logged points after them", slug, len(l.Points), len(l.PastEstimates))
		}
		// And it climbs: each estimated max beats the one before, which is
		// what the demo, and the App Store screenshots taken from it, show.
		// Y is in SVG space, so higher on the chart is a smaller Y.
		for i := 1; i < len(l.Points); i++ {
			if l.Points[i].Y >= l.Points[i-1].Y {
				t.Errorf("%s: point %d (%s) isn't above point %d (%s)", slug, i, l.Points[i].Label, i-1, l.Points[i-1].Label)
			}
		}
	}

	// Every date is in the past.
	for _, q := range []string{
		`SELECT COUNT(*) FROM workouts WHERE started_at >= datetime('now') OR completed_at IS NULL OR completed_at >= datetime('now')`,
		`SELECT COUNT(*) FROM sets WHERE completed_at IS NULL OR completed_at >= datetime('now')`,
		`SELECT COUNT(*) FROM training_maxes WHERE effective_from >= datetime('now')`,
		`SELECT COUNT(*) FROM blocks WHERE started_at >= datetime('now')`,
		`SELECT COUNT(*) FROM past_estimates WHERE recorded_on >= date('now')`,
		// And the training maxes date from before the first session.
		`SELECT COUNT(*) FROM training_maxes WHERE effective_from >= (SELECT MIN(started_at) FROM workouts)`,
	} {
		var n int
		if err := conn.QueryRow(q).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%d rows from %s", n, q)
		}
	}

	// A second seed leaves the database alone.
	var before, after int
	conn.QueryRow(`SELECT COUNT(*) FROM workouts`).Scan(&before)
	seeded, err = Seed(ctx, conn, log)
	if err != nil || seeded {
		t.Errorf("second Seed = %v, %v; want nothing done", seeded, err)
	}
	conn.QueryRow(`SELECT COUNT(*) FROM workouts`).Scan(&after)
	if before != 34 || after != before {
		t.Errorf("workouts %d before the second seed and %d after, want 34 both times", before, after)
	}
}
