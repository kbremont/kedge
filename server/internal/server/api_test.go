package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestDemoKeepsNoTrainingMaxNotes sends a training max with a note through
// the API. The public demo keeps the value and drops the note, which every
// visitor after would otherwise see on the lift's page; a lifter's own server
// keeps both.
func TestDemoKeepsNoTrainingMaxNotes(t *testing.T) {
	for _, c := range []struct {
		name     string
		demo     bool
		wantNote string
	}{
		{"own server", false, "left by a visitor"},
		{"demo", true, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newHarnessWith(t, Config{Password: "test", Demo: c.demo})
			h.setup(map[string]float64{"press": 115, "deadlift": 365, "bench": 205, "squat": 305}, 0)

			body := `{"lift": "press", "value": 120, "note": "left by a visitor"}`
			req := httptest.NewRequest(http.MethodPost, "/api/v1/training-maxes", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.AddCookie(h.cookie)
			rec := httptest.NewRecorder()
			h.h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("set training max: %d %s", rec.Code, rec.Body.String())
			}

			history, err := h.q.ListTrainingMaxHistory(context.Background(), "press")
			if err != nil || len(history) == 0 {
				t.Fatalf("history = %v, %v", history, err)
			}
			if got := history[0]; got.Value != 120 || got.Note != c.wantNote {
				t.Errorf("latest training max %v with note %q, want 120 with %q", got.Value, got.Note, c.wantNote)
			}
		})
	}
}
