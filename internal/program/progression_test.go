package program

import "testing"

// TestTMTestVerdict pins the boundaries of the 3-5 rep window at 100%.
func TestTMTestVerdict(t *testing.T) {
	cases := []struct {
		reps int
		want TMVerdict
	}{
		{0, VerdictTooHigh},
		{2, VerdictTooHigh},
		{3, VerdictRight},
		{5, VerdictRight},
		{6, VerdictBump},
		{12, VerdictBump},
	}
	for _, tc := range cases {
		if got := TMTestVerdict(tc.reps); got != tc.want {
			t.Errorf("TMTestVerdict(%d) = %q, want %q", tc.reps, got, tc.want)
		}
	}
}

// TestTMTestProposal checks that the test only ever proposes a reset, never a
// raise: a 6+ result waits for the end of the block.
func TestTMTestProposal(t *testing.T) {
	s := DefaultSettings()

	cases := []struct {
		name    string
		current float64
		reps    int
		want    float64
	}{
		{"too high resets 10%", 300, 2, 270},
		{"reset rounds to the plate increment", 135, 1, 120}, // 121.5
		{"right holds", 300, 4, 300},
		{"bump still holds mid-block", 300, 8, 300},
		{"reset never drops below the bar", 45, 0, 45},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TMTestProposal(tc.current, tc.reps, s); !nearly(got, tc.want) {
				t.Errorf("TMTestProposal(%v, %d) = %v, want %v", tc.current, tc.reps, got, tc.want)
			}
		})
	}
}

// TestRecommendBump pins the end-of-block rule: the anchor decides the
// direction and a 6+ TM test adds a second step.
func TestRecommendBump(t *testing.T) {
	cfg := LeaderAnchorBBB()
	s := DefaultSettings()

	hitAll := []PRSet{{5, 8}, {3, 5}, {1, 3}}
	missedOne := []PRSet{{5, 7}, {3, 2}, {1, 2}}
	exactlyMinimum := []PRSet{{5, 5}, {3, 3}, {1, 1}}

	cases := []struct {
		name       string
		current    float64
		isUpper    bool
		anchor     []PRSet
		testReps   int
		hasTest    bool
		want       float64
		wantReason BumpReason
	}{
		{"upper hit all", 200, true, hitAll, 4, true, 205, ReasonHitPR},
		{"lower hit all", 400, false, hitAll, 4, true, 410, ReasonHitPR},
		{"making the minimum counts as a hit", 200, true, exactlyMinimum, 3, true, 205, ReasonHitPR},
		{"upper hit all and test 6+", 200, true, hitAll, 6, true, 210, ReasonHitPRAndTM},
		{"lower hit all and test 6+", 400, false, hitAll, 9, true, 420, ReasonHitPRAndTM},
		{"no test means one step", 200, true, hitAll, 0, false, 205, ReasonHitPR},
		{"missed one resets 10%", 300, false, missedOne, 9, true, 270, ReasonMissedPR},
		{"no anchor results holds", 300, false, nil, 9, true, 300, ReasonNoEvidence},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := RecommendBump(cfg, tc.current, tc.isUpper, tc.anchor, tc.testReps, tc.hasTest, s)
			if !nearly(got, tc.want) {
				t.Errorf("proposed %v, want %v", got, tc.want)
			}
			if reason != tc.wantReason {
				t.Errorf("reason %q, want %q", reason, tc.wantReason)
			}
		})
	}
}
