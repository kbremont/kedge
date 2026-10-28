package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kbremont/531-tracker/server/internal/db"
)

// The bounds on a past estimate. They match the add form's min and max, so
// anything the browser accepts the server does too.
const (
	minEstimateWeight = 1
	maxEstimateReps   = 30
)

// AddPastEstimate records an estimate from before the app: an estimated max
// entered as weight x 1, or a PR set such as 135 x 8, on a date written
// YYYY-MM-DD. It returns ErrNotFound for an unknown lift, and an InputError
// for a date in the future, a weight under minEstimateWeight or reps outside 1
// to maxEstimateReps; either way nothing is written.
func (s *Service) AddPastEstimate(ctx context.Context, slug, date string, weight float64, reps int) error {
	if _, err := s.q.GetLift(ctx, slug); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("load lift %s: %w", slug, err)
	}

	recordedOn, ok := validPastEstimate(date, weight, reps, time.Now())
	if !ok {
		return inputError("That estimate was not saved. Check the date, weight and reps.")
	}

	err := s.q.InsertPastEstimate(ctx, db.InsertPastEstimateParams{
		LiftSlug:   slug,
		RecordedOn: recordedOn,
		Weight:     weight,
		Reps:       int64(reps),
	})
	if err != nil {
		return fmt.Errorf("insert past estimate for %s: %w", slug, err)
	}
	return nil
}

// validPastEstimate checks a past estimate: a date that is not in the future,
// a weight of at least minEstimateWeight and a rep count from 1 to
// maxEstimateReps. It returns the date as stored.
func validPastEstimate(date string, weight float64, reps int, now time.Time) (recordedOn string, ok bool) {
	d, err := time.Parse(dayLayout, strings.TrimSpace(date))
	if err != nil {
		return "", false
	}
	recordedOn = d.Format(dayLayout)
	if recordedOn > now.Format(dayLayout) {
		return "", false
	}
	if !(weight >= minEstimateWeight) || math.IsInf(weight, 0) { // the negation also catches NaN
		return "", false
	}
	if reps < 1 || reps > maxEstimateReps {
		return "", false
	}
	return recordedOn, true
}

// DeletePastEstimate removes an estimate from before the app. The slug has to
// match, so a stale page aimed at another lift does nothing.
func (s *Service) DeletePastEstimate(ctx context.Context, slug string, id int64) error {
	err := s.q.DeletePastEstimate(ctx, db.DeletePastEstimateParams{ID: id, LiftSlug: slug})
	if err != nil {
		return fmt.Errorf("delete past estimate %d for %s: %w", id, slug, err)
	}
	return nil
}
