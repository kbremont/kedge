package program

// Training max progression. The app recommends; it never applies. Everything
// here turns logged reps into a proposed number for the user to accept or edit.

// TMVerdict is what a 7th week test says about a training max.
type TMVerdict string

const (
	VerdictTooHigh TMVerdict = "too_high" // under 3 reps at 100%
	VerdictRight   TMVerdict = "right"    // 3 to 5 reps
	VerdictBump    TMVerdict = "bump"     // 6 or more
)

// The TM test asks for 3-5 reps at 100% of the training max.
const (
	tmTestMinReps = 3
	tmTestMaxReps = 5
)

// resetFactor is the standard 5/3/1 reset: a training max that has stopped
// moving comes down by 10%.
const resetFactor = 0.9

// TMTestVerdict reads the reps performed at 100% of the training max.
func TMTestVerdict(reps int) TMVerdict {
	switch {
	case reps < tmTestMinReps:
		return VerdictTooHigh
	case reps <= tmTestMaxReps:
		return VerdictRight
	default:
		return VerdictBump
	}
}

// TMTestProposal is the training max the test proposes straight away. It only
// ever lowers: a test that says "bump" is held over to the end of the block,
// so the anchor never runs on a training max raised mid-block.
func TMTestProposal(current float64, reps int, s Settings) float64 {
	if TMTestVerdict(reps) == VerdictTooHigh {
		return ResetTM(current, s)
	}
	return current
}

// ResetTM takes 10% off a training max, rounded to a loadable weight.
func ResetTM(current float64, s Settings) float64 {
	return RoundWeight(current*resetFactor, s)
}

// PRSet is one rep-max set from the anchor: the minimum the book prescribes
// and the reps actually performed.
type PRSet struct {
	Target int
	Actual int
}

// BumpReason explains an end-of-block recommendation.
type BumpReason string

const (
	ReasonMissedPR   BumpReason = "missed_pr"   // an anchor PR set fell short: reset
	ReasonHitPR      BumpReason = "hit_pr"      // every PR set made its minimum: one step
	ReasonHitPRAndTM BumpReason = "hit_pr_test" // and the TM test went 6+: two steps
	ReasonNoEvidence BumpReason = "no_evidence" // no anchor results to go on: hold
)

// RecommendBump proposes the training max for the next block. The anchor
// decides the direction: missing the minimum on any PR set resets the training
// max, and making every one earns a step. A TM test that went 6+ adds a second
// step on top. With no anchor results, nothing is proposed.
func RecommendBump(cfg Config, current float64, isUpper bool, anchor []PRSet, testReps int, hasTest bool, s Settings) (float64, BumpReason) {
	if len(anchor) == 0 {
		return current, ReasonNoEvidence
	}
	for _, pr := range anchor {
		if pr.Actual < pr.Target {
			return ResetTM(current, s), ReasonMissedPR
		}
	}
	step := TMIncrementFor(cfg, isUpper)
	if hasTest && TMTestVerdict(testReps) == VerdictBump {
		return current + 2*step, ReasonHitPRAndTM
	}
	return current + step, ReasonHitPR
}
