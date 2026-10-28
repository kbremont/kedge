// Package api is the JSON API the iOS app talks to, under /api/v1. Like the
// pages in package web, its handlers only translate: they decode a request,
// call package training, and encode what it returns. Neither makes a training
// decision of its own.
//
// The wire types in types.go are the contract with the app. They are kept
// apart from training's read models, so renaming a Go field cannot silently
// break an installed app, and `make generate` turns them into the app's
// TypeScript types.
//
// A request that changes something answers with the screen it leads to, the
// same one the web form redirects to, so the app can show it without a
// second request.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/kbremont/531-tracker/server/internal/auth"
	"github.com/kbremont/531-tracker/server/internal/training"
)

// APIVersion is the version of the API under /api/v1, reported by info. It
// only changes for a change that breaks an installed app.
const APIVersion = 1

// maxBodyBytes bounds a request body. The largest real one, the settings, is
// well under a kilobyte.
const maxBodyBytes = 1 << 20

// Handler serves the API. Each unexported method is one route.
type Handler struct {
	svc      *training.Service
	sessions *auth.Sessions
	log      *slog.Logger
	version  string
}

// New returns the API over a training service. version is the server's build,
// reported by info.
func New(svc *training.Service, sessions *auth.Sessions, log *slog.Logger, version string) *Handler {
	return &Handler{svc: svc, sessions: sessions, log: log, version: version}
}

// Routes returns the API, to be mounted at /api/. Everything but info and
// login needs a session token, and any other path under /api/ is a JSON 404.
func (h *Handler) Routes() http.Handler {
	app := http.NewServeMux()
	app.HandleFunc("GET /api/v1/dashboard", h.dashboard)

	app.HandleFunc("POST /api/v1/workouts", h.startWorkout)
	app.HandleFunc("GET /api/v1/workouts/{id}", h.workout)
	app.HandleFunc("PUT /api/v1/workouts/{id}/sets/{idx}", h.logSet)
	app.HandleFunc("DELETE /api/v1/workouts/{id}/sets/{idx}", h.undoSet)
	app.HandleFunc("POST /api/v1/workouts/{id}/finish", h.finishWorkout)
	app.HandleFunc("DELETE /api/v1/workouts/{id}", h.abandonWorkout)

	app.HandleFunc("GET /api/v1/history", h.history)
	app.HandleFunc("GET /api/v1/lifts/{slug}", h.lift)
	app.HandleFunc("POST /api/v1/lifts/{slug}/estimates", h.addPastEstimate)
	app.HandleFunc("DELETE /api/v1/lifts/{slug}/estimates/{id}", h.deletePastEstimate)

	app.HandleFunc("GET /api/v1/settings", h.settings)
	app.HandleFunc("PUT /api/v1/settings", h.saveSettings)
	app.HandleFunc("POST /api/v1/training-maxes", h.setTrainingMax)
	app.HandleFunc("POST /api/v1/tm-review", h.acceptReview)
	app.HandleFunc("POST /api/v1/tm-tests/{id}", h.decideTMTest)
	app.HandleFunc("POST /api/v1/skip-cycle", h.skipCycle)

	app.HandleFunc("GET /api/v1/setup", h.setup)
	app.HandleFunc("PUT /api/v1/setup", h.saveSetup)
	app.HandleFunc("GET /api/v1/setup/position", h.setupPosition)
	app.HandleFunc("GET /api/v1/setup/history/{cycle}", h.setupHistory)
	app.HandleFunc("PUT /api/v1/setup/history/{cycle}", h.saveSetupHistory)

	app.HandleFunc("GET /api/v1/backup", h.backup)

	app.HandleFunc("/api/", h.notFound)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/info", h.info)
	mux.HandleFunc("POST /api/v1/login", h.login)
	mux.Handle("/api/", h.sessions.RequireAPI(app))
	return noStore(mux)
}

// noStore keeps every answer out of caches: each one is the user's training,
// as it stands right now.
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// writeJSON answers with v as JSON.
func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.ErrorContext(r.Context(), "encode response", "path", r.URL.Path, "error", err)
	}
}

// errBadJSON is a request body that is not the JSON the route expects.
var errBadJSON = errors.New("bad request body")

// decode reads a JSON request body into v. Fields it does not know are
// ignored, so an app newer than the server still works.
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return errBadJSON
	}
	return nil
}

// pathID reads a numeric path segment. One that is not a number is a 404,
// since no such thing can exist.
func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, training.ErrNotFound
	}
	return id, nil
}
