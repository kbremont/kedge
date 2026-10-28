package api

import (
	"math"

	"github.com/kbremont/531-tracker/server/internal/training"
)

// Converters from training's read models to the wire types generated from the
// spec. Lists are always made, never left nil, so the JSON has [] rather than
// null, as the spec requires.

func liftTMs(in []training.LiftTM) []LiftTM {
	out := make([]LiftTM, 0, len(in))
	for _, l := range in {
		out = append(out, LiftTM{Slug: l.Slug, Name: l.Name, Value: l.Value})
	}
	return out
}

func toDashboard(d *training.Dashboard) Dashboard {
	out := Dashboard{
		NeedsSetup:  d.NeedsSetup,
		Unit:        d.Unit,
		ProgramName: d.ProgramName,
		Tms:         liftTMs(d.TMs),
		Recent:      historyItems(d.Recent),
		TestResults: make([]TMTestResult, 0, len(d.TestResults)),
	}
	if n := d.Next; n != nil {
		out.Next = &NextUp{
			LiftName:      n.LiftName,
			LiftSlug:      n.LiftSlug,
			CycleLabel:    n.CycleLabel,
			Phase:         Phase(n.Phase),
			TrainingMax:   n.TrainingMax,
			InProgress:    n.InProgress,
			WorkoutID:     n.WorkoutID,
			BlockNumber:   n.BlockNumber,
			RoundOfBlock:  n.RoundOfBlock,
			RoundsInBlock: n.RoundsInBlock,
			Blocked:       n.Blocked,
			CanSkip:       n.CanSkip,
		}
	}
	for _, r := range d.TestResults {
		out.TestResults = append(out.TestResults, TMTestResult{
			WorkoutID:     r.WorkoutID,
			LiftName:      r.LiftName,
			Unit:          r.Unit,
			Tested:        r.Tested,
			Reps:          r.Reps,
			Headline:      r.Headline,
			Detail:        r.Detail,
			Tone:          Tone(r.Tone),
			NeedsDecision: r.NeedsDecision,
			Current:       r.Current,
			Proposed:      r.Proposed,
		})
	}
	if r := d.Review; r != nil {
		out.Review = &TMReview{
			Title: r.Title,
			Intro: r.Intro,
			Key:   r.Key,
			Unit:  r.Unit,
			Rows:  make([]TMReviewRow, 0, len(r.Rows)),
		}
		for _, row := range r.Rows {
			out.Review.Rows = append(out.Review.Rows, TMReviewRow{
				Slug:     row.Slug,
				Name:     row.Name,
				Evidence: row.Evidence,
				Verdict:  row.Verdict,
				Tone:     Tone(row.Tone),
				Current:  row.Current,
				Proposed: row.Proposed,
			})
		}
	}
	return out
}

func toWorkout(w *training.Workout) Workout {
	out := Workout{
		ID:          w.ID,
		LiftName:    w.LiftName,
		CycleLabel:  w.CycleLabel,
		Phase:       Phase(w.Phase),
		Unit:        w.Unit,
		TrainingMax: w.TrainingMax,
		Sets:        make([]Set, 0, len(w.Sets)),
		CurrentIdx:  w.CurrentIdx,
		RestSeconds: w.RestSeconds,
		Completed:   w.Completed,
		BestE1RM:    w.BestE1RM,
	}
	for _, s := range w.Sets {
		out.Sets = append(out.Sets, toSet(s))
	}
	return out
}

func toSet(s training.Set) Set {
	out := Set{
		Idx:        s.Idx,
		Tag:        SetTag(s.Tag),
		Weight:     s.Weight,
		Pct:        s.Pct,
		TargetReps: s.TargetReps,
		MaxReps:    s.MaxReps,
		IsAMRAP:    s.IsAMRAP,
		RepTarget:  s.RepTarget(),
		Plates:     s.Plates,
		PlateLine:  s.PlateLine(),
		Unloadable: s.Unloadable,
		Done:       s.Done,
		ActualReps: s.ActualReps,
		AsksReps:   s.AsksReps(),
		RepChoices: []RepChoice{},
	}
	if out.AsksReps {
		for _, r := range s.RepChoices() {
			out.RepChoices = append(out.RepChoices, RepChoice{Reps: r, Target: s.IsTargetRep(r)})
		}
	}
	return out
}

