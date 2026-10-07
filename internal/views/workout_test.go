package views

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func baseWorkoutVM() WorkoutVM {
	return WorkoutVM{
		ID:          1,
		LiftName:    "Squat",
		CycleLabel:  "Leader 1",
		Unit:        "lb",
		TrainingMax: 300,
		RestSeconds: 90,
		Sets: []SetVM{
			{Idx: 0, Tag: "Warmup", Weight: 135, TargetReps: 5, Done: true, ActualReps: 5},
			{Idx: 1, Tag: "Warmup", Weight: 185, TargetReps: 3, Done: true, ActualReps: 3},
			{Idx: 2, Tag: "Main", Weight: 225, TargetReps: 5},
			{Idx: 3, Tag: "Main", Weight: 255, TargetReps: 5},
		},
	}
}

func render(t *testing.T, w WorkoutVM) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Workout(w).Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render: %v", err)
	}
	return buf.String()
}

// TestRestTimerPosition pins the timer to directly above the set it precedes,
// so it travels down the list with the session instead of sitting fixed at
// the top of the page.
func TestRestTimerPosition(t *testing.T) {
	vm := baseWorkoutVM()
	vm.CurrentIdx = 2
	vm.ShowRest = true

	html := render(t, vm)

	restAt := strings.Index(html, `id="rest"`)
	set1At := strings.Index(html, `id="set-1"`)
	set2At := strings.Index(html, `id="set-2"`)

	if restAt < 0 {
		t.Fatal("expected rest timer to render")
	}
	if !(set1At < restAt && restAt < set2At) {
		t.Errorf("rest timer at %d, want between set-1 (%d) and set-2 (%d)", restAt, set1At, set2At)
	}
}

// TestRestTimerNotRendered covers the two cases where no timer, and no
// leftover style block, should appear at all.
func TestRestTimerNotRendered(t *testing.T) {
	cases := []struct {
		name string
		vm   func(WorkoutVM) WorkoutVM
	}{
		{
			name: "ShowRest false",
			vm: func(vm WorkoutVM) WorkoutVM {
				vm.CurrentIdx = 2
				vm.ShowRest = false
				return vm
			},
		},
		{
			name: "all sets logged",
			vm: func(vm WorkoutVM) WorkoutVM {
				for i := range vm.Sets {
					vm.Sets[i].Done = true
				}
				vm.CurrentIdx = -1
				vm.ShowRest = true
				return vm
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := render(t, tc.vm(baseWorkoutVM()))
			if strings.Contains(html, `id="rest"`) {
				t.Error("expected no rest timer")
			}
			if strings.Contains(html, "<style") {
				t.Error("expected no stray style block")
			}
		})
	}
}

// TestRestStyle pins the countdown to the same N everywhere it appears, since
// the bar and the counter must reach zero together.
func TestRestStyle(t *testing.T) {
	got := restStyle(45)
	for _, want := range []string{
		"animation-duration:45s",
		"steps(45,end)",
		"--rest-left:45",
		"from{--rest-left:45}",
		"to{--rest-left:0}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("restStyle(45) = %q, want substring %q", got, want)
		}
	}
}

func TestRestStyleClampsNonPositive(t *testing.T) {
	for _, seconds := range []int{0, -5} {
		got := restStyle(seconds)
		if !strings.Contains(got, "animation-duration:1s") || !strings.Contains(got, "--rest-left:1") {
			t.Errorf("restStyle(%d) = %q, want clamped to 1", seconds, got)
		}
	}
}

