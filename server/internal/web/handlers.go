package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/kbremont/kedge/server/internal/training"
)

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Dashboard(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if d.NeedsSetup {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	h.render(w, r, Dashboard(*d))
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	hist, err := h.svc.History(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, History(*hist))
}

func (h *Handler) Lift(w http.ResponseWriter, r *http.Request) {
	lift, err := h.svc.Lift(r.Context(), r.PathValue("slug"))
	if err != nil {
		h.notFoundOrFail(w, r, err)
		return
	}
	h.render(w, r, LiftDetail(LiftPage{Lift: *lift, Error: r.URL.Query().Get("error") != ""}))
}

// AddPastEstimate records an estimate from before the app. Input the
// browser should already have caught goes back to the lift with an error and
// nothing is written.
func (h *Handler) AddPastEstimate(w http.ResponseWriter, r *http.Request) {
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

	err = h.svc.AddPastEstimate(r.Context(), slug, r.PostForm.Get("recorded_on"), weight, reps)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/lift/"+slug+"?error=1", http.StatusSeeOther)
	case err != nil:
		h.notFoundOrFail(w, r, err)
	default:
		http.Redirect(w, r, "/lift/"+slug, http.StatusSeeOther)
	}
}

func (h *Handler) DeletePastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	if err := h.svc.DeletePastEstimate(r.Context(), slug, id); err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/lift/"+slug, http.StatusSeeOther)
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.Settings(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, Settings(SettingsPage{Settings: *settings, Saved: r.URL.Query().Get("saved") == "1"}))
}

func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
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
	current, err := h.svc.Settings(ctx)
	if err != nil {
		h.fail(w, r, err)
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
			Mode:  r.PostFormValue(bbbModeField(b.Slug)),
			Value: formFloat(r.PostForm, bbbValueField(b.Slug)),
		}
	}

	err = h.svc.SaveSettings(ctx, in)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	case err != nil:
		h.fail(w, r, err)
	default:
		http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
	}
}

// SetTrainingMax appends to the training max log from settings.
func (h *Handler) SetTrainingMax(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	value, err := strconv.ParseFloat(r.PostFormValue("value"), 64)
	if err != nil {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	err = h.svc.SetTrainingMax(r.Context(), r.PostFormValue("lift"), value, r.PostFormValue("note"))
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	case err != nil:
		h.fail(w, r, err)
	default:
		http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
	}
}

// AcceptReview writes the training maxes the user settled on at the
// start of a block. A review no longer pending, or a value missing, goes back
// to the dashboard with nothing written.
func (h *Handler) AcceptReview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	err := h.svc.AcceptReview(r.Context(), r.PostFormValue("key"), formTMs(r.PostForm))
	h.backToDashboard(w, r, err)
}

// DecideTMTest settles a failed TM test: lower the training max, or
// keep it.
func (h *Handler) DecideTMTest(w http.ResponseWriter, r *http.Request) {
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
	h.backToDashboard(w, r, h.svc.DecideTMTest(r.Context(), id, keep, tm))
}

// SkipCycle passes on an optional cycle before its first session.
func (h *Handler) SkipCycle(w http.ResponseWriter, r *http.Request) {
	h.backToDashboard(w, r, h.svc.SkipCycle(r.Context()))
}

// backToDashboard finishes a training max decision. Whether it was made, was
// no longer on offer or had a bad value, the dashboard shows where things
// stand.
func (h *Handler) backToDashboard(w http.ResponseWriter, r *http.Request, err error) {
	var input *training.InputError
	if err != nil && !errors.As(err, &input) && !errors.Is(err, training.ErrStale) && !errors.Is(err, training.ErrHeld) {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Backup streams a consistent snapshot of the database. It is safe to
// call while the app is running.
func (h *Handler) Backup(w http.ResponseWriter, r *http.Request) {
	backup, err := h.svc.Backup(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	defer backup.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", backup.Name))
	if _, err := io.Copy(w, backup); err != nil {
		h.log.ErrorContext(r.Context(), "backup stream interrupted", "error", err)
	}
}

func (h *Handler) notFoundOrFail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, training.ErrNotFound) {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	h.fail(w, r, err)
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
		if slug, ok := strings.CutPrefix(field, tmField("")); ok {
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
