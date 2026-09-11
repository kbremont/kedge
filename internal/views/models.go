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
}

// DashboardVM is the home screen.
type DashboardVM struct {
	Unit        string
	ProgramName string
	Next        *NextUp
	TMs         []LiftTM
	Recent      []HistoryItem

	// CycleDone is set when the last session closed out a cycle, prompting
	// the user to consider a training max bump. Nothing is applied for them.
	CycleDone      bool
	CycleDoneLabel string
	// CycleDoneKey identifies this rollover so dismissing it sticks.
	CycleDoneKey string
	Suggestions  []TMSuggestion
}

// TMSuggestion is an offer, never an action.
type TMSuggestion struct {
	Slug    string
	Name    string
	Current float64
	Next    float64
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

// ChartPoint is one AMRAP set plotted over time.
type ChartPoint struct {
	X, Y  float64 // already in SVG user space
	Label string
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
	// AsksReps is set for anchor cycles, where the PR set is the only number
	// worth recording; leader rounds are 5's PRO and fully determined.
	AsksReps bool
	Rounds   []SetupRoundVM
	Error    string
}