// TestRangedSetAsksForReps covers the 80% x 3-5 set of the deload: it is
// written as a range and logged by picking the reps, starting from a single
// one so a short set can be recorded, with the range itself highlighted.
func TestRangedSetAsksForReps(t *testing.T) {
	vm := baseWorkoutVM()
	vm.Sets = append(vm.Sets, SetVM{Idx: 4, Tag: "Main", Weight: 300, Pct: 1, TargetReps: 3, MaxReps: 5})
	vm.Sets[2].Done, vm.Sets[3].Done = true, true
	vm.CurrentIdx = 4

	html := render(t, vm)

	if !strings.Contains(html, "3–5 reps") {
		t.Error("expected the set to read 3–5 reps")
	}
	if strings.Contains(html, `aria-label="Log set"`) {
		t.Error("a ranged set should ask for reps, not offer a single tick")
	}
	for r := 1; r <= 10; r++ {
		if !strings.Contains(html, `value="`+FmtInt(r)+`">`+FmtInt(r)+`</button>`) {
			t.Errorf("missing a button for %d reps", r)
		}
	}
	for _, r := range []int{3, 4, 5} {
		if !strings.Contains(html, `class="target" type="submit" name="reps" value="`+FmtInt(r)+`"`) {
			t.Errorf("%d reps should be highlighted as in range", r)
		}
	}
	for _, r := range []int{2, 6} {
		if strings.Contains(html, `class="target" type="submit" name="reps" value="`+FmtInt(r)+`"`) {
			t.Errorf("%d reps should not be highlighted", r)
		}
	}
}

// TestTMTestTopSet covers the 100% AMRAP of a TM test: it reads 3+, offers
// every answer from a single rep so a failed test is one tap, highlights the
// 3-5 window that says the training max is right, and shows the number to beat.
func TestTMTestTopSet(t *testing.T) {
	vm := baseWorkoutVM()
	vm.BestE1RM = 330
	vm.Sets = append(vm.Sets, SetVM{Idx: 4, Tag: "Main", Weight: 300, Pct: 1, TargetReps: 3, MaxReps: 5, IsAMRAP: true})
	vm.Sets[2].Done, vm.Sets[3].Done = true, true
	vm.CurrentIdx = 4

	html := render(t, vm)

	if !strings.Contains(html, "3+ reps") {
		t.Error("expected the set to read 3+ reps")
	}
	if strings.Contains(html, `aria-label="Log set"`) {
		t.Error("an AMRAP set should ask for reps, not offer a single tick")
	}
	if !strings.Contains(html, "Best estimated max 330 lb") {
		t.Error("expected the best estimated max to be shown")
	}
	for r := 1; r <= 10; r++ {
		if !strings.Contains(html, `value="`+FmtInt(r)+`">`+FmtInt(r)+`</button>`) &&
			!strings.Contains(html, `class="target" type="submit" name="reps" value="`+FmtInt(r)+`">`) {
			t.Errorf("missing a button for %d reps", r)
		}
	}
	for _, r := range []int{3, 4, 5} {
		if !strings.Contains(html, `class="target" type="submit" name="reps" value="`+FmtInt(r)+`"`) {
			t.Errorf("%d reps should be highlighted as in range", r)
		}
	}
	for _, r := range []int{1, 2, 6} {
		if strings.Contains(html, `class="target" type="submit" name="reps" value="`+FmtInt(r)+`"`) {
			t.Errorf("%d reps should not be highlighted", r)
		}
	}
}

// TestAnchorPRSetStartsAtTarget keeps the picker for a PR set without a window
// starting at its minimum, which covers all but the most exceptional result.
func TestAnchorPRSetStartsAtTarget(t *testing.T) {
	got := repChoices(SetVM{TargetReps: 5, IsAMRAP: true})
	if got[0] != 5 || got[9] != 14 {
		t.Errorf("repChoices = %v, want 5 through 14", got)
	}
	got = repChoices(SetVM{TargetReps: 3, MaxReps: 5, IsAMRAP: true})
	if got[0] != 1 || got[9] != 10 {
		t.Errorf("repChoices with a window = %v, want 1 through 10", got)
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
		if got := RepTarget(tc.reps, tc.max, tc.amrap); got != tc.want {
			t.Errorf("RepTarget(%d, %d, %v) = %q, want %q", tc.reps, tc.max, tc.amrap, got, tc.want)
		}
	}
}
