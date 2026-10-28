package api

// The wire types: what the API sends and accepts, and the only file tygo
// reads to write the app's types.gen.ts. Each one mirrors a training read
// model, minus what only the web needs, plus what a template works out with a
// method, so the app only has to display it.
//
// Lists are never null, and the only fields that can be are pointers. Dates
// are formatted as the server shows them, in its own time zone.

// Info says which API and server build the app is talking to. It needs no
// session, so the app can check it before signing in.
type Info struct {
	// APIVersion only changes for a change that breaks an installed app.
	APIVersion int    `json:"apiVersion"`
	Version    string `json:"version"`
}

// ErrorResponse is the body of every answer that is not a success. The
// message can be shown as it is.
type ErrorResponse struct {
	Error string `json:"error"`
}

// LoginRequest signs in with the server's password.
type LoginRequest struct {
	Password string `json:"password"`
}

// LoginResponse is a session token, sent as Authorization: Bearer <token>.
// Tokens are not stored on the server, so signing out is forgetting it.
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"` // RFC 3339
}

// LiftTM is one lift and its current training max.
type LiftTM struct {
	Slug  string  `json:"slug"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// LiftRef names a lift.
type LiftRef struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// Dashboard is the Today screen.
type Dashboard struct {
	// NeedsSetup is set until training maxes have been entered, and nothing
	// else is filled in until then. The app goes to setup.
	NeedsSetup  bool          `json:"needsSetup"`
	Unit        string        `json:"unit"`
	ProgramName string        `json:"programName"`
	Next        *NextUp       `json:"next"`
	TMs         []LiftTM      `json:"tms"`
	Recent      []HistoryItem `json:"recent"`

	// TestResults are the current block's TM test outcomes: the session just
	// finished, and any lift that failed and is still waiting on a decision.
	TestResults []TMTestResult `json:"testResults"`

	// Review is set at the start of each block after the first. Training
	// waits until it is accepted.
	Review *TMReview `json:"review"`
}

// NextUp is the session the dashboard offers to start, or the one in
// progress.
//
//tygo:emit export type Phase = "leader" | "tm_test" | "anchor" | "deload";
type NextUp struct {
	LiftName    string  `json:"liftName"`
	LiftSlug    string  `json:"liftSlug"`
	CycleLabel  string  `json:"cycleLabel"`
	Phase       string  `json:"phase" tstype:"Phase"`
	TrainingMax float64 `json:"trainingMax"`
	// InProgress is set when a session has been started but not finished,
	// and WorkoutID is then that session; otherwise it is 0.
	InProgress    bool  `json:"inProgress"`
	WorkoutID     int64 `json:"workoutId"`
	BlockNumber   int   `json:"blockNumber"`
	RoundOfBlock  int   `json:"roundOfBlock"`
	RoundsInBlock int   `json:"roundsInBlock"`

	// Blocked is set while a training max decision is waiting; the session
	// cannot start until it is made.
	Blocked bool `json:"blocked"`
	// CanSkip offers to pass on an optional cycle before its first session.
	CanSkip bool `json:"canSkip"`
}

// TMTestResult reports one lift's TM test. A pass is just news; a failure
// offers to lower the training max, and the anchor waits until it is
// settled.
//
//tygo:emit export type Tone = "up" | "hold" | "down";
type TMTestResult struct {
	WorkoutID int64   `json:"workoutId"`
	LiftName  string  `json:"liftName"`
	Unit      string  `json:"unit"`
	Tested    float64 `json:"tested"` // the training max that was tested
	Reps      int     `json:"reps"`   // performed at 100%
	Headline  string  `json:"headline"`
	Detail    string  `json:"detail"`
	Tone      string  `json:"tone" tstype:"Tone"`

	// NeedsDecision is set for a failed test still waiting on the user, who
	// can lower Current to Proposed (or another value), or keep it.
	NeedsDecision bool    `json:"needsDecision"`
	Current       float64 `json:"current"`
	Proposed      float64 `json:"proposed"`
}

// TMReview proposes a training max for every lift at the start of a block.
// Nothing changes until the user accepts it, edited or not.
type TMReview struct {
	Title string        `json:"title"`
	Intro string        `json:"intro"`
	Key   string        `json:"key"` // sent back when accepting
	Unit  string        `json:"unit"`
	Rows  []TMReviewRow `json:"rows"`
}

// TMReviewRow is one lift's evidence and proposed training max.
type TMReviewRow struct {
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Evidence string  `json:"evidence"` // for example "PR sets 8, 5, 3 (min 5, 3, 1)"
	Verdict  string  `json:"verdict"`  // for example "Hit every PR set: +5 lb"
	Tone     string  `json:"tone" tstype:"Tone"`
	Current  float64 `json:"current"`
	Proposed float64 `json:"proposed"`
}

// Workout is the session screen.
type Workout struct {
	ID          int64   `json:"id"`
	LiftName    string  `json:"liftName"`
	CycleLabel  string  `json:"cycleLabel"`
	Phase       string  `json:"phase" tstype:"Phase"`
	Unit        string  `json:"unit"`
	TrainingMax float64 `json:"trainingMax"`
	Sets        []Set   `json:"sets"`
	CurrentIdx  int     `json:"currentIdx"` // -1 once every set is logged
	// RestSeconds is the rest after the last set logged, by its kind. The app
	// starts its rest timer from it when a set is logged.
	RestSeconds int `json:"restSeconds"`
	// Completed marks a session viewed from history rather than performed.
	Completed bool `json:"completed"`
	// BestE1RM is the best estimated max recorded for this lift, shown
	// against the PR set so there is a number to chase. 0 if there is none.
	BestE1RM float64 `json:"bestE1rm"`
}

// Set is one row on the session screen.
//
//tygo:emit export type SetTag = "Warmup" | "Main" | "BBB" | "FSL" | "Supplemental";
type Set struct {
	Idx        int     `json:"idx"`
	Tag        string  `json:"tag" tstype:"SetTag"`
	Weight     float64 `json:"weight"`
	Pct        float64 `json:"pct"` // of the training max, as a fraction
	TargetReps int     `json:"targetReps"`
	MaxReps    int     `json:"maxReps"` // top of a rep range such as 3–5, or 0
	IsAMRAP    bool    `json:"isAmrap"`
	// RepTarget is the target as the book writes it: "5", "3+" or "3–5".
	RepTarget string `json:"repTarget"`
	Plates    string `json:"plates"` // "45, 25, 5" per side
	// PlateLine says how to load the bar: "Bar + 45, 25 per side".
	PlateLine string `json:"plateLine"`
	// Unloadable is how far short the bar falls when it cannot be loaded
	// exactly, or 0.
	Unloadable float64 `json:"unloadable"`
	Done       bool    `json:"done"`
	ActualReps int     `json:"actualReps"`
	// AsksReps is set for a set whose reps are entered rather than ticked
	// off, with RepChoices offering them one tap each. RepChoices is empty
	// otherwise.
	AsksReps   bool        `json:"asksReps"`
	RepChoices []RepChoice `json:"repChoices"`
}

// RepChoice is one answer on the rep picker. Target marks the answers that
// meet the prescription.
type RepChoice struct {
	Reps   int  `json:"reps"`
	Target bool `json:"target"`
}

// History is the history list.
type History struct {
	Items []HistoryItem `json:"items"`
	Lifts []LiftRef     `json:"lifts"`
}

// HistoryItem is one completed session.
type HistoryItem struct {
	WorkoutID  int64  `json:"workoutId"`
	LiftName   string `json:"liftName"`
	LiftSlug   string `json:"liftSlug"`
	CycleLabel string `json:"cycleLabel"`
	Date       string `json:"date"`    // "Jan 2"
	Summary    string `json:"summary"` // "255 x 8" for anchor rounds, otherwise the top set
}

// Lift is the per-lift screen.
type Lift struct {
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Unit          string          `json:"unit"`
	CurrentTM     float64         `json:"currentTm"`
	Chart         Chart           `json:"chart"`
	BestE1RM      float64         `json:"bestE1rm"`
	TMHistory     []TMHistoryItem `json:"tmHistory"`
	PastEstimates []PastEstimate  `json:"pastEstimates"`
	// Today is the latest date a past estimate can have, as YYYY-MM-DD.
	Today string `json:"today"`
}

// Chart is the trend of a lift's estimated max, already laid out in a
// Width x Height space with y growing downwards, as in SVG. Points is empty
// when there is nothing to plot.
type Chart struct {
	Width  float64      `json:"width"`
	Height float64      `json:"height"`
	Path   string       `json:"path"` // an SVG path through the points
	Points []ChartPoint `json:"points"`
	// YMin and YMax are the estimated maxes at the bottom and top of the plot,
	// for the axis labels.
	YMin float64 `json:"yMin"`
	YMax float64 `json:"yMax"`
}

// ChartPoint is one estimate: an AMRAP set logged in the app or, when Past is
// set, one entered from before it.
type ChartPoint struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Label string  `json:"label"`
	Past  bool    `json:"past"`
}

// PastEstimate is an estimate from before the app: a set of Weight x Reps
// worth Estimate as a one rep max.
type PastEstimate struct {
	ID       int64   `json:"id"`
	Date     string  `json:"date"` // "Mar 3, 2026"
	Weight   float64 `json:"weight"`
	Reps     int     `json:"reps"`
	Estimate float64 `json:"estimate"`
}

// TMHistoryItem is one entry from the training max log, newest first.
type TMHistoryItem struct {
	Value float64 `json:"value"`
	Date  string  `json:"date"`
	Note  string  `json:"note"`
}

// Settings is the settings screen.
//
//tygo:emit export type BBBMode = "program" | "percent" | "weight";
type Settings struct {
	Unit              string    `json:"unit"`
	Rounding          float64   `json:"rounding"`
	BarWeight         float64   `json:"barWeight"`
	Plates            []float64 `json:"plates"` // per side, heaviest first
	Warmups           bool      `json:"warmups"`
	RestWarmupSeconds int       `json:"restWarmupSeconds"`
	RestMainSeconds   int       `json:"restMainSeconds"`
	RestSuppSeconds   int       `json:"restSuppSeconds"`
	TMs               []LiftTM  `json:"tms"`
	BBB               []LiftBBB `json:"bbb"`
}

// LiftBBB is one lift's supplemental override.
type LiftBBB struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Mode string `json:"mode" tstype:"BBBMode"`
	// Value is a whole percent when Mode is percent, otherwise a weight.
	Value float64 `json:"value"`
}

// Setup is the first-run screen where training maxes are entered.
type Setup struct {
	Lifts []LiftTM `json:"lifts"` // with the current training max, or 0
	Unit  string   `json:"unit"`
}

// SetupPosition asks a new user which cycle they are on. Starting fresh needs
// nothing sent: the block already starts at the first cycle.
type SetupPosition struct {
	Cycles []SetupCycle `json:"cycles"`
}

// SetupCycle is one choice on the position screen.
type SetupCycle struct {
	Index  int    `json:"index"`
	Label  string `json:"label"`
	Detail string `json:"detail"`
}

// SetupHistory is the checklist of sessions already done in the chosen
// cycle. The first one left unticked becomes the next session.
type SetupHistory struct {
	CycleIndex int    `json:"cycleIndex"`
	CycleLabel string `json:"cycleLabel"`
	Unit       string `json:"unit"`
	// AsksReps is set for anchor and TM test cycles, where the AMRAP set is
	// worth recording; RepsLabel names that set.
	AsksReps  bool         `json:"asksReps"`
	RepsLabel string       `json:"repsLabel"`
	Rounds    []SetupRound `json:"rounds"`
}

// SetupRound groups the sessions of one round.
type SetupRound struct {
	Round int        `json:"round"`
	Days  []SetupDay `json:"days"`
}

// SetupDay is one session on the checklist, with the weight of its top main
// set so it is recognisable.
type SetupDay struct {
	Slug      string  `json:"slug"`
	Name      string  `json:"name"`
	TopWeight float64 `json:"topWeight"`
	PRTarget  int     `json:"prTarget"` // 0 without a PR set
}

// Request bodies.

// LogSetInput logs the reps done on a set.
type LogSetInput struct {
	Reps int `json:"reps"`
}

// PastEstimateInput adds an estimate from before the app.
type PastEstimateInput struct {
	Date   string  `json:"date"` // YYYY-MM-DD, not after the lift's today
	Weight float64 `json:"weight"`
	Reps   int     `json:"reps"`
}

// SettingsInput changes the settings. A number that is not positive keeps
// what is there now; plates need at least one positive.
type SettingsInput struct {
	Rounding          float64   `json:"rounding"`
	BarWeight         float64   `json:"barWeight"`
	Plates            []float64 `json:"plates"`
	Warmups           bool      `json:"warmups"`
	RestWarmupSeconds int       `json:"restWarmupSeconds"`
	RestMainSeconds   int       `json:"restMainSeconds"`
	RestSuppSeconds   int       `json:"restSuppSeconds"`
	// BBB is each lift's override by slug. A lift left out, or a value that
	// is not positive, goes back to the program's default.
	BBB map[string]BBBInput `json:"bbb"`
}

// BBBInput is one lift's supplemental override.
type BBBInput struct {
	Mode  string  `json:"mode" tstype:"BBBMode"`
	Value float64 `json:"value"`
}

// TrainingMaxInput changes one lift's training max from settings.
type TrainingMaxInput struct {
	Lift  string  `json:"lift"` // slug
	Value float64 `json:"value"`
	Note  string  `json:"note"`
}

// TMReviewInput accepts the block review with the training maxes the user
// settled on, by lift slug. Every lift needs a positive value.
type TMReviewInput struct {
	Key string             `json:"key"`
	TMs map[string]float64 `json:"tms"`
}

// TMTestDecision settles a failed TM test: keep the training max, or lower
// it to TM.
//
//tygo:emit export type TMTestAction = "keep" | "lower";
type TMTestDecision struct {
	Action string  `json:"action" tstype:"TMTestAction"`
	TM     float64 `json:"tm"`
}

// SetupInput records the first training maxes, by lift slug. Every lift
// needs a positive value.
type SetupInput struct {
	TMs map[string]float64 `json:"tms"`
}

// SetupHistoryInput joins the block at a cycle, with the sessions of it
// already done.
type SetupHistoryInput struct {
	Done []PastSession `json:"done"`
}

// PastSession is a session already done. PRReps is the reps on its AMRAP
// set, or 0 if not given or not asked for.
type PastSession struct {
	Round  int    `json:"round"`
	Slug   string `json:"slug"`
	PRReps int    `json:"prReps"`
}
