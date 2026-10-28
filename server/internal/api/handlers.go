package api

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kbremont/531-tracker/server/internal/training"
)

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, r, http.StatusOK, Info{APIVersion: APIVersion, Version: h.version})
}

// login exchanges the password for a session token. It is the same token the
// browser keeps in its cookie, so the two sign in and expire alike.
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in LoginRequest
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if !h.sessions.CheckPassword(in.Password) {
		h.writeJSON(w, r, http.StatusUnauthorized, ErrorResponse{Error: "Incorrect password."})
		return
	}
	token, expires := h.sessions.Issue()
	h.writeJSON(w, r, http.StatusOK, LoginResponse{Token: token, ExpiresAt: expires.UTC().Format(time.RFC3339)})
}

// dashboard answers the Today screen. Before training maxes are entered it
// reports needsSetup, where the web redirects to the setup page.
func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Dashboard(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toDashboard(d))
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	hist, err := h.svc.History(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toHistory(hist))
}

func (h *Handler) lift(w http.ResponseWriter, r *http.Request) {
	h.sendLift(w, r, r.PathValue("slug"))
}

func (h *Handler) sendLift(w http.ResponseWriter, r *http.Request, slug string) {
	l, err := h.svc.Lift(r.Context(), slug)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toLift(l))
}

func (h *Handler) addPastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var in PastEstimateInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.AddPastEstimate(r.Context(), slug, in.Date, in.Weight, in.Reps); err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendLift(w, r, slug)
}

func (h *Handler) deletePastEstimate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	id, err := pathID(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.DeletePastEstimate(r.Context(), slug, id); err != nil {
		h.fail(w, r, err)
		return
	}
	h.sendLift(w, r, slug)
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Settings(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, toSettings(s))
}

func (h *Handler) saveSettings(w http.ResponseWriter, r *http.Request) {
	var in SettingsInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	change := training.SettingsInput{
		Rounding:          in.Rounding,
		BarWeight:         in.BarWeight,
		Plates:            in.Plates,
		Warmups:           in.Warmups,
		RestWarmupSeconds: in.RestWarmupSeconds,
		RestMainSeconds:   in.RestMainSeconds,
		RestSuppSeconds:   in.RestSuppSeconds,
		BBB:               make(map[string]training.LiftBBB, len(in.BBB)),
	}
	for slug, b := range in.BBB {
		change.BBB[slug] = training.LiftBBB{Mode: b.Mode, Value: b.Value}
	}
	if err := h.svc.SaveSettings(r.Context(), change); err != nil {
		h.fail(w, r, err)
		return
	}
	h.settings(w, r)
}

func (h *Handler) setTrainingMax(w http.ResponseWriter, r *http.Request) {
	var in TrainingMaxInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.SetTrainingMax(r.Context(), in.Lift, in.Value, in.Note); err != nil {
		h.fail(w, r, err)
		return
	}
	h.settings(w, r)
}

// acceptReview writes the training maxes settled on at the start of a block.
// A review no longer pending is a 409.
func (h *Handler) acceptReview(w http.ResponseWriter, r *http.Request) {
	var in TMReviewInput
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.AcceptReview(r.Context(), in.Key, in.TMs); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

// decideTMTest settles a failed TM test. Any action but keep lowers, as on
// the web form.
func (h *Handler) decideTMTest(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var in TMTestDecision
	if err := decode(w, r, &in); err != nil {
		h.fail(w, r, err)
		return
	}
	if err := h.svc.DecideTMTest(r.Context(), id, in.Action == "keep", in.TM); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

func (h *Handler) skipCycle(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SkipCycle(r.Context()); err != nil {
		h.fail(w, r, err)
		return
	}
	h.dashboard(w, r)
}

// backup streams a consistent snapshot of the database, as the web's
// download does.
func (h *Handler) backup(w http.ResponseWriter, r *http.Request) {
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
