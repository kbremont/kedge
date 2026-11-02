package training

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/kbremont/kedge/server/internal/db"
)

// TestWithoutNotesDropsATrainingMaxNote keeps the value of a training max set
// from settings and drops its note, as the public demo needs, while the usual
// service keeps both.
func TestWithoutNotesDropsATrainingMaxNote(t *testing.T) {
	cases := []struct {
		name     string
		svc      func(*Service) *Service
		wantNote string
	}{
		{"kept", func(s *Service) *Service { return s }, "from the notebook"},
		{"dropped", (*Service).WithoutNotes, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			conn, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			conn.SetMaxOpenConns(1)
			t.Cleanup(func() { conn.Close() })
			if err := db.Migrate(conn); err != nil {
				t.Fatalf("migrate: %v", err)
			}
			svc := c.svc(New(conn, slog.New(slog.NewTextHandler(io.Discard, nil))))

			ctx := context.Background()
			if err := svc.SetTrainingMax(ctx, "press", 120, "from the notebook"); err != nil {
				t.Fatalf("SetTrainingMax: %v", err)
			}
			history, err := svc.q.ListTrainingMaxHistory(ctx, "press")
			if err != nil || len(history) != 1 {
				t.Fatalf("history = %v, %v; want one entry", history, err)
			}
			if history[0].Value != 120 || history[0].Note != c.wantNote {
				t.Errorf("got %v with note %q, want 120 with %q", history[0].Value, history[0].Note, c.wantNote)
			}
		})
	}
}
