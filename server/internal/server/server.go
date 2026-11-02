// Package server assembles the application: the routes, authentication,
// /healthz and panic recovery. The pages live in package web, the app's JSON
// API in package api, and the rules both apply in package training.
package server

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"github.com/kbremont/kedge/server/internal/api"
	"github.com/kbremont/kedge/server/internal/auth"
	"github.com/kbremont/kedge/server/internal/training"
	"github.com/kbremont/kedge/server/internal/web"
)

// Config holds everything the server needs from its environment.
type Config struct {
	Password string
	Version  string
}

type Server struct {
	conn     *sql.DB
	log      *slog.Logger
	sessions *auth.Sessions
	pages    *web.Handler
	api      *api.Server
}

func New(conn *sql.DB, log *slog.Logger, cfg Config) *Server {
	sessions := auth.New(cfg.Password)
	svc := training.New(conn, log)
	return &Server{
		conn:     conn,
		log:      log,
		sessions: sessions,
		pages:    web.New(svc, sessions, log, cfg.Version),
		api:      api.New(svc, sessions, log, cfg.Version),
	}
}

// Handler returns the fully routed application.
func (s *Server) Handler() http.Handler {
	pages := s.pages
	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /login", pages.Login)
	mux.HandleFunc("POST /login", pages.Login)
	mux.HandleFunc("GET /manifest.webmanifest", pages.Manifest)
	mux.Handle("GET /static/", pages.Static())

	// Application routes.
	app := http.NewServeMux()
	app.HandleFunc("GET /{$}", pages.Dashboard)
	app.HandleFunc("GET /setup", pages.Setup)
	app.HandleFunc("POST /setup", pages.SaveSetup)
	app.HandleFunc("GET /setup/position", pages.SetupPosition)
	app.HandleFunc("POST /setup/position", pages.SaveSetupPosition)
	app.HandleFunc("GET /setup/history", pages.SetupHistory)
	app.HandleFunc("POST /setup/history", pages.SaveSetupHistory)
	app.HandleFunc("POST /logout", pages.Logout)

	app.HandleFunc("POST /workout/start", pages.StartWorkout)
	app.HandleFunc("GET /workout/{id}", pages.Workout)
	app.HandleFunc("POST /workout/{id}/set/{idx}", pages.LogSet)
	app.HandleFunc("POST /workout/{id}/set/{idx}/undo", pages.UndoSet)
	app.HandleFunc("POST /workout/{id}/finish", pages.FinishWorkout)
	app.HandleFunc("POST /workout/{id}/abandon", pages.AbandonWorkout)

	app.HandleFunc("GET /history", pages.History)
	app.HandleFunc("GET /lift/{slug}", pages.Lift)
	app.HandleFunc("POST /lift/{slug}/estimates", pages.AddPastEstimate)
	app.HandleFunc("POST /lift/{slug}/estimates/{id}/delete", pages.DeletePastEstimate)
	app.HandleFunc("GET /settings", pages.Settings)
	app.HandleFunc("POST /settings", pages.SaveSettings)
	app.HandleFunc("POST /training-max", pages.SetTrainingMax)
	app.HandleFunc("POST /tm-review", pages.AcceptReview)
	app.HandleFunc("POST /tm-test/{id}", pages.DecideTMTest)
	app.HandleFunc("POST /skip-cycle", pages.SkipCycle)
	app.HandleFunc("GET /backup", pages.Backup)

	// The app's API checks its own sessions, answering a JSON 401 where a
	// page would redirect to /login. As the longer pattern it wins over "/".
	mux.Handle("/api/", s.api.Routes())

	mux.Handle("/", s.sessions.RequirePage(app))
	return s.recoverPanic(mux)
}

// healthz reports whether the database answers, for the container's health
// check.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if err := s.conn.PingContext(r.Context()); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic serving request", "path", r.URL.Path, "panic", rec)
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"Something went wrong. Check the server log."}` + "\n"))
					return
				}
				http.Error(w, "Something went wrong. Check the server log.", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
