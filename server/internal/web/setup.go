package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kbremont/kedge/server/internal/training"
)

// Onboarding lets someone who is already mid-programme start using the app
// without keying in their whole training history: training maxes first, then
// the cycle they are on, then the sessions of it they have already done.

func (h *Handler) Setup(w http.ResponseWriter, r *http.Request) {
	setup, err := h.svc.Setup(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, Setup(SetupPage{Setup: *setup}))
}

func (h *Handler) SaveSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	values := formTMs(r.PostForm)
	unit := r.PostForm.Get("unit")
	err := h.svc.SaveSetup(ctx, values, unit)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		// Show the form again with what was entered.
		setup, err := h.svc.Setup(ctx)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		for i, l := range setup.Lifts {
			setup.Lifts[i].Value = values[l.Slug]
		}
		if setup.CanChangeUnit && unit != "" {
			setup.Unit = unit
		}
		w.WriteHeader(http.StatusBadRequest)
		h.render(w, r, Setup(SetupPage{Setup: *setup, Error: input.Msg}))
	case err != nil:
		h.fail(w, r, err)
	default:
		// Training maxes alone are not enough to know which session is next;
		// ask where in the programme they are before opening the dashboard.
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
	}
}

func (h *Handler) SetupPosition(w http.ResponseWriter, r *http.Request) {
	pos, err := h.svc.SetupPosition(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.render(w, r, SetupPosition(*pos))
}

func (h *Handler) SaveSetupPosition(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	choice := r.PostFormValue("cycle")
	if choice == "" || choice == "fresh" {
		// Nothing to record; the block already starts at cycle zero.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	pos, err := h.svc.SetupPosition(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= len(pos.Cycles) {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/setup/history?cycle=%d", idx), http.StatusSeeOther)
}

func (h *Handler) SetupHistory(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.URL.Query().Get("cycle"))
	if err != nil {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	hist, err := h.svc.SetupHistory(r.Context(), idx)
	if err != nil {
		h.setupHistoryFailed(w, r, err)
		return
	}
	h.render(w, r, SetupHistory(*hist))
}

func (h *Handler) SaveSetupHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	idx, err := strconv.Atoi(r.PostFormValue("cycle"))
	if err != nil {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	// The checkboxes are named for the sessions of the cycle, so read back the
	// checklist the form was drawn from.
	hist, err := h.svc.SetupHistory(ctx, idx)
	if err != nil {
		h.setupHistoryFailed(w, r, err)
		return
	}

	var done []training.PastSession
	for _, rd := range hist.Rounds {
		for _, day := range rd.Days {
			if r.PostFormValue(doneField(rd.Round, day.Slug)) == "" {
				continue
			}
			reps, _ := strconv.Atoi(r.PostFormValue(repsField(rd.Round, day.Slug)))
			done = append(done, training.PastSession{Round: rd.Round, Slug: day.Slug, PRReps: reps})
		}
	}
	if err := h.svc.SaveSetupHistory(ctx, idx, done); err != nil {
		h.setupHistoryFailed(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// setupHistoryFailed sends the user back a step when the checklist cannot be
// shown: to enter training maxes, or to pick a cycle that exists.
func (h *Handler) setupHistoryFailed(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, training.ErrNeedsSetup):
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
	case errors.Is(err, training.ErrUnknownCycle):
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
	default:
		h.fail(w, r, err)
	}
}
