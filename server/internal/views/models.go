// Package views renders the HTML. Templates render the read models from
// package training, plus the little that only a page needs: a flag from the
// query string, or a form error to show.
package views

import "github.com/kbremont/531-tracker/server/internal/training"

// Nav marks which top-level section is active.
type Nav string

const (
	NavHome     Nav = "home"
	NavHistory  Nav = "history"
	NavSettings Nav = "settings"
)

// WorkoutPage is the session screen.
type WorkoutPage struct {
	training.Workout
	// ShowRest shows the rest timer above the current set, after a set has
	// just been logged.
	ShowRest bool
}

// LiftPage is the per-lift detail screen.
type LiftPage struct {
	training.Lift
	// Error is set when the last past estimate submitted was rejected.
	Error bool
}

// SettingsPage is the settings screen.
type SettingsPage struct {
	training.Settings
	// Saved confirms the last change.
	Saved bool
}

// SetupPage is the first-run screen where training maxes are entered.
type SetupPage struct {
	training.Setup
	Error string
}
