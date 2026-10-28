// Package training applies the 5/3/1 program to the database: where the user
// is in their block, the sessions they start and log, the training max
// decisions, onboarding, past estimates, settings and history.
//
// The arithmetic lives in package program, which is pure. This package reads
// and writes the database around it and returns read models: everything a
// screen shows, already worked out, so the web pages and the app's API only
// present it. Neither should make a training decision of its own.
package training

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/kbremont/531-tracker/server/internal/db"
)

// Service is the training rules over one database.
type Service struct {
	conn *sql.DB
	q    *db.Queries
	log  *slog.Logger
	now  func() time.Time
}

// New returns the service for a migrated database.
func New(conn *sql.DB, log *slog.Logger) *Service {
	return &Service{conn: conn, q: db.New(conn), log: log, now: time.Now}
}

// WithClock returns a copy of the service that reads the time from now: the
// latest date a past estimate can have, and the backup's filename. Tests use
// it to pin dates, and the demo seeder to place sessions in the past.
func (s *Service) WithClock(now func() time.Time) *Service {
	c := *s
	c.now = now
	return &c
}

var (
	// ErrNotFound means the workout, lift or estimate asked for does not exist.
	ErrNotFound = errors.New("not found")
	// ErrNeedsSetup means the training maxes have not been entered yet.
	ErrNeedsSetup = errors.New("training maxes not set")
	// ErrHeld means training waits on a training max decision: a failed TM
	// test before the anchor, or the review at the start of a block.
	ErrHeld = errors.New("training is waiting on a training max decision")
	// ErrNothingScheduled means there is no next session to start.
	ErrNothingScheduled = errors.New("nothing scheduled")
	// ErrStale means the decision asked for is no longer on offer, for example
	// a review that was already accepted or a cycle that can no longer be
	// skipped. Nothing was written.
	ErrStale = errors.New("no longer on offer")
	// ErrUnknownCycle means a cycle index is not in the program.
	ErrUnknownCycle = errors.New("unknown cycle")
)

// InputError rejects a value the user entered. Nothing was written, and the
// message can be shown as it is.
type InputError struct {
	Msg string
}

func (e *InputError) Error() string { return e.Msg }

func inputError(msg string) error { return &InputError{Msg: msg} }

// tx runs fn inside a transaction, rolling back on error.
func (s *Service) tx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}
