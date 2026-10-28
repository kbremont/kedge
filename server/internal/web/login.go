package web

import (
	"fmt"
	"net/http"
)

// Login shows the sign-in page, or on POST checks the password and signs the
// browser in.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if h.sessions.Authenticated(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		h.render(w, r, Login(""))
		return
	}

	if err := r.ParseForm(); err != nil {
		h.fail(w, r, fmt.Errorf("parse login form: %w", err))
		return
	}
	if !h.sessions.CheckPassword(r.PostFormValue("password")) {
		w.WriteHeader(http.StatusUnauthorized)
		h.render(w, r, Login("Incorrect password."))
		return
	}
	h.sessions.SetCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout signs the browser out.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.ClearCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
