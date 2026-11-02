package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/kbremont/kedge/server/internal/training"
)

// fail answers with the status and message for an error. Where the web
// quietly goes back to the dashboard, the API says why nothing happened, so
// the app can tell the user and fetch the dashboard again.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := errorResponse(err)
	if status == http.StatusInternalServerError {
		// The detail belongs in the log, not the app.
		s.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "error", err)
	}
	s.writeJSON(w, r, status, ErrorResponse{Error: msg})
}

// errBadJSON is a request body that is not the JSON the route expects.
var errBadJSON = errors.New("bad request body")

// errorResponse maps an error from training, or from reading the request, to
// a status and a message that can be shown as it is.
func errorResponse(err error) (status int, msg string) {
	var input *training.InputError
	switch {
	case errors.As(err, &input):
		return http.StatusBadRequest, input.Msg
	case errors.Is(err, errBadJSON):
		return http.StatusBadRequest, "The request body is not valid JSON for this route."
	case errors.Is(err, training.ErrNotFound), errors.Is(err, training.ErrUnknownCycle):
		return http.StatusNotFound, "Not found."
	case errors.Is(err, training.ErrHeld):
		return http.StatusConflict, "Training is waiting on a training max decision."
	case errors.Is(err, training.ErrStale):
		return http.StatusConflict, "That is no longer on offer."
	case errors.Is(err, training.ErrNothingScheduled):
		return http.StatusConflict, "There is no session to start."
	case errors.Is(err, training.ErrNeedsSetup):
		return http.StatusConflict, "Enter your training maxes first."
	}
	return http.StatusInternalServerError, "Something went wrong. Check the server log."
}

// invalid answers a request the spec does not allow, before any handler sees
// it: a path that is not a route is a 404, a route asked for with a method
// it does not have a 405, and a body that does not match a 400 saying why.
// router finds the methods a 405 allows.
func (s *Server) invalid(router routers.Router) nethttpmiddleware.ErrorHandlerWithOpts {
	return func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
		var reqErr *openapi3filter.RequestError
		switch {
		case opts.MatchedRoute == nil && errors.Is(err, routers.ErrMethodNotAllowed):
			w.Header().Set("Allow", strings.Join(allowed(router, r), ", "))
			s.writeJSON(w, r, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed."})
		case opts.MatchedRoute == nil:
			s.fail(w, r, training.ErrNotFound)
		case errors.As(err, &reqErr) && reqErr.Parameter != nil && reqErr.Parameter.In == openapi3.ParameterInPath:
			// A path segment that is not a number, say, names nothing that
			// could exist.
			s.fail(w, r, training.ErrNotFound)
		case errors.As(err, &reqErr):
			s.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{Error: invalidBody(reqErr)})
		default:
			s.fail(w, r, err)
		}
	}
}

// invalidBody says what is wrong with a request body in a sentence: the
// property and the reason, never the schema.
func invalidBody(err *openapi3filter.RequestError) string {
	var schemaErr *openapi3.SchemaError
	var parseErr *openapi3filter.ParseError
	switch {
	case errors.As(err, &schemaErr):
		if at := schemaErr.JSONPointer(); len(at) > 0 {
			return fmt.Sprintf("The request body is not valid: %s at /%s.", schemaErr.Reason, strings.Join(at, "/"))
		}
		return fmt.Sprintf("The request body is not valid: %s.", schemaErr.Reason)
	case errors.Is(err, openapi3filter.ErrInvalidRequired):
		return "The request body is missing."
	case errors.As(err, &parseErr), err.Reason == "":
		return "The request body is not valid JSON for this route."
	}
	// Such as a Content-Type other than application/json.
	return fmt.Sprintf("The request body is not valid: %s.", err.Reason)
}

// allowed lists the methods the spec has for a request's path.
func allowed(router routers.Router, r *http.Request) []string {
	var methods []string
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		probe := r.WithContext(r.Context()) // a shallow copy
		probe.Method = method
		if _, _, err := router.FindRoute(probe); err == nil {
			methods = append(methods, method)
		}
	}
	return methods
}

// unreadable answers a request the generated code could not read: a body
// that will not decode into its type, or a path parameter that will not
// bind. Both have already been checked against the spec, so this is a body
// that matched it and still did not fit, such as a count too large for an
// int.
func (s *Server) unreadable(w http.ResponseWriter, r *http.Request, err error) {
	var param *InvalidParamFormatError
	if errors.As(err, &param) {
		s.fail(w, r, training.ErrNotFound)
		return
	}
	s.fail(w, r, fmt.Errorf("%w: %v", errBadJSON, err))
}
