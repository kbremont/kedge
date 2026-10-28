package server

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kbremont/531-tracker/server/internal/training"
	"github.com/kbremont/531-tracker/server/internal/views"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dashboard(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if d.NeedsSetup {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	s.render(w, r, views.Dashboard(*d))
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	h, err := s.svc.History(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, views.History(*h))
}

func (s *Server) handleLift(w http.ResponseWriter, r *http.Request) {
	lift, err := s.svc.Lift(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.notFoundOrFail(w, r, err)
		return
	}
	s.render(w, r, views.LiftDetail(views.LiftPage{Lift: *lift, Error: r.URL.Query().Get("error") != ""}))
}

// handleAddPastEstimate records an estimate from before the app. Input the
// browser should already have caught goes back to the lift with an error and
// nothing is written.
func (s *Server) handleAddPastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	// A form that does not parse leaves the fields empty, which training
	// rejects once it has checked the lift exists.
	r.ParseForm()

	// Unparseable numbers become zeros, which training rejects as well.
	weight, _ := strconv.ParseFloat(strings.TrimSpace(r.PostForm.Get("weight")), 64)
	reps, err := strconv.Atoi(strings.TrimSpace(r.PostForm.Get("reps")))
	if err != nil {
		reps = 0
	}

	err = s.svc.AddPastEstimate(r.Context(), slug, r.PostForm.Get("recorded_on"), weight, reps)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/lift/"+slug+"?error=1", http.StatusSeeOther)
	case err != nil:
		s.notFoundOrFail(w, r, err)
	default:
		http.Redirect(w, r, "/lift/"+slug, http.StatusSeeOther)
	}
}

func (s *Server) handleDeletePastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	if err := s.svc.DeletePastEstimate(r.Context(), slug, id); err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/lift/"+slug, http.StatusSeeOther)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.svc.Settings(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, views.Settings(views.SettingsPage{Settings: *settings, Saved: r.URL.Query().Get("saved") == "1"}))
}

func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	plates, err := parsePlates(r.PostFormValue("plates"))
	if err != nil {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	// The BBB inputs are named for each lift, so read back the lifts the form
	// was drawn from.
	current, err := s.svc.Settings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	in := training.SettingsInput{
		Rounding:          formFloat(r.PostForm, "rounding"),
		BarWeight:         formFloat(r.PostForm, "bar_weight"),
		Plates:            plates,
		Warmups:           r.PostFormValue("warmups") != "",
		RestWarmupSeconds: int(formFloat(r.PostForm, "rest_warmup_seconds")),
		RestMainSeconds:   int(formFloat(r.PostForm, "rest_main_seconds")),
		RestSuppSeconds:   int(formFloat(r.PostForm, "rest_supp_seconds")),
		BBB:               make(map[string]training.LiftBBB, len(current.BBB)),
	}
	for _, b := range current.BBB {
		in.BBB[b.Slug] = training.LiftBBB{
			Mode:  r.PostFormValue(views.BBBModeField(b.Slug)),
			Value: formFloat(r.PostForm, views.BBBValueField(b.Slug)),
		}
	}

	err = s.svc.SaveSettings(ctx, in)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	case err != nil:
		s.fail(w, r, err)
	default:
		http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
	}
}

// handleSetTrainingMax appends to the training max log from settings.
func (s *Server) handleSetTrainingMax(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	value, err := strconv.ParseFloat(r.PostFormValue("value"), 64)
	if err != nil {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	err = s.svc.SetTrainingMax(r.Context(), r.PostFormValue("lift"), value, r.PostFormValue("note"))
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	case err != nil:
		s.fail(w, r, err)
	default:
		http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
	}
}

// handleAcceptReview writes the training maxes the user settled on at the
// start of a block. A review no longer pending, or a value missing, goes back
// to the dashboard with nothing written.
func (s *Server) handleAcceptReview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err := s.svc.AcceptReview(r.Context(), r.PostFormValue("key"), formTMs(r.PostForm))
	s.backToDashboard(w, r, err)
}

// handleTMTestDecision settles a failed TM test: lower the training max, or
// keep it.
func (s *Server) handleTMTestDecision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	keep := r.PostFormValue("action") == "keep"
	tm, _ := strconv.ParseFloat(strings.TrimSpace(r.PostFormValue("tm")), 64)
	s.backToDashboard(w, r, s.svc.DecideTMTest(r.Context(), id, keep, tm))
}

// handleSkipCycle passes on an optional cycle before its first session.
func (s *Server) handleSkipCycle(w http.ResponseWriter, r *http.Request) {
	s.backToDashboard(w, r, s.svc.SkipCycle(r.Context()))
}

// backToDashboard finishes a training max decision. Whether it was made, was
// no longer on offer or had a bad value, the dashboard shows where things
// stand.
func (s *Server) backToDashboard(w http.ResponseWriter, r *http.Request, err error) {
	var input *training.InputError
	if err != nil && !errors.As(err, &input) && !errors.Is(err, training.ErrStale) && !errors.Is(err, training.ErrHeld) {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleBackup streams a consistent snapshot of the database. It is safe to
// call while the app is running.
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	backup, err := s.svc.Backup(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer backup.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", backup.Name))
	if _, err := io.Copy(w, backup); err != nil {
		s.log.ErrorContext(r.Context(), "backup stream interrupted", "error", err)
	}
}

func (s *Server) notFoundOrFail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, training.ErrNotFound) {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	s.fail(w, r, err)
}

// formFloat reads a number from a form, or 0 when the field is missing or not
// a number.
func formFloat(form url.Values, field string) float64 {
	v, err := strconv.ParseFloat(form.Get(field), 64)
	if err != nil {
		return 0
	}
	return v
}

// formTMs reads the training max inputs, one per lift, by slug. A value that
// is not a number reads as 0, which training rejects.
func formTMs(form url.Values) map[string]float64 {
	out := make(map[string]float64)
	for field := range form {
		if slug, ok := strings.CutPrefix(field, views.TMField("")); ok {
			out[slug], _ = strconv.ParseFloat(strings.TrimSpace(form.Get(field)), 64)
		}
	}
	return out
}

// parsePlates reads the comma-separated plate list shown in settings.
func parsePlates(in string) ([]float64, error) {
	fields := strings.FieldsFunc(in, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]float64, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", f)
		}
		out = append(out, v)
	}
	return out, nil
}