func historyItems(in []training.HistoryItem) []HistoryItem {
	out := make([]HistoryItem, 0, len(in))
	for _, h := range in {
		out = append(out, HistoryItem{
			WorkoutID:  h.WorkoutID,
			LiftName:   h.LiftName,
			LiftSlug:   h.LiftSlug,
			CycleLabel: h.CycleLabel,
			Date:       h.Date,
			Summary:    h.Summary,
		})
	}
	return out
}

func toHistory(h *training.History) History {
	out := History{
		Items: historyItems(h.Items),
		Lifts: make([]LiftRef, 0, len(h.Lifts)),
	}
	for _, l := range h.Lifts {
		out.Lifts = append(out.Lifts, LiftRef{Slug: l.Slug, Name: l.Name})
	}
	return out
}

func toLift(l *training.Lift) Lift {
	out := Lift{
		Name:      l.Name,
		Slug:      l.Slug,
		Unit:      l.Unit,
		CurrentTM: l.CurrentTM,
		Chart: Chart{
			Width:  training.ChartWidth,
			Height: training.ChartHeight,
			Path:   l.Path,
			Points: make([]ChartPoint, 0, len(l.Points)),
			YMin:   l.YMin,
			YMax:   l.YMax,
		},
		BestE1RM:      l.BestE1RM,
		TMHistory:     make([]TMHistoryItem, 0, len(l.TMHistory)),
		PastEstimates: make([]PastEstimate, 0, len(l.PastEstimates)),
		Today:         l.Today,
	}
	for _, p := range l.Points {
		// The path is drawn to a tenth of a unit, and so are the points on it.
		out.Chart.Points = append(out.Chart.Points, ChartPoint{X: tenth(p.X), Y: tenth(p.Y), Label: p.Label, Past: p.Past})
	}
	for _, h := range l.TMHistory {
		out.TMHistory = append(out.TMHistory, TMHistoryItem{Value: h.Value, Date: h.Date, Note: h.Note})
	}
	for _, e := range l.PastEstimates {
		out.PastEstimates = append(out.PastEstimates, PastEstimate{
			ID:       e.ID,
			Date:     e.Date,
			Weight:   e.Weight,
			Reps:     e.Reps,
			Estimate: e.Estimate,
		})
	}
	return out
}

func tenth(v float64) float64 { return math.Round(v*10) / 10 }

func toSettings(s *training.Settings) Settings {
	out := Settings{
		Unit:              s.Unit,
		Rounding:          s.Rounding,
		BarWeight:         s.BarWeight,
		Plates:            append(make([]float64, 0, len(s.Plates)), s.Plates...),
		Warmups:           s.Warmups,
		RestWarmupSeconds: s.RestWarmupSeconds,
		RestMainSeconds:   s.RestMainSeconds,
		RestSuppSeconds:   s.RestSuppSeconds,
		Tms:               liftTMs(s.TMs),
		BBB:               make([]LiftBBB, 0, len(s.BBB)),
	}
	for _, b := range s.BBB {
		// A percent is stored as a fraction, and 0.55 * 100 is not quite 55.
		value := math.Round(b.Value*1e6) / 1e6
		out.BBB = append(out.BBB, LiftBBB{Slug: b.Slug, Name: b.Name, Mode: BBBMode(b.Mode), Value: value})
	}
	return out
}

func toSetup(s *training.Setup) Setup {
	return Setup{Lifts: liftTMs(s.Lifts), Unit: s.Unit}
}

func toSetupPosition(p *training.SetupPosition) SetupPosition {
	out := SetupPosition{Cycles: make([]SetupCycle, 0, len(p.Cycles))}
	for _, c := range p.Cycles {
		out.Cycles = append(out.Cycles, SetupCycle{Index: c.Index, Label: c.Label, Detail: c.Detail})
	}
	return out
}

func toSetupHistory(h *training.SetupHistory) SetupHistory {
	out := SetupHistory{
		CycleIndex: h.CycleIndex,
		CycleLabel: h.CycleLabel,
		Unit:       h.Unit,
		AsksReps:   h.AsksReps,
		RepsLabel:  h.RepsLabel,
		Rounds:     make([]SetupRound, 0, len(h.Rounds)),
	}
	for _, r := range h.Rounds {
		round := SetupRound{Round: r.Round, Days: make([]SetupDay, 0, len(r.Days))}
		for _, d := range r.Days {
			round.Days = append(round.Days, SetupDay{Slug: d.Slug, Name: d.Name, TopWeight: d.TopWeight, PRTarget: d.PRTarget})
		}
		out.Rounds = append(out.Rounds, round)
	}
	return out
}
