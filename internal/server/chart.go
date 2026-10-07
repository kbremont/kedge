package server

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/kbremont/531-tracker/internal/db"
	"github.com/kbremont/531-tracker/internal/program"
	"github.com/kbremont/531-tracker/internal/views"
)

// Plot area within the 320x160 viewBox declared by the chart template.
const (
	plotLeft   = 10.0
	plotRight  = 310.0
	plotTop    = 10.0
	plotBottom = 130.0
)

func (s *Server) handleLift(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slug := r.PathValue("slug")
	lift, err := s.q.GetLift(ctx, slug)
	if err != nil {
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}

	raw, _, err := s.loadSettings(ctx)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	vm := views.LiftVM{Name: lift.Name, Slug: lift.Slug, Unit: raw.Unit}

	if tm, err := s.q.GetCurrentTrainingMax(ctx, slug); err == nil {
		vm.CurrentTM = tm.Value
	}

	history, err := s.q.ListTrainingMaxHistory(ctx, slug)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list training max history for %s: %w", slug, err))
		return
	}
	for _, h := range history {
		vm.TMHistory = append(vm.TMHistory, views.TMHistoryItem{
			Value: h.Value,
			Date:  formatTimestamp(h.EffectiveFrom),
			Note:  h.Note,
		})
	}

	amraps, err := s.q.ListAMRAPSets(ctx, slug)
	if err != nil {
		s.fail(w, r, fmt.Errorf("list AMRAP sets for %s: %w", slug, err))
		return
	}
	vm.Points, vm.Path, vm.YMin, vm.YMax, vm.BestE1RM = buildChart(amraps)

	s.render(w, r, views.LiftDetail(vm))
}

// buildChart turns AMRAP sets into a trend of estimated one rep maxes. The
// coordinates are computed here rather than in the template so the arithmetic
// can be read in one place.
func buildChart(rows []db.ListAMRAPSetsRow) (points []views.ChartPoint, path string, yMin, yMax, best float64) {
	values := make([]float64, 0, len(rows))
	for _, row := range rows {
		e := program.Epley1RM(row.Weight, int(row.ActualReps.Int64))
		if e <= 0 {
			continue
		}
		values = append(values, e)
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
	points = make([]views.ChartPoint, 0, len(values))
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
		points = append(points, views.ChartPoint{X: x, Y: y, Label: views.FmtWeight(v)})
	}
	return points, b.String(), yMin, yMax, best
}
