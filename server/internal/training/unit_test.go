package training

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"math"
	"testing"

	"github.com/kbremont/kedge/server/internal/db"
)

func newUnitService(t *testing.T) *Service {
	t.Helper()
	conn, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if err := db.Migrate(conn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(conn, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

var kgTMs = map[string]float64{"press": 50, "deadlift": 170, "bench": 87.5, "squat": 142.5}
var lbTMs = map[string]float64{"press": 115, "deadlift": 375, "bench": 195, "squat": 315}

// TestSetupInKilograms brings the kilogram bar, rounding, plates and training
// max steps with the unit, puts a BBB set as a fixed weight back to the
// program, and replaces the training maxes entered in pounds before.
func TestSetupInKilograms(t *testing.T) {
	ctx := context.Background()
	svc := newUnitService(t)

	if err := svc.SaveSetup(ctx, lbTMs, ""); err != nil {
		t.Fatalf("SaveSetup in pounds: %v", err)
	}
	if err := svc.SaveSettings(ctx, SettingsInput{
		Plates: []float64{45, 25, 10, 5, 2.5},
		BBB:    map[string]LiftBBB{"deadlift": {Mode: "weight", Value: 185}, "squat": {Mode: "percent", Value: 55}},
	}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	setup, err := svc.Setup(ctx)
	if err != nil || !setup.CanChangeUnit || setup.Unit != "lb" {
		t.Fatalf("Setup = %+v, %v; want lb that can change", setup, err)
	}

	if err := svc.SaveSetup(ctx, kgTMs, "kg"); err != nil {
		t.Fatalf("SaveSetup in kilograms: %v", err)
	}

	got, err := svc.Settings(ctx)
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got.Unit != "kg" || got.BarWeight != 20 || got.Rounding != 2.5 {
		t.Errorf("settings = %s, bar %v, rounding %v; want kg, 20, 2.5", got.Unit, got.BarWeight, got.Rounding)
	}
	wantPlates := []float64{25, 20, 15, 10, 5, 2.5, 1.25}
	if len(got.Plates) != len(wantPlates) {
		t.Fatalf("plates = %v, want %v", got.Plates, wantPlates)
	}
	for i := range wantPlates {
		if got.Plates[i] != wantPlates[i] {
			t.Fatalf("plates = %v, want %v", got.Plates, wantPlates)
		}
	}
	for _, b := range got.BBB {
		switch b.Slug {
		case "deadlift":
			if b.Mode != "program" {
				t.Errorf("deadlift BBB = %s, want program: 185 was in pounds", b.Mode)
			}
		case "squat":
			if b.Mode != "percent" || math.Abs(b.Value-55) > 1e-9 {
				t.Errorf("squat BBB = %s %v, want percent 55 kept", b.Mode, b.Value)
			}
		}
	}
	for _, tm := range got.TMs {
		if tm.Value != kgTMs[tm.Slug] {
			t.Errorf("%s TM = %v, want %v", tm.Slug, tm.Value, kgTMs[tm.Slug])
		}
	}
	history, err := svc.q.ListTrainingMaxHistory(ctx, "press")
	if err != nil || len(history) != 1 {
		t.Errorf("press TM history = %v, %v; want only the kilogram entry", history, err)
	}

	_, cfg, err := svc.activeProgram(ctx)
	if err != nil {
		t.Fatalf("activeProgram: %v", err)
	}
	if cfg.TMIncrementUpper != 2.5 || cfg.TMIncrementLower != 5 {
		t.Errorf("TM steps = %v and %v, want 2.5 and 5", cfg.TMIncrementUpper, cfg.TMIncrementLower)
	}
}

// TestUnitStaysOnceTrainingIsLogged refuses a new unit once a session or a
// past estimate is in the old one, since nothing is ever converted, and still
// takes the same unit or none.
func TestUnitStaysOnceTrainingIsLogged(t *testing.T) {
	cases := []struct {
		name string
		log  func(context.Context, *Service) error
	}{
		{"session done in setup", func(ctx context.Context, s *Service) error {
			return s.SaveSetupHistory(ctx, 0, []PastSession{{Round: 1, Slug: "press", PRReps: 8}})
		}},
		{"past estimate", func(ctx context.Context, s *Service) error {
			return s.AddPastEstimate(ctx, "bench", "2025-01-01", 225, 5)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			svc := newUnitService(t)
			if err := svc.SaveSetup(ctx, lbTMs, ""); err != nil {
				t.Fatalf("SaveSetup: %v", err)
			}
			if err := c.log(ctx, svc); err != nil {
				t.Fatalf("log: %v", err)
			}

			setup, err := svc.Setup(ctx)
			if err != nil || setup.CanChangeUnit {
				t.Fatalf("Setup = %+v, %v; want a unit that can't change", setup, err)
			}
			var input *InputError
			if err := svc.SaveSetup(ctx, kgTMs, "kg"); !errors.As(err, &input) {
				t.Errorf("SaveSetup in kilograms = %v, want an InputError", err)
			}
			if raw, _ := svc.q.GetSettings(ctx); raw.Unit != "lb" || raw.BarWeight != 45 {
				t.Errorf("settings = %s with a %v bar, want lb and 45 untouched", raw.Unit, raw.BarWeight)
			}
			if err := svc.SaveSetup(ctx, lbTMs, "lb"); err != nil {
				t.Errorf("SaveSetup in the same unit: %v", err)
			}
			if err := svc.SaveSetup(ctx, lbTMs, ""); err != nil {
				t.Errorf("SaveSetup without a unit: %v", err)
			}
		})
	}
}

// TestUnknownUnitIsRefused keeps the web form, which skips the spec's enum,
// from saving a unit nothing can display.
func TestUnknownUnitIsRefused(t *testing.T) {
	svc := newUnitService(t)
	var input *InputError
	if err := svc.SaveSetup(context.Background(), kgTMs, "stone"); !errors.As(err, &input) {
		t.Errorf("SaveSetup in stone = %v, want an InputError", err)
	}
}
