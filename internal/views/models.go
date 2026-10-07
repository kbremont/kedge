// Package views renders the HTML. It depends only on the view models declared
// here, so templates never reach into the database or the program engine.
package views

// Nav marks which top-level section is active.
type Nav string

const (
	NavHome     Nav = "home"
	NavHistory  Nav = "history"
	NavSettings Nav = "settings"
)

// SetVM is one row on the session screen.
type SetVM struct {
	Idx        int
	Tag        string // Warmup, Main, BBB, FSL
	Weight     float64
	Pct        float64
	TargetReps int
	MaxReps    int // top of a rep range such as 3-5, or 0
	IsAMRAP    bool
	Plates     string  // "45, 25, 5" per side
	Unloadable float64 // non-zero when the bar cannot be loaded exactly
	Done       bool
	ActualReps int
}

// WorkoutVM is the session screen.
type WorkoutVM struct {
	ID          int64
	LiftName    string
	CycleLabel  string
	Phase       string
	Unit        string
	TrainingMax float64
	Sets        []SetVM
	CurrentIdx  int // -1 once every set is logged
	RestSeconds int
	ShowRest    bool
	// Completed marks a session being viewed from history rather than performed.
	Completed bool

	// BestE1RM is the best estimated max recorded for this lift, shown
	// against the PR set so there is a number to chase.
	BestE1RM float64
}

// LiftTM is one lift and its current training max.
type LiftTM struct {
	Slug  string
	Name  string
	Value float64
}

// NextUp describes the session the dashboard offers to start.
type NextUp struct {
	LiftName    string
	LiftSlug    string
	CycleLabel  string
	Phase       string
	TrainingMax float64
	Day         int
	// InProgress is set when a session has been started but not finished.
	InProgress    bool
	WorkoutID     int64
	BlockNumber   int
	RoundOfBlock  int
	RoundsInBlock int

	// Blocked is set while a training max decision is waiting; the next
	// session cannot start until it is made.
	Blocked bool
	// CanSkip offers to pass on an optional cycle before its first session.
	CanSkip bool
}

// DashboardVM is the home screen.
type DashboardVM struct {
	Unit        string
	ProgramName string
	Next        *NextUp
	TMs         []LiftTM
	Recent      []HistoryItem

	// TestResults are the current block's TM test outcomes: the session just
	// finished, and any lift that failed and is still waiting on a decision.
	TestResults []TMTestResultVM

	// Review is set at the start of each new block, the one place training
	// maxes go up. Training waits until it is accepted.
	Review *TMReviewVM
}

// TMTestResultVM reports one lift's TM test. A pass is just news; a failure
// offers to lower the training max, and the anchor waits until it is settled.
type TMTestResultVM struct {
	WorkoutID int64
	LiftName  string
	Unit      string
	Tested    float64 // the training max that was tested
	Reps      int     // performed at 100%
	Headline  string
	Detail    string
	Tone      string // down, hold, or up

	// NeedsDecision is set for a failed test still waiting on the user.
	NeedsDecision bool
	Current       float64
	Proposed      float64
}

// TMReviewVM proposes a training max for every lift at the start of a block.
// It is an offer, never an action: nothing changes until the user accepts it,
// edited or not.
type TMReviewVM struct {
	Title string
	Intro string
	Key   string // identifies this review so accepting it sticks
	Unit  string
	Rows  []TMReviewRow
}

// TMReviewRow is one lift's evidence and proposed training max.
type TMReviewRow struct {
	Slug     string
	Name     string
	Evidence string // what was lifted, for example "PR sets 8, 5, 3 (min 5, 3, 1)"
	Verdict  string // for example "Hit every PR set: +5 lb"
	Tone     string // down, hold, or up, for styling the verdict
	Current  float64
	Proposed float64
	Field    string // form field name for the proposed value
}

// HistoryItem is one completed session.
type HistoryItem struct {
	WorkoutID  int64
	LiftName   string
	LiftSlug   string
	CycleLabel string
	Date       string
	Summary    string // "255 x 8" for anchor rounds, otherwise the top set
}

// HistoryVM is the history list.
type HistoryVM struct {
	Items []HistoryItem
	Lifts []LiftTM
}

// ChartPoint is one estimate plotted over time: an AMRAP set logged in the app
// or, when Past is set, one entered from before it.
type ChartPoint struct {
	X, Y  float64 // already in SVG user space
	Label string
	Past  bool
}

// LiftVM is the per-lift detail screen.
type LiftVM struct {
	Name      string
	Slug      string
	Unit      string
	CurrentTM float64
	Points    []ChartPoint
	Path      string // SVG polyline path for the estimated-max trend
	YMin      float64
	YMax      float64
	BestE1RM  float64
	TMHistory []TMHistoryItem

	PastEstimates []PastEstimateVM
	Today         string // YYYY-MM-DD, the latest date a past estimate can have
	Error         bool   // the last past estimate submitted was rejected
}

// PastEstimateVM is an estimate from before the app, as listed under the
// chart: a set of Weight x Reps worth Estimate as a one rep max.
type PastEstimateVM struct {
	ID       int64
	Date     string
	Weight   float64
	Reps     int
	Estimate float64
}

// TMHistoryItem is one entry from the append-only training max log.
type TMHistoryItem struct {
	Value float64
	Date  string
	Note  string
}

// SettingsVM is the settings screen.
type SettingsVM struct {
	Unit              string
	Rounding          float64
	BarWeight         float64
	Plates            string
	Warmups           bool
	RestWarmupSeconds int
	RestMainSeconds   int
	RestSuppSeconds   int
	TMs               []LiftTM
	BBB               []LiftBBB
	Saved             bool
}

// LiftBBB is one lift's supplemental override on the settings screen.
type LiftBBB struct {
	Slug, Name string
	Mode       string  // program, percent, weight
	Value      float64 // whole percent when Mode is percent, otherwise a weight
	ModeField  string  // form field name for the mode select
	ValueField string  // form field name for the value input
}

// SetupVM is the first-run screen where training maxes are entered.
type SetupVM struct {
	Lifts []LiftTM
	Unit  string
	Error string
}

// SetupCycleOption is one choice on the "where are you?" screen.
type SetupCycleOption struct {
	Index  int
	Label  string
	Detail string
}

// SetupPositionVM asks a new user which cycle they are currently on, so the
// app can start the block there rather than at the beginning.
type SetupPositionVM struct {
	Cycles []SetupCycleOption
}

// SetupDayVM is one session in the onboarding checklist. The weight of the top
// main set is shown so the session is recognisable at a glance.
type SetupDayVM struct {
	Slug      string
	Name      string
	TopWeight float64
	PRTarget  int
	Field     string // form field name for the "done" checkbox
	RepsField string // form field name for the PR set reps
}

// SetupRoundVM groups the sessions of one round.
type SetupRoundVM struct {
	Round int
	Days  []SetupDayVM
}

// SetupHistoryVM is the checklist of sessions already completed in the chosen
// cycle. Whichever box is left unticked first becomes the next session, so this
// one screen answers both "what have you done" and "where are you".
type SetupHistoryVM struct {
	CycleIndex int
	CycleLabel string
	Unit       string
	// AsksReps is set for anchor and TM test cycles, where the AMRAP set is the
	// only number worth recording; leader rounds are 5's PRO and fully
	// determined.
	AsksReps bool
	// RepsLabel names that set: the PR set on the anchor, the 100% set on the
	// TM test.
	RepsLabel string
	Rounds    []SetupRoundVM
	Error     string
}
