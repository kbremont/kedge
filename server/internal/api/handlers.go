package api

import (
	"context"
	"fmt"
	"time"

	"github.com/kbremont/kedge/server/internal/training"
)

// Each method is one operation in the spec. Success is the typed response
// for its status; an error from training is returned as it is, and fail maps
// it to a status and a message.

func (s *Server) GetInfo(_ context.Context, _ GetInfoRequestObject) (GetInfoResponseObject, error) {
	return GetInfo200JSONResponse{APIVersion: APIVersion, Version: s.version}, nil
}

// Login exchanges the password for a session token. It is the same token the
// browser keeps in its cookie, so the two sign in and expire alike.
func (s *Server) Login(_ context.Context, req LoginRequestObject) (LoginResponseObject, error) {
	if !s.sessions.CheckPassword(req.Body.Password) {
		return Login401JSONResponse{UnauthorizedJSONResponse{Error: "Incorrect password."}}, nil
	}
	token, expires := s.sessions.Issue()
	return Login200JSONResponse{Token: token, ExpiresAt: expires.UTC().Format(time.RFC3339)}, nil
}

// GetDashboard answers the Today screen. Before training maxes are entered
// it reports needsSetup, where the web redirects to the setup page.
func (s *Server) GetDashboard(ctx context.Context, _ GetDashboardRequestObject) (GetDashboardResponseObject, error) {
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return GetDashboard200JSONResponse(d), nil
}

// dashboard is the Today screen, which every change that leads back to it
// answers with.
func (s *Server) dashboard(ctx context.Context) (Dashboard, error) {
	d, err := s.svc.Dashboard(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	return toDashboard(d), nil
}

func (s *Server) GetHistory(ctx context.Context, _ GetHistoryRequestObject) (GetHistoryResponseObject, error) {
	hist, err := s.svc.History(ctx)
	if err != nil {
		return nil, err
	}
	return GetHistory200JSONResponse(toHistory(hist)), nil
}

func (s *Server) GetLift(ctx context.Context, req GetLiftRequestObject) (GetLiftResponseObject, error) {
	l, err := s.lift(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return GetLift200JSONResponse(l), nil
}

func (s *Server) lift(ctx context.Context, slug string) (Lift, error) {
	l, err := s.svc.Lift(ctx, slug)
	if err != nil {
		return Lift{}, err
	}
	return toLift(l), nil
}

func (s *Server) AddPastEstimate(ctx context.Context, req AddPastEstimateRequestObject) (AddPastEstimateResponseObject, error) {
	in := req.Body
	if err := s.svc.AddPastEstimate(ctx, req.Slug, in.Date, in.Weight, in.Reps); err != nil {
		return nil, err
	}
	l, err := s.lift(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return AddPastEstimate200JSONResponse(l), nil
}

func (s *Server) DeletePastEstimate(ctx context.Context, req DeletePastEstimateRequestObject) (DeletePastEstimateResponseObject, error) {
	if err := s.svc.DeletePastEstimate(ctx, req.Slug, req.ID); err != nil {
		return nil, err
	}
	l, err := s.lift(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return DeletePastEstimate200JSONResponse(l), nil
}

func (s *Server) GetSettings(ctx context.Context, _ GetSettingsRequestObject) (GetSettingsResponseObject, error) {
	set, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	return GetSettings200JSONResponse(set), nil
}

func (s *Server) settings(ctx context.Context) (Settings, error) {
	set, err := s.svc.Settings(ctx)
	if err != nil {
		return Settings{}, err
	}
	return toSettings(set), nil
}

func (s *Server) SaveSettings(ctx context.Context, req SaveSettingsRequestObject) (SaveSettingsResponseObject, error) {
	in := req.Body
	change := training.SettingsInput{
		Rounding:          in.Rounding,
		BarWeight:         in.BarWeight,
		Plates:            in.Plates,
		Warmups:           in.Warmups,
		RestWarmupSeconds: in.RestWarmupSeconds,
		RestMainSeconds:   in.RestMainSeconds,
		RestSuppSeconds:   in.RestSuppSeconds,
		BBB:               make(map[string]training.LiftBBB, len(in.BBB)),
	}
	for slug, b := range in.BBB {
		change.BBB[slug] = training.LiftBBB{Mode: string(b.Mode), Value: b.Value}
	}
	if err := s.svc.SaveSettings(ctx, change); err != nil {
		return nil, err
	}
	set, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	return SaveSettings200JSONResponse(set), nil
}

func (s *Server) SetTrainingMax(ctx context.Context, req SetTrainingMaxRequestObject) (SetTrainingMaxResponseObject, error) {
	in := req.Body
	if err := s.svc.SetTrainingMax(ctx, in.Lift, in.Value, deref(in.Note)); err != nil {
		return nil, err
	}
	set, err := s.settings(ctx)
	if err != nil {
		return nil, err
	}
	return SetTrainingMax200JSONResponse(set), nil
}

// AcceptReview writes the training maxes settled on at the start of a block.
// A review no longer pending is a 409.
func (s *Server) AcceptReview(ctx context.Context, req AcceptReviewRequestObject) (AcceptReviewResponseObject, error) {
	if err := s.svc.AcceptReview(ctx, req.Body.Key, req.Body.Tms); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return AcceptReview200JSONResponse(d), nil
}

// DecideTMTest settles a failed TM test: keep the training max, or lower it.
func (s *Server) DecideTMTest(ctx context.Context, req DecideTMTestRequestObject) (DecideTMTestResponseObject, error) {
	in := req.Body
	if err := s.svc.DecideTMTest(ctx, req.ID, in.Action == TMTestActionKeep, deref(in.TM)); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return DecideTMTest200JSONResponse(d), nil
}

func (s *Server) SkipCycle(ctx context.Context, _ SkipCycleRequestObject) (SkipCycleResponseObject, error) {
	if err := s.svc.SkipCycle(ctx); err != nil {
		return nil, err
	}
	d, err := s.dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return SkipCycle200JSONResponse(d), nil
}

// GetBackup streams a consistent snapshot of the database, as the web's
// download does.
func (s *Server) GetBackup(ctx context.Context, _ GetBackupRequestObject) (GetBackupResponseObject, error) {
	backup, err := s.svc.Backup(ctx)
	if err != nil {
		return nil, err
	}
	resp := GetBackup200ApplicationOctetStreamResponse{
		// Sending the answer closes the body, which removes the snapshot.
		Body: backup,
		Headers: GetBackup200ResponseHeaders{
			ContentDisposition: fmt.Sprintf("attachment; filename=%q", backup.Name),
		},
	}
	if info, err := backup.Stat(); err == nil {
		resp.ContentLength = info.Size()
	}
	return resp, nil
}
