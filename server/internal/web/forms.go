package web

import "fmt"

// Form field names, shared by the templates that draw the forms and the
// handlers that read them back.

// tmField is the input for a lift's training max, on setup and the block
// review.
func tmField(slug string) string { return "tm_" + slug }

// doneField is the onboarding checkbox for a session already done.
func doneField(round int, slug string) string { return fmt.Sprintf("done_r%d_%s", round, slug) }

// repsField is the onboarding input for that session's PR set reps.
func repsField(round int, slug string) string { return fmt.Sprintf("reps_r%d_%s", round, slug) }

// bbbModeField is the settings select for a lift's BBB mode.
func bbbModeField(slug string) string { return "bbb_mode_" + slug }

// bbbValueField is the settings input for a lift's BBB value.
func bbbValueField(slug string) string { return "bbb_value_" + slug }
