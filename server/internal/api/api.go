// Package api is the JSON API the iOS app talks to, under /api/v1. Like the
// pages in package web, its handlers only translate: they take a request,
// call package training, and convert what it returns. Neither makes a
// training decision of its own.
//
// The contract with the app is the OpenAPI spec in api/openapi.yaml, at the
// root of the module. oapi.gen.go is generated from it by `make generate`:
// the wire types, StrictServerInterface with one typed method per operation,
// which Server implements, a router, and the spec itself. Every request is
// checked against the spec before a handler sees it. The wire types are kept
// apart from training's read models, so renaming a Go field cannot silently
// break an installed app.
//
// A request that changes something answers with the screen it leads to, the
// same one the web form redirects to, so the app can show it without a
// second request.
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/kbremont/kedge/server/internal/auth"
	"github.com/kbremont/kedge/server/internal/training"
)

// APIVersion is the version of the API under /api/v1, reported by info. It
// only changes for a change that breaks an installed app, and it is the
// spec's info.version.
const APIVersion = 1

// basePath is where the API is served, the spec's only server.
const basePath = "/api/v1"

// maxBodyBytes bounds a request body. The largest real one, the settings, is
// well under a kilobyte.
const maxBodyBytes = 1 << 20

// Server serves the API. Each exported method is one operation in the spec.
type Server struct {
	svc      *training.Service
	sessions *auth.Sessions
	log      *slog.Logger
	version  string
}

var _ StrictServerInterface = (*Server)(nil)

// New returns the API over a training service. version is the server's build,
// reported by info.
func New(svc *training.Service, sessions *auth.Sessions, log *slog.Logger, version string) *Server {
	return &Server{svc: svc, sessions: sessions, log: log, version: version}
}

// Routes returns the API, to be mounted at /api/. The operations the spec
// marks public, with security: [], need no session, and neither does the
// spec itself at /api/v1/openapi.json. Anything else under /api/ needs a
// session token first, so without one even a path that is not a route is a
// 401. Then the request is checked against the spec, and only one it allows
// reaches a handler.
func (s *Server) Routes() http.Handler {
	spec, err := GetSpec()
	if err != nil {
		panic(fmt.Sprintf("api: load the embedded spec: %v", err))
	}
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		panic(fmt.Sprintf("api: route the embedded spec: %v", err))
	}

	ops := HandlerWithOptions(NewStrictHandlerWithOptions(s, nil, StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  s.unreadable,
		ResponseErrorHandlerFunc: s.fail,
	}), StdHTTPServerOptions{
		BaseURL:          basePath,
		BaseRouter:       http.NewServeMux(),
		ErrorHandlerFunc: s.unreadable,
	})
	validate := nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			// The session was checked before the request got here.
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		ErrorHandlerWithOpts:  s.invalid(router),
		SilenceServersWarning: true, // the server is a path, with no host to check
	})
	api := http.MaxBytesHandler(validate(ops), maxBodyBytes)

	mux := http.NewServeMux()
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if op.Security != nil && len(*op.Security) == 0 {
				mux.Handle(method+" "+basePath+path, api)
			}
		}
	}
	mux.HandleFunc("GET "+basePath+"/openapi.json", s.openAPI)
	mux.Handle("/api/", s.sessions.RequireAPI(api))
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

// openAPI serves the spec as JSON, for docs viewers such as Swagger UI or
// Scalar. It holds nothing private, so it needs no session and any origin
// may read it.
func (s *Server) openAPI(w http.ResponseWriter, r *http.Request) {
	spec, err := GetSpecJSON()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(spec)
}

// writeJSON answers with v as JSON.
func (s *Server) writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.ErrorContext(r.Context(), "encode response", "path", r.URL.Path, "error", err)
	}
}

// deref reads an optional request field, which is its zero value when left
// out.
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
