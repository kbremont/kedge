package views

import "fmt"

// Form field names, shared by the templates that render the forms and the
// handlers that read them back.

// TMField is the input for a lift's training max, on setup and the block
// review.
func TMField(slug string) string { return "tm_" + slug }

// DoneField is the onboarding checkbox for a session already done.
func DoneField(round int, slug string) string { return fmt.Sprintf("done_r%d_%s", round, slug) }

// RepsField is the onboarding input for that session's PR set reps.
func RepsField(round int, slug string) string { return fmt.Sprintf("reps_r%d_%s", round, slug) }

// BBBModeField is the settings select for a lift's BBB mode.
func BBBModeField(slug string) string { return "bbb_mode_" + slug }

// BBBValueField is the settings input for a lift's BBB value.
func BBBValueField(slug string) string { return "bbb_value_" + slug }
