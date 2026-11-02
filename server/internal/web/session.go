package web

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kbremont/kedge/server/internal/training"
)

// StartWorkout starts the next scheduled session, or resumes the one in
// progress. When there is nothing to start, or a training max decision is
// waiting, the dashboard says why.
func (h *Handler) StartWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := h.svc.StartWorkout(r.Context())
	switch {
	case errors.Is(err, training.ErrNothingScheduled), errors.Is(err, training.ErrHeld):
		http.Redirect(w, r, "/", http.StatusSeeOther)
	case errors.Is(err, training.ErrNeedsSetup):
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
	case err != nil:
		h.fail(w, r, err)
	default:
		http.Redirect(w, r, fmt.Sprintf("/workout/%d", id), http.StatusSeeOther)
	}
}

func (h *Handler) Workout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	workout, err := h.svc.Workout(r.Context(), id)
	if err != nil {
		h.notFoundOrFail(w, r, err)
		return
	}
	h.render(w, r, Workout(WorkoutPage{Workout: *workout, ShowRest: r.URL.Query().Get("rest") == "1"}))
}

func (h *Handler) LogSet(w http.ResponseWriter, r *http.Request) {
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
	err = h.svc.LogSet(r.Context(), workoutID, idx, reps)
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		http.Redirect(w, r, back, http.StatusSeeOther)
	case err != nil:
		h.fail(w, r, err)
	default:
		h.redirectToNextSet(w, r, workoutID)
	}
}

func (h *Handler) UndoSet(w http.ResponseWriter, r *http.Request) {
	workoutID, idx, err := pathIDAndIdx(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.svc.UndoSet(r.Context(), workoutID, idx); err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d#set-%d", workoutID, idx), http.StatusSeeOther)
}

// redirectToNextSet sends the browser back to the session, anchored on the
// set now due. The page load doubles as the start of the rest timer.
func (h *Handler) redirectToNextSet(w http.ResponseWriter, r *http.Request, workoutID int64) {
	workout, err := h.svc.Workout(r.Context(), workoutID)
	if err != nil && !errors.Is(err, training.ErrNotFound) {
		h.fail(w, r, err)
		return
	}
	if workout != nil && workout.CurrentIdx >= 0 {
		http.Redirect(w, r, fmt.Sprintf("/workout/%d?rest=1#rest", workoutID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workout/%d", workoutID), http.StatusSeeOther)
}

func (h *Handler) FinishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.svc.FinishWorkout(r.Context(), id); err != nil {
		h.fail(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// AbandonWorkout removes a session that was started by mistake.
func (h *Handler) AbandonWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.svc.AbandonWorkout(r.Context(), id); err != nil {
		h.fail(w, r, err)
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
