package api

import (
	"errors"
	"net/http"

	"github.com/kbremont/531-tracker/server/internal/training"
)

// fail answers with the status and message for an error. Where the web
// quietly goes back to the dashboard, the API says why nothing happened, so
// the app can tell the user and fetch the dashboard again.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := errorResponse(err)
	if status == http.StatusInternalServerError {
		// The detail belongs in the log, not the app.
		h.log.ErrorContext(r.Context(), "request failed", "path", r.URL.Path, "method", r.Method, "error", err)
	}
	h.writeJSON(w, r, status, ErrorResponse{Error: msg})
}

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

// notFound answers any path under /api/ that is not a route.
func (h *Handler) notFound(w http.ResponseWriter, r *http.Request) {
	h.fail(w, r, training.ErrNotFound)
}
