package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kbremont/531-tracker/server/internal/training"
	"github.com/kbremont/531-tracker/server/internal/views"
)

// handleStartWorkout starts the next scheduled session, or resumes the one in
// progress. When there is nothing to start, or a training max decision is
// waiting, the dashboard says why.
func (s *Server) handleStartWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := s.svc.StartWorkout(r.Context())
	switch {
	case errors.Is(err, training.ErrNothingScheduled), errors.Is(err, training.ErrHeld):
		http.Redirect(w, r, "/", http.StatusSeeOther)
	case errors.Is(err, training.ErrNeedsSetup):
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
	case err != nil:
		s.fail(w, r, err)
	default:
		http.Redirect(w, r, fmt.Sprintf("/workout/%d", id), http.StatusSeeOther)
	}
}

func (s *Server) handleWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	workout, err := s.svc.Workout(r.Context(), id)
	if err != nil {
		s.notFoundOrFail(w, r, err)
		return
	}
	s.render(w, r, views.Workout(views.WorkoutPage{Workout: *workout, ShowRest: r.URL.Query().Get("rest") == "1"}))
}

func (s *Server) handleLogSet(w http.ResponseWriter, r *http.Request) {
	workoutID, idx, err := pathIDAndIdx(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Reps the form should not have allowed go back to the set, unlogged.
	back := fmt.Sprintf("/workout/%d#set-%d", workoutID, idx)
	reps, err := strconv.Atoi(r.PostFormValue("reps"))
	if err != nil {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	err = s.svc.LogSet(r.Context(), workoutID, idx, reps)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, back, http.StatusSeeOther)
	case err != nil:
		s.fail(w, r, err)
	default:
		s.redirectToNextSet(w, r, workoutID)
	}
}

func (s *Server) handleUndoSet(w http.ResponseWriter, r *http.Request) {
	workoutID, idx, err := pathIDAndIdx(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.svc.UndoSet(r.Context(), workoutID, idx); err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d#set-%d", workoutID, idx), http.StatusSeeOther)
}

// redirectToNextSet sends the browser back to the session, anchored on the
// set now due. The page load doubles as the start of the rest timer.
func (s *Server) redirectToNextSet(w http.ResponseWriter, r *http.Request, workoutID int64) {
	workout, err := s.svc.Workout(r.Context(), workoutID)
	if err != nil && !errors.Is(err, training.ErrNotFound) {
		s.fail(w, r, err)
		return
	}
	if workout != nil && workout.CurrentIdx >= 0 {
		http.Redirect(w, r, fmt.Sprintf("/workout/%d?rest=1#rest", workoutID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d", workoutID), http.StatusSeeOther)
}

func (s *Server) handleFinishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.svc.FinishWorkout(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleAbandonWorkout removes a session that was started by mistake.
func (s *Server) handleAbandonWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := s.svc.AbandonWorkout(r.Context(), id); err != nil {
		s.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func pathIDAndIdx(r *http.Request) (int64, int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	idx, err := strconv.ParseInt(r.PathValue("idx"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return id, idx, nil
}
