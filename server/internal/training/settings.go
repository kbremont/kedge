package training

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kbremont/531-tracker/server/internal/db"
)

// Settings returns the settings screen.
func (s *Service) Settings(ctx context.Context) (*Settings, error) {
	raw, settings, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	tms, err := s.q.ListCurrentTrainingMaxes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list training maxes: %w", err)
	}
	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list lifts: %w", err)
	}

	out := &Settings{
		Unit:              raw.Unit,
		Rounding:          raw.Rounding,
		BarWeight:         raw.BarWeight,
		Plates:            settings.Plates,
		Warmups:           settings.Warmups,
		RestWarmupSeconds: int(raw.RestWarmupSeconds),
		RestMainSeconds:   int(raw.RestMainSeconds),
		RestSuppSeconds:   int(raw.RestSuppSeconds),
	}
	for _, tm := range tms {
		out.TMs = append(out.TMs, LiftTM{Slug: tm.Slug, Name: tm.Name, Value: tm.Value})
	}
	for _, l := range lifts {
		bbb := LiftBBB{Slug: l.Slug, Name: l.Name, Mode: l.BbbMode, Value: l.BbbValue}
		if bbb.Mode == "percent" {
			bbb.Value *= 100
		}
		out.BBB = append(out.BBB, bbb)
	}
	return out, nil
}

// SettingsInput is a change to the settings. A number that is not positive
// keeps what is there now.
type SettingsInput struct {
	Rounding          float64
	BarWeight         float64
	Plates            []float64 // per side, in any order; at least one must be positive
	Warmups           bool
	RestWarmupSeconds int
	RestMainSeconds   int
	RestSuppSeconds   int
	// BBB is each lift's supplemental override by slug. A lift left out, an
	// unknown mode, or a value that is not positive goes back to the
	// program's default.
	BBB map[string]LiftBBB
}

// SaveSettings applies a change to the settings. Plates with none positive
// are an InputError, and nothing is written.
func (s *Service) SaveSettings(ctx context.Context, in SettingsInput) error {
	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		return err
	}

	plates := make([]float64, 0, len(in.Plates))
	for _, p := range in.Plates {
		if p > 0 {
			plates = append(plates, p)
		}
	}
	if len(plates) == 0 {
		return inputError("List at least one plate.")
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(plates)))
	encoded, err := json.Marshal(plates)
	if err != nil {
		return fmt.Errorf("encode plates: %w", err)
	}

	warmups := int64(0)
	if in.Warmups {
		warmups = 1
	}

	params := db.UpdateSettingsParams{
		Unit:              raw.Unit, // pounds only for now; the column exists for later
		Rounding:          positiveOr(in.Rounding, raw.Rounding),
		BarWeight:         positiveOr(in.BarWeight, raw.BarWeight),
		Plates:            string(encoded),
		Warmups:           warmups,
		RestWarmupSeconds: int64(positiveOr(float64(in.RestWarmupSeconds), float64(raw.RestWarmupSeconds))),
		RestMainSeconds:   int64(positiveOr(float64(in.RestMainSeconds), float64(raw.RestMainSeconds))),
		RestSuppSeconds:   int64(positiveOr(float64(in.RestSuppSeconds), float64(raw.RestSuppSeconds))),
	}
	if err := s.q.UpdateSettings(ctx, params); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}

	lifts, err := s.q.ListLifts(ctx)
	if err != nil {
		return fmt.Errorf("list lifts: %w", err)
	}
	return s.tx(ctx, func(q *db.Queries) error {
		for _, l := range lifts {
			bbb := in.BBB[l.Slug]
			mode, value := bbb.Mode, positiveOr(bbb.Value, 0)
			switch mode {
			case "percent":
				value /= 100
			case "weight":
				// value is already an absolute weight
			default:
				mode = "program"
			}
			if value <= 0 {
				mode = "program"
			}
			if err := q.SetLiftBBB(ctx, db.SetLiftBBBParams{
				BbbMode:  mode,
				BbbValue: value,
				Slug:     l.Slug,
			}); err != nil {
				return fmt.Errorf("set BBB for %s: %w", l.Slug, err)
			}
		}
		return nil
	})
}

// SetTrainingMax appends to the training max log. It never updates in place,
// so past sessions keep the weights they were performed at.
func (s *Service) SetTrainingMax(ctx context.Context, slug string, value float64, note string) error {
	if value <= 0 {
		return inputError("Enter a positive training max.")
	}
	if _, err := s.q.SetTrainingMax(ctx, db.SetTrainingMaxParams{
		LiftSlug: slug,
		Value:    value,
		Note:     note,
	}); err != nil {
		return fmt.Errorf("set training max for %s: %w", slug, err)
	}
	return nil
}

func positiveOr(v, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	return v
}
