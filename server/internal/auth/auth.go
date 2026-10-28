// Package auth signs and checks the session tokens that keep the user signed
// in. A browser carries the token in a cookie; the app sends it as a Bearer
// token. Both are the same token, so one password and one secret cover both.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// CookieName is the browser session cookie.
	CookieName = "s531"
	// TTL is how long a session lasts. A personal app should not log you out
	// at the gym.
	TTL = 60 * 24 * time.Hour
)

// Sessions issues and checks session tokens. A token is the expiry as a Unix
// time, a dot, and an HMAC of it, so nothing has to be stored to check one.
type Sessions struct {
	secret   []byte
	password string
}

// New returns the sessions for a password. The signing secret is derived
// from the password so that no second secret has to be configured, and so
// that changing the password invalidates every session.
func New(password string) *Sessions {
	sum := sha256.Sum256([]byte("531-tracker/session/v1:" + password))
	return &Sessions{secret: sum[:], password: password}
}

// CheckPassword compares in constant time so the sign-in does not leak the
// password's prefix through timing. An empty password never matches.
func (s *Sessions) CheckPassword(got string) bool {
	if len(s.password) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.password)) == 1
}

// Issue returns a new token and when it expires.
func (s *Sessions) Issue() (token string, expires time.Time) {
	expires = time.Now().Add(TTL)
	return s.Sign(expires), expires
}

// Sign returns a token that is valid until expires.
func (s *Sessions) Sign(expires time.Time) string {
	exp := strconv.FormatInt(expires.Unix(), 10)
	return exp + "." + s.mac(exp)
}

// Valid reports whether a token was signed with this password and has not
// expired.
func (s *Sessions) Valid(token string) bool {
	exp, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(sig), []byte(s.mac(exp))) != 1 {
		return false
	}
	unix, err := strconv.ParseInt(exp, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Before(time.Unix(unix, 0))
}

func (s *Sessions) mac(exp string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(exp))
	return hex.EncodeToString(mac.Sum(nil))
}

// Authenticated reports whether a request carries a valid token, either as
// the session cookie or as an Authorization: Bearer header.
func (s *Sessions) Authenticated(r *http.Request) bool {
	if scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " "); ok && strings.EqualFold(scheme, "Bearer") {
		return s.Valid(token)
	}
	c, err := r.Cookie(CookieName)
	if err != nil {
		return false
	}
	return s.Valid(c.Value)
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

// SetCookie signs the browser in.
func (s *Sessions) SetCookie(w http.ResponseWriter, r *http.Request) {
	token, expires := s.Issue()
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie signs the browser out.
func (s *Sessions) ClearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// RequirePage guards routes reached from a browser. Unauthenticated requests
// are sent to the sign-in page rather than given a bare 401.
func (s *Sessions) RequirePage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.Authenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAPI guards routes called by the app, which wants a status it can act
// on rather than a redirect to a page: a JSON 401 that sends it to sign in.
func (s *Sessions) RequireAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.Authenticated(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"Sign in to continue."}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
