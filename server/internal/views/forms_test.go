package views

import "testing"

func TestFormFieldNames(t *testing.T) {
	if got, want := DoneField(2, "press"), "done_r2_press"; got != want {
		t.Errorf("doneField = %q, want %q", got, want)
	}
	if got, want := RepsField(3, "deadlift"), "reps_r3_deadlift"; got != want {
		t.Errorf("repsField = %q, want %q", got, want)
	}
	if got, want := BBBModeField("press"), "bbb_mode_press"; got != want {
		t.Errorf("bbbModeField = %q, want %q", got, want)
	}
	if got, want := BBBValueField("deadlift"), "bbb_value_deadlift"; got != want {
		t.Errorf("bbbValueField = %q, want %q", got, want)
	}
}
