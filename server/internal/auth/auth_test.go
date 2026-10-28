package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	s := New("secret")
	hour := time.Now().Add(time.Hour)

	cases := []struct {
		name  string
		token string
		want  bool
	}{
		{"signed", s.Sign(hour), true},
		{"expired", s.Sign(time.Now().Add(-time.Second)), false},
		{"other password", New("other").Sign(hour), false},
		{"tampered expiry", "9" + s.Sign(hour), false},
		{"no signature", strings.Split(s.Sign(hour), ".")[0], false},
		{"empty", "", false},
		{"junk", "abc.def", false},
	}
	for _, tc := range cases {
		if got := s.Valid(tc.token); got != tc.want {
			t.Errorf("%s: Valid = %v, want %v", tc.name, got, tc.want)
		}
	}

	token, expires := s.Issue()
	if !s.Valid(token) {
		t.Error("an issued token is not valid")
	}
	if d := time.Until(expires); d < TTL-time.Minute || d > TTL {
		t.Errorf("issued token expires in %v, want %v", d, TTL)
	}
}

// TestTokenFormat pins the token to the format the server has always issued,
// so signed-in browsers stay signed in across upgrades.
func TestTokenFormat(t *testing.T) {
	const want = "2000000000.4b4175492e77689dbe875a040005553898d5ef70bf72f1c41f992104346070d9"
	if got := New("secret").Sign(time.Unix(2000000000, 0)); got != want {
		t.Errorf("Sign = %q, want %q", got, want)
	}
}

func TestCheckPassword(t *testing.T) {
	s := New("secret")
	if !s.CheckPassword("secret") || s.CheckPassword("Secret") || s.CheckPassword("") {
		t.Error("CheckPassword accepts only the exact password")
	}
	if New("").CheckPassword("") {
		t.Error("an empty password must never sign anyone in")
	}
}

func TestAuthenticated(t *testing.T) {
	s := New("secret")
	good := s.Sign(time.Now().Add(time.Hour))

	cases := []struct {
		name string
		req  func(*http.Request)
		want bool
	}{
		{"nothing", func(*http.Request) {}, false},
		{"cookie", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: CookieName, Value: good}) }, true},
		{"bad cookie", func(r *http.Request) { r.AddCookie(&http.Cookie{Name: CookieName, Value: "x"}) }, false},
		{"bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+good) }, true},
		{"bearer lower case", func(r *http.Request) { r.Header.Set("Authorization", "bearer "+good) }, true},
		{"bad bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer x") }, false},
		{"basic", func(r *http.Request) { r.Header.Set("Authorization", "Basic "+good) }, false},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		tc.req(r)
		if got := s.Authenticated(r); got != tc.want {
			t.Errorf("%s: Authenticated = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestRequire(t *testing.T) {
	s := New("secret")
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("in")) })

	rec := httptest.NewRecorder()
	s.RequirePage(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/settings", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Errorf("page without a session = %d to %q, want a redirect to /login", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	s.RequireAPI(ok).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil))
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Content-Type") != "application/json" ||
		!strings.Contains(rec.Body.String(), `"error"`) {
		t.Errorf("API without a session = %d %q %q, want a JSON 401", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}

	for _, guard := range []func(http.Handler) http.Handler{s.RequirePage, s.RequireAPI} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("Authorization", "Bearer "+s.Sign(time.Now().Add(time.Hour)))
		rec := httptest.NewRecorder()
		guard(ok).ServeHTTP(rec, r)
		if rec.Body.String() != "in" {
			t.Errorf("a valid session was turned away: %d", rec.Code)
		}
	}
}

func TestCookies(t *testing.T) {
	s := New("secret")

	cases := []struct {
		name   string
		req    func(*http.Request)
		secure bool
	}{
		{"plain", func(*http.Request) {}, false},
		{"behind an HTTPS proxy", func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") }, true},
		{"direct TLS", func(r *http.Request) { r.TLS = &tls.ConnectionState{} }, true},
	}
	for _, tc := range cases {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		tc.req(r)

		rec := httptest.NewRecorder()
		s.SetCookie(rec, r)
		c := rec.Result().Cookies()[0]
		if c.Name != CookieName || !s.Valid(c.Value) || !c.HttpOnly || c.Secure != tc.secure || c.SameSite != http.SameSiteLaxMode {
			t.Errorf("%s: set cookie %+v", tc.name, c)
		}

		rec = httptest.NewRecorder()
		s.ClearCookie(rec, r)
		c = rec.Result().Cookies()[0]
		if c.Name != CookieName || c.Value != "" || c.MaxAge >= 0 || c.Secure != tc.secure {
			t.Errorf("%s: cleared cookie %+v", tc.name, c)
		}
	}
}
