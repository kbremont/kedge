package server

import (
	"fmt"
	"net/http"

	"github.com/kbremont/531-tracker/server/internal/views"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.sessions.Authenticated(r) {
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
	if !s.sessions.CheckPassword(r.PostFormValue("password")) {
		w.WriteHeader(http.StatusUnauthorized)
		s.render(w, r, views.Login("Incorrect password."))
		return
	}
	s.sessions.SetCookie(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.ClearCookie(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
