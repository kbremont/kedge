// Package server wires the HTTP routes to the program engine and the
// database. Anything that can be expressed as a pure function lives in
// package program instead; what remains here is orchestration.
package server

import (
	"context"
	"database/sql"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/kbremont/531-tracker/internal/assets"
	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/views"
)

// Config holds everything the server needs from its environment.
type Config struct {
	Password string
	Version  string
}

type Server struct {
	conn     *sql.DB
	q        *db.Queries
	log      *slog.Logger
	secret   []byte
	password string
}

func New(conn *sql.DB, log *slog.Logger, cfg Config) *Server {
	views.AssetVersion = cfg.Version
	return &Server{
		conn:     conn,
		q:        db.New(conn),
		log:      log,
		secret:   sessionSecret(cfg.Password),
		password: cfg.Password,
	}
}

// Handler returns the fully routed application.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public routes.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.conn.PingContext(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /login", s.handleLogin)
	mux.HandleFunc("POST /login", s.handleLogin)
	mux.HandleFunc("GET /manifest.webmanifest", s.handleManifest)
	mux.Handle("GET /static/", s.staticHandler())

	// Application routes.
	app := http.NewServeMux()
	app.HandleFunc("GET /{$}", s.handleDashboard)
	app.HandleFunc("GET /setup", s.handleSetup)
	app.HandleFunc("POST /setup", s.handleSetupSave)
	app.HandleFunc("GET /setup/position", s.handleSetupPosition)
	app.HandleFunc("POST /setup/position", s.handleSetupPositionSave)
	app.HandleFunc("GET /setup/history", s.handleSetupHistory)
	app.HandleFunc("POST /setup/history", s.handleSetupHistorySave)
	app.HandleFunc("POST /logout", s.handleLogout)

	app.HandleFunc("POST /workout/start", s.handleStartWorkout)
	app.HandleFunc("GET /workout/{id}", s.handleWorkout)
	app.HandleFunc("POST /workout/{id}/set/{idx}", s.handleLogSet)
	app.HandleFunc("POST /workout/{id}/set/{idx}/undo", s.handleUndoSet)
	app.HandleFunc("POST /workout/{id}/finish", s.handleFinishWorkout)
	app.HandleFunc("POST /workout/{id}/abandon", s.handleAbandonWorkout)

	app.HandleFunc("GET /history", s.handleHistory)
	app.HandleFunc("GET /lift/{slug}", s.handleLift)
	app.HandleFunc("GET /settings", s.handleSettings)
	app.HandleFunc("POST /settings", s.handleSaveSettings)
	app.HandleFunc("POST /training-max", s.handleSetTrainingMax)
	app.HandleFunc("POST /dismiss-suggestions", s.handleDismissSuggestions)
	app.HandleFunc("GET /backup", s.handleBackup)

	mux.Handle("/", s.requireAuth(app))
	return s.recoverPanic(mux)
}

func (s *Server) staticHandler() http.Handler {
	sub, err := fs.Sub(assets.FS, "static")
	if err != nil {
		panic(err) // the embed path is a compile-time constant
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assets are content-addressed by the ?v= query the templates append,
		// so they can be cached hard.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	}))
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Write([]byte(`{
  "name": "5/3/1 Tracker",
  "short_name": "5/3/1",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "orientation": "portrait",
  "background_color": "#0d0f12",
  "theme_color": "#0d0f12",
  "icons": [
    { "src": "/static/icon-192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/static/icon-512.png", "sizes": "512x512", "type": "image/png", "purpose": "any maskable" }
  ]
}`))
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		s.log.ErrorContext(r.Context(), "render failed", "path", r.URL.Path, "error", err)
	}
}

// fail logs the underlying error and shows the user something generic; the
// detail belongs in the log, not the page.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	s.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "error", err)
	http.Error(w, "Something went wrong. Check the server log.", http.StatusInternalServerError)
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic serving request", "path", r.URL.Path, "panic", rec)
				http.Error(w, "Something went wrong. Check the server log.", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// tx runs fn inside a transaction, rolling back on error.
func (s *Server) tx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}
