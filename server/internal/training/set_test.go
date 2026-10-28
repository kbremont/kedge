package training

import "testing"

// TestAnchorPRSetStartsAtTarget keeps the picker for a PR set without a window
// starting at its minimum, which covers all but the most exceptional result.
func TestAnchorPRSetStartsAtTarget(t *testing.T) {
	got := Set{TargetReps: 5, IsAMRAP: true}.RepChoices()
	if got[0] != 5 || got[9] != 14 {
		t.Errorf("RepChoices = %v, want 5 through 14", got)
	}
	got = Set{TargetReps: 3, MaxReps: 5, IsAMRAP: true}.RepChoices()
	if got[0] != 1 || got[9] != 10 {
		t.Errorf("RepChoices with a window = %v, want 1 through 10", got)
	}
}

func TestRepTarget(t *testing.T) {
	cases := []struct {
		reps, max int
		amrap     bool
		want      string
	}{
		{5, 0, false, "5"},
		{3, 0, true, "3+"},
		{3, 5, false, "3–5"},
	}
	for _, tc := range cases {
		if got := (Set{TargetReps: tc.reps, MaxReps: tc.max, IsAMRAP: tc.amrap}).RepTarget(); got != tc.want {
			t.Errorf("RepTarget(%d, %d, %v) = %q, want %q", tc.reps, tc.max, tc.amrap, got, tc.want)
		}
	}
}
