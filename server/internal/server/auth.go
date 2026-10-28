package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kbremont/531-tracker/server/internal/views"
)

const (
	sessionCookie = "s531"
	sessionTTL    = 60 * 24 * time.Hour // a personal app should not log you out at the gym
)

// sessionSecret is derived from the password so that no second secret has to
// be configured, and so that changing the password invalidates old sessions.
func sessionSecret(password string) []byte {
	sum := sha256.Sum256([]byte("531-tracker/session/v1:" + password))
	return sum[:]
}

func (s *Server) signSession(expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(exp))
	return exp + "." + hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) validSession(value string) bool {
	exp, sig, ok := strings.Cut(value, ".")
	if !ok {
		return false
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(exp))
	want := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(sig), []byte(want)) != 1 {
		return false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Before(time.Unix(unix, 0))
}

func (s *Server) authed(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	return s.validSession(c.Value)
}

// secureRequest reports whether the connection reaching the user is HTTPS.
// Behind a terminating proxy such as Tailscale Serve the local connection is
// plain HTTP, so the forwarded header is the only signal available.
func secureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request) {
	expires := time.Now().Add(sessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.signSession(expires),
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// requireAuth wraps the application routes. Unauthenticated requests are sent
// to the sign-in page rather than given a bare 401, since every route here is
// reached from a browser.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authed(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// checkPassword compares in constant time so the endpoint does not leak the
// password's prefix through timing.
func (s *Server) checkPassword(got string) bool {
	if len(s.password) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.password)) == 1
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.authed(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		s.render(w, r, views.Login(""))
		return
	}

	if err := r.ParseForm(); err != nil {
		s.fail(w, r, fmt.Errorf("parse login form: %w", err))
		return
	}
	if !s.checkPassword(r.PostFormValue("password")) {
		w.WriteHeader(http.StatusUnauthorized)
		s.render(w, r, views.Login("Incorrect password."))
		return
	}
	s.setSessionCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
