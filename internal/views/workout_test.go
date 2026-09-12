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
