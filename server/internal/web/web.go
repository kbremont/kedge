// Package web is the browser UI: thin handlers that read a form, call package
// training and redirect or render, the templ templates, and the embedded
// static files. Templates render training's read models, plus the little that
// only a page needs: a flag from the query string, or a form error to show.
package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/kbremont/kedge/server/internal/auth"
	"github.com/kbremont/kedge/server/internal/training"
)

// static holds the CSS and icons, so the compiled binary is the whole
// deployment.
//
//go:embed static
var static embed.FS

// Handler serves the pages. Each exported method is one route.
type Handler struct {
	svc      *training.Service
	sessions *auth.Sessions
	log      *slog.Logger
}

// New returns the pages for a training service. version is appended to
// static asset URLs so a deploy busts the cache.
func New(svc *training.Service, sessions *auth.Sessions, log *slog.Logger, version string) *Handler {
	AssetVersion = version
	return &Handler{svc: svc, sessions: sessions, log: log}
}

// Static serves the embedded static files under /static/.
func (h *Handler) Static() http.Handler {
	sub, err := fs.Sub(static, "static")
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

// Manifest serves the web app manifest that lets the site install to the
// home screen.
func (h *Handler) Manifest(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		h.log.ErrorContext(r.Context(), "render failed", "path", r.URL.Path, "error", err)
	}
}

// fail logs the underlying error and shows the user something generic; the
// detail belongs in the log, not the page.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	h.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "error", err)
	http.Error(w, "Something went wrong. Check the server log.", http.StatusInternalServerError)
}
