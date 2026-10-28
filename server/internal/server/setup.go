package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kbremont/531-tracker/server/internal/training"
	"github.com/kbremont/531-tracker/server/internal/views"
)

// Onboarding lets someone who is already mid-programme start using the app
// without keying in their whole training history: training maxes first, then
// the cycle they are on, then the sessions of it they have already done.

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	setup, err := s.svc.Setup(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, views.Setup(views.SetupPage{Setup: *setup}))
}

func (s *Server) handleSetupSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	values := formTMs(r.PostForm)
	err := s.svc.SaveSetup(ctx, values)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		// Show the form again with what was entered.
		setup, err := s.svc.Setup(ctx)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		for i, l := range setup.Lifts {
			setup.Lifts[i].Value = values[l.Slug]
		}
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, r, views.Setup(views.SetupPage{Setup: *setup, Error: input.Msg}))
	case err != nil:
		s.fail(w, r, err)
	default:
		// Training maxes alone are not enough to know which session is next;
		// ask where in the programme they are before opening the dashboard.
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
	}
}

func (s *Server) handleSetupPosition(w http.ResponseWriter, r *http.Request) {
	pos, err := s.svc.SetupPosition(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, views.SetupPosition(*pos))
}

func (s *Server) handleSetupPositionSave(w http.ResponseWriter, r *http.Request) {
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

	pos, err := s.svc.SetupPosition(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 0 || idx >= len(pos.Cycles) {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/setup/history?cycle=%d", idx), http.StatusSeeOther)
}

func (s *Server) handleSetupHistory(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.URL.Query().Get("cycle"))
	if err != nil {
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
		return
	}
	hist, err := s.svc.SetupHistory(r.Context(), idx)
	if err != nil {
		s.setupHistoryFailed(w, r, err)
		return
	}
	s.render(w, r, views.SetupHistory(*hist))
}

func (s *Server) handleSetupHistorySave(w http.ResponseWriter, r *http.Request) {
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
	hist, err := s.svc.SetupHistory(ctx, idx)
	if err != nil {
		s.setupHistoryFailed(w, r, err)
		return
	}

	var done []training.PastSession
	for _, rd := range hist.Rounds {
		for _, day := range rd.Days {
			if r.PostFormValue(views.DoneField(rd.Round, day.Slug)) == "" {
				continue
			}
			reps, _ := strconv.Atoi(r.PostFormValue(views.RepsField(rd.Round, day.Slug)))
			done = append(done, training.PastSession{Round: rd.Round, Slug: day.Slug, PRReps: reps})
		}
	}
	if err := s.svc.SaveSetupHistory(ctx, idx, done); err != nil {
		s.setupHistoryFailed(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// setupHistoryFailed sends the user back a step when the checklist cannot be
// shown: to enter training maxes, or to pick a cycle that exists.
func (s *Server) setupHistoryFailed(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, training.ErrNeedsSetup):
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
	case errors.Is(err, training.ErrUnknownCycle):
		http.Redirect(w, r, "/setup/position", http.StatusSeeOther)
	default:
		s.fail(w, r, err)
	}
}
