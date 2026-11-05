package api

import (
	"context"

	"github.com/kbremont/kedge/server/internal/training"
)

// Onboarding: training maxes first, then the cycle the user is on, then the
// sessions of it already done. Picking a cycle writes nothing, so it has no
// route; the app goes straight to that cycle's checklist, or to the
// dashboard to start fresh.

func (s *Server) GetSetup(ctx context.Context, _ GetSetupRequestObject) (GetSetupResponseObject, error) {
	setup, err := s.svc.Setup(ctx)
	if err != nil {
		return nil, err
	}
	return GetSetup200JSONResponse(toSetup(setup)), nil
}

// SaveSetup records the first training maxes and answers with the next
// question: where in the programme the user is.
func (s *Server) SaveSetup(ctx context.Context, req SaveSetupRequestObject) (SaveSetupResponseObject, error) {
	unit := ""
	if req.Body.Unit != nil {
		unit = string(*req.Body.Unit)
	}
	if err := s.svc.SaveSetup(ctx, req.Body.Tms, unit); err != nil {
		return nil, err
	}
	pos, err := s.setupPosition(ctx)
	if err != nil {
		return nil, err
	}
	return SaveSetup200JSONResponse(pos), nil
}

func (s *Server) GetSetupPosition(ctx context.Context, _ GetSetupPositionRequestObject) (GetSetupPositionResponseObject, error) {
	pos, err := s.setupPosition(ctx)
	if err != nil {
		return nil, err
	}
	return GetSetupPosition200JSONResponse(pos), nil
}

func (s *Server) setupPosition(ctx context.Context) (SetupPosition, error) {
	pos, err := s.svc.SetupPosition(ctx)
	if err != nil {
		return SetupPosition{}, err
	}
	return toSetupPosition(pos), nil
}

func (s *Server) GetSetupHistory(ctx context.Context, req GetSetupHistoryRequestObject) (GetSetupHistoryResponseObject, error) {
	hist, err := s.svc.SetupHistory(ctx, req.Cycle)
	if err != nil {
		return nil, err
	}
	return GetSetupHistory200JSONResponse(toSetupHistory(hist)), nil
}

// SaveSetupHistory joins the block at a cycle with the sessions already done,
// and answers with the dashboard that now picks up after them.
func (s *Server) SaveSetupHistory(ctx context.Context, req SaveSetupHistoryRequestObject) (SaveSetupHistoryResponseObject, error) {
	done := make([]training.PastSession, 0, len(req.Body.Done))
	for _, d := range req.Body.Done {
		done = append(done, training.PastSession{Round: d.Round, Slug: d.Slug, PRReps: deref(d.PRReps)})
	}
	if err := s.svc.SaveSetupHistory(ctx, req.Cycle, done); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return SaveSetupHistory200JSONResponse(d), nil
}
