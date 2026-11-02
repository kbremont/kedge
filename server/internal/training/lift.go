package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/kbremont/kedge/server/internal/db"
	"github.com/kbremont/kedge/server/internal/program"
)

// The chart is drawn in a ChartWidth x ChartHeight viewBox, with the points
// inside the plot area.
const (
	ChartWidth  = 320.0
	ChartHeight = 160.0

	plotLeft   = 10.0
	plotRight  = 310.0
	plotTop    = 10.0
	plotBottom = 130.0
)

// Lift returns a lift's page, or ErrNotFound: its training max and history,
// the trend of its estimated max, and the estimates from before the app.
func (s *Service) Lift(ctx context.Context, slug string) (*Lift, error) {
	lift, err := s.q.GetLift(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load lift %s: %w", slug, err)
	}

	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}

	l := &Lift{Name: lift.Name, Slug: lift.Slug, Unit: raw.Unit}

	if tm, err := s.q.GetCurrentTrainingMax(ctx, slug); err == nil {
		l.CurrentTM = tm.Value
	}

	history, err := s.q.ListTrainingMaxHistory(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("list training max history for %s: %w", slug, err)
	}
	for _, h := range history {
		l.TMHistory = append(l.TMHistory, TMHistoryItem{
			Value: h.Value,
			Date:  formatTimestamp(h.EffectiveFrom),
			Note:  h.Note,
		})
	}

	ests, err := s.estimates(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("list estimates for %s: %w", slug, err)
	}
	l.Points, l.Path, l.YMin, l.YMax, l.BestE1RM = buildChart(ests)

	for _, e := range ests {
		if !e.Past {
			continue
		}
		l.PastEstimates = append(l.PastEstimates, PastEstimate{
			ID:       e.ID,
			Date:     formatDay(e.When),
			Weight:   e.Weight,
			Reps:     e.Reps,
			Estimate: program.Epley1RM(e.Weight, e.Reps),
		})
	}
	l.Today = s.now().Format(dayLayout)
	return l, nil
}

// estimate is one set, performed or remembered, that says what a lift can do.
type estimate struct {
	When   string // a date, "2006-01-02", optionally followed by a time
	Weight float64
	Reps   int
	Past   bool  // entered from before the app rather than logged in it
	ID     int64 // the past_estimates row, for a past estimate
}

// mergeEstimates puts AMRAP sets logged in the app and estimates entered from
// before it into date order. Each input is already in order. On the same day a
// past estimate sorts first, since what was done before the app is older than
// anything done in it.
func mergeEstimates(logged []db.ListAMRAPSetsRow, past []db.PastEstimate) []estimate {
	out := make([]estimate, 0, len(logged)+len(past))
	i, j := 0, 0
	for i < len(logged) || j < len(past) {
		if j < len(past) && (i == len(logged) || past[j].RecordedOn <= day(logged[i].CompletedAt.String)) {
			p := past[j]
			out = append(out, estimate{When: p.RecordedOn, Weight: p.Weight, Reps: int(p.Reps), Past: true, ID: p.ID})
			j++
			continue
		}
		l := logged[i]
		out = append(out, estimate{When: l.CompletedAt.String, Weight: l.Weight, Reps: int(l.ActualReps.Int64)})
		i++
	}
	return out
}

// day is the date part of a timestamp or a date.
func day(when string) string {
	if len(when) > len(dayLayout) {
		return when[:len(dayLayout)]
	}
	return when
}

// estimates returns everything known about what a lift can do, oldest first:
// the AMRAP sets logged in the app and the estimates entered from before it.
func (s *Service) estimates(ctx context.Context, slug string) ([]estimate, error) {
	logged, err := s.q.ListAMRAPSets(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("list AMRAP sets: %w", err)
	}
	past, err := s.q.ListPastEstimates(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("list past estimates: %w", err)
	}
	return mergeEstimates(logged, past), nil
}

// buildChart turns estimates into a trend of estimated one rep maxes. The
// coordinates are computed here rather than where the chart is drawn so the
// arithmetic can be read in one place.
func buildChart(ests []estimate) (points []ChartPoint, path string, yMin, yMax, best float64) {
	values := make([]float64, 0, len(ests))
	past := make([]bool, 0, len(ests))
	for _, est := range ests {
		e := program.Epley1RM(est.Weight, est.Reps)
		if e <= 0 {
			continue
		}
		values = append(values, e)
		past = append(past, est.Past)
		if e > best {
			best = e
		}
	}
	if len(values) == 0 {
		return nil, "", 0, 0, best
	}

	lo, hi := values[0], values[0]
	for _, v := range values {
		lo = math.Min(lo, v)
		hi = math.Max(hi, v)
	}
	// Pad the range so the line is not pinned to the frame, and round out to
	// tidy axis labels.
	pad := math.Max((hi-lo)*0.15, 5)
	yMin = math.Floor((lo-pad)/5) * 5
	yMax = math.Ceil((hi+pad)/5) * 5
	if yMax == yMin {
		yMax = yMin + 5
	}

	var b strings.Builder
	points = make([]ChartPoint, 0, len(values))
	for i, v := range values {
		// A lone point sits in the middle; a line spans the full width.
		x := (plotLeft + plotRight) / 2
		if len(values) > 1 {
			x = plotLeft + (plotRight-plotLeft)*float64(i)/float64(len(values)-1)
		}
		y := plotBottom - (plotBottom-plotTop)*(v-yMin)/(yMax-yMin)

		if i == 0 {
			fmt.Fprintf(&b, "M %.1f %.1f", x, y)
		} else {
			fmt.Fprintf(&b, " L %.1f %.1f", x, y)
		}
		points = append(points, ChartPoint{X: x, Y: y, Label: FormatWeight(v), Past: past[i]})
	}
	return points, b.String(), yMin, yMax, best
}
