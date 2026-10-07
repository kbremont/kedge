package server

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kbremont/531-tracker/internal/db"
)

// The bounds on a past estimate. They match the add form's min and max, so
// anything the browser accepts the server does too.
const (
	minEstimateWeight = 1
	maxEstimateReps   = 30
)

// handleAddPastEstimate records an estimate from before the app: an estimated
// max entered as weight x 1, or a PR set such as 135 x 8. Input the browser
// should already have caught goes back to the lift with an error and nothing
// is written.
func (s *Server) handleAddPastEstimate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	if _, err := s.q.GetLift(ctx, slug); err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	back := "/lift/" + slug
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, back+"?error=1", http.StatusSeeOther)
		return
	}

	recordedOn, weight, reps, ok := parsePastEstimate(r.PostForm.Get("recorded_on"), r.PostForm.Get("weight"), r.PostForm.Get("reps"), time.Now())
	if !ok {
		http.Redirect(w, r, back+"?error=1", http.StatusSeeOther)
		return
	}

	err := s.q.InsertPastEstimate(ctx, db.InsertPastEstimateParams{
		LiftSlug:   slug,
		RecordedOn: recordedOn,
		Weight:     weight,
		Reps:       reps,
	})
	if err != nil {
		s.fail(w, r, fmt.Errorf("insert past estimate for %s: %w", slug, err))
		return
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// parsePastEstimate checks a submitted past estimate: a date that is not in
// the future, a weight of at least minEstimateWeight and a rep count from 1 to
// maxEstimateReps.
func parsePastEstimate(date, weight, reps string, now time.Time) (recordedOn string, w float64, n int64, ok bool) {
	d, err := time.Parse(dayLayout, strings.TrimSpace(date))
	if err != nil {
		return "", 0, 0, false
	}
	recordedOn = d.Format(dayLayout)
	if recordedOn > now.Format(dayLayout) {
		return "", 0, 0, false
	}

	w, err = strconv.ParseFloat(strings.TrimSpace(weight), 64)
	if err != nil || !(w >= minEstimateWeight) || math.IsInf(w, 0) { // the negation also catches NaN
		return "", 0, 0, false
	}
	n, err = strconv.ParseInt(strings.TrimSpace(reps), 10, 64)
	if err != nil || n < 1 || n > maxEstimateReps {
		return "", 0, 0, false
	}
	return recordedOn, w, n, true
}

func (s *Server) handleDeletePastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	err = s.q.DeletePastEstimate(r.Context(), db.DeletePastEstimateParams{ID: id, LiftSlug: slug})
	if err != nil {
		s.fail(w, r, fmt.Errorf("delete past estimate %d for %s: %w", id, slug, err))
		return
	}
	http.Redirect(w, r, "/lift/"+slug, http.StatusSeeOther)
}
