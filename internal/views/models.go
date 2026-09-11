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
	InProgress   bool
	WorkoutID    int64
	BlockNumber  int
	WeekOfBlock  int
	WeeksInBlock int
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
	Summary    string // "255 x 8" for anchor weeks, otherwise the top set
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
	Unit        string
	Rounding    float64
	BarWeight   float64
	Plates      string
	Warmups     bool
	RestSeconds int
	TMs         []LiftTM
	Saved       bool
}

// SetupVM is the first-run screen where training maxes are entered.
type SetupVM struct {
	Lifts []LiftTM
	Unit  string
	Error string
}
