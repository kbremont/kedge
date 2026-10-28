package api

import (
	"net/http"
)

// startWorkout starts the next session, or resumes the one in progress, and
// answers with it. When there is nothing to start, or a training max decision
// is waiting, it is a 409 and the app fetches the dashboard to show why.
func (h *Handler) startWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := h.svc.StartWorkout(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendWorkout(w, r, id)
}

func (h *Handler) workout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendWorkout(w, r, id)
}

func (h *Handler) sendWorkout(w http.ResponseWriter, r *http.Request, id int64) {
	workout, err := h.svc.Workout(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toWorkout(workout))
}

// logSet records the reps done on a set. The session it answers with carries
// the rest to time before the next set.
func (h *Handler) logSet(w http.ResponseWriter, r *http.Request) {
	id, idx, err := workoutAndSet(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var in LogSetInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.LogSet(r.Context(), id, idx, in.Reps); err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendWorkout(w, r, id)
}

func (h *Handler) undoSet(w http.ResponseWriter, r *http.Request) {
	id, idx, err := workoutAndSet(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.UndoSet(r.Context(), id, idx); err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendWorkout(w, r, id)
}

func (h *Handler) finishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.FinishWorkout(r.Context(), id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

// abandonWorkout removes a session started by mistake.
func (h *Handler) abandonWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.AbandonWorkout(r.Context(), id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

func workoutAndSet(r *http.Request) (id, idx int64, err error) {
	if id, err = pathID(r, "id"); err != nil {
		return 0, 0, err
	}
	if idx, err = pathID(r, "idx"); err != nil {
		return 0, 0, err
	}
	return id, idx, nil
}
