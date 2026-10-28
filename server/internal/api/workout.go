package api

import (
	"context"
)

// StartWorkout starts the next session, or resumes the one in progress, and
// answers with it. When there is nothing to start, or a training max decision
// is waiting, it is a 409 and the app fetches the dashboard to show why.
func (s *Server) StartWorkout(ctx context.Context, _ StartWorkoutRequestObject) (StartWorkoutResponseObject, error) {
	id, err := s.svc.StartWorkout(ctx)
	if err != nil {
		return nil, err
	}
	w, err := s.workout(ctx, id)
	if err != nil {
		return nil, err
	}
	return StartWorkout200JSONResponse(w), nil
}

func (s *Server) GetWorkout(ctx context.Context, req GetWorkoutRequestObject) (GetWorkoutResponseObject, error) {
	w, err := s.workout(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return GetWorkout200JSONResponse(w), nil
}

func (s *Server) workout(ctx context.Context, id int64) (Workout, error) {
	w, err := s.svc.Workout(ctx, id)
	if err != nil {
		return Workout{}, err
	}
	return toWorkout(w), nil
}

// LogSet records the reps done on a set. The session it answers with carries
// the rest to time before the next set. The spec makes reps required: 0 is a
// set failed, so a body without it is refused rather than read as 0.
func (s *Server) LogSet(ctx context.Context, req LogSetRequestObject) (LogSetResponseObject, error) {
	if err := s.svc.LogSet(ctx, req.ID, req.Idx, req.Body.Reps); err != nil {
		return nil, err
	}
	w, err := s.workout(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return LogSet200JSONResponse(w), nil
}

func (s *Server) UndoSet(ctx context.Context, req UndoSetRequestObject) (UndoSetResponseObject, error) {
	if err := s.svc.UndoSet(ctx, req.ID, req.Idx); err != nil {
		return nil, err
	}
	w, err := s.workout(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return UndoSet200JSONResponse(w), nil
}

func (s *Server) FinishWorkout(ctx context.Context, req FinishWorkoutRequestObject) (FinishWorkoutResponseObject, error) {
	if err := s.svc.FinishWorkout(ctx, req.ID); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return FinishWorkout200JSONResponse(d), nil
}

// AbandonWorkout removes a session started by mistake.
func (s *Server) AbandonWorkout(ctx context.Context, req AbandonWorkoutRequestObject) (AbandonWorkoutResponseObject, error) {
	if err := s.svc.AbandonWorkout(ctx, req.ID); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return AbandonWorkout200JSONResponse(d), nil
}
