package api

import (
	"net/http"
	"strconv"

	"github.com/kbremont/531-tracker/server/internal/training"
)

// Onboarding: training maxes first, then the cycle the user is on, then the
// sessions of it already done. Picking a cycle writes nothing, so it has no
// route; the app goes straight to that cycle's checklist, or to the
// dashboard to start fresh.

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Setup(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toSetup(s))
}

// saveSetup records the first training maxes and answers with the next
// question: where in the programme the user is.
func (h *Handler) saveSetup(w http.ResponseWriter, r *http.Request) {
	var in SetupInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.SaveSetup(r.Context(), in.TMs); err != nil {
		h.fail(w, r, err)
		return
	}
	h.setupPosition(w, r)
}

func (h *Handler) setupPosition(w http.ResponseWriter, r *http.Request) {
	pos, err := h.svc.SetupPosition(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toSetupPosition(pos))
}

func (h *Handler) setupHistory(w http.ResponseWriter, r *http.Request) {
	idx, err := pathCycle(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	hist, err := h.svc.SetupHistory(r.Context(), idx)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toSetupHistory(hist))
}

// saveSetupHistory joins the block at a cycle with the sessions already done,
// and answers with the dashboard that now picks up after them.
func (h *Handler) saveSetupHistory(w http.ResponseWriter, r *http.Request) {
	idx, err := pathCycle(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var in SetupHistoryInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	done := make([]training.PastSession, 0, len(in.Done))
	for _, d := range in.Done {
		done = append(done, training.PastSession{Round: d.Round, Slug: d.Slug, PRReps: d.PRReps})
	}
	if err := h.svc.SaveSetupHistory(r.Context(), idx, done); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

func pathCycle(r *http.Request) (int, error) {
	idx, err := strconv.Atoi(r.PathValue("cycle"))
	if err != nil {
		return 0, training.ErrUnknownCycle
	}
	return idx, nil
}
