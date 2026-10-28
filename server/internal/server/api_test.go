package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAPIMountedBesidePages checks the API is reached through the whole
// stack: info answers without a session, and the rest answer a JSON 401
// rather than the redirect to /login a page gets.
func TestAPIMountedBesidePages(t *testing.T) {
	h := newHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rec := httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	var info struct {
		APIVersion int `json:"apiVersion"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &info) != nil || info.APIVersion != 1 {
		t.Errorf("info: %d %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	rec = httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("dashboard without a session: %d %q, want a JSON 401", rec.Code, rec.Header().Get("Content-Type"))
	}

	// The page still redirects.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Errorf("page without a session: %d to %q, want a redirect to /login", rec.Code, rec.Header().Get("Location"))
	}

	// And the browser's session reaches the API.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.AddCookie(h.cookie)
	rec = httptest.NewRecorder()
	h.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("dashboard with the session cookie: %d", rec.Code)
	}
}
