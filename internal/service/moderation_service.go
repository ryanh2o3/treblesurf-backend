package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/google/uuid"
)

// ModerationService handles admin moderation operations.
type ModerationService struct {
	reportRepo repository.ContentReportRepository
	actionRepo repository.ModerationActionRepository
	userRepo   repository.UserRepository
}

// NewModerationService creates a new ModerationService.
func NewModerationService(
	reportRepo repository.ContentReportRepository,
	actionRepo repository.ModerationActionRepository,
	userRepo repository.UserRepository,
) (*ModerationService, error) {
	switch {
	case reportRepo == nil:
		return nil, fmt.Errorf("content report repository is required")
	case actionRepo == nil:
		return nil, fmt.Errorf("moderation action repository is required")
	case userRepo == nil:
		return nil, fmt.Errorf("user repository is required")
	}
	return &ModerationService{
		reportRepo: reportRepo,
		actionRepo: actionRepo,
		userRepo:   userRepo,
	}, nil
}

// GetPendingReports retrieves pending reports for the moderation queue.
func (s *ModerationService) GetPendingReports(
	ctx context.Context,
	limit, offset int,
) ([]*model.ContentReport, error) {
	return s.reportRepo.GetPendingReports(ctx, limit, offset)
}

// GetReportDetails retrieves a report with additional context.
func (s *ModerationService) GetReportDetails(
	ctx context.Context,
	reportID string,
) (*model.ContentReport, error) {
	return s.reportRepo.GetByID(ctx, reportID)
}

// ResolveReportInput contains the input for resolving a report.
type ResolveReportInput struct {
	ReportID   string
	Resolution string
	Notes      string
	AdminID    string
}

// ResolveReport resolves a report and creates a moderation action record.
func (s *ModerationService) ResolveReport(
	ctx context.Context,
	input ResolveReportInput,
) error {
	// Get the report to find the target
	report, err := s.reportRepo.GetByID(ctx, input.ReportID)
	if err != nil {
		return fmt.Errorf("getting report: %w", err)
	}

	// Resolve the report
	if err := s.reportRepo.Resolve(
		ctx,
		input.ReportID,
		input.Resolution,
		input.Notes,
		input.AdminID,
	); err != nil {
		return fmt.Errorf("resolving report: %w", err)
	}

	// Create moderation action record
	action := &model.ModerationAction{
		ID:          uuid.NewString(),
		ReportID:    input.ReportID,
		ActionType:  input.Resolution,
		TargetType:  string(model.TargetTypeReport),
		TargetID:    report.SurfReportID,
		PerformedBy: input.AdminID,
		Notes:       input.Notes,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.actionRepo.Create(ctx, action); err != nil {
		return fmt.Errorf("creating action: %w", err)
	}

	return nil
}

// DismissReport dismisses a report as not requiring action.
func (s *ModerationService) DismissReport(
	ctx context.Context,
	reportID, adminID, notes string,
) error {
	// Update the report status to dismissed
	if err := s.reportRepo.UpdateStatus(
		ctx,
		reportID,
		string(model.ReportStatusDismissed),
		adminID,
	); err != nil {
		return fmt.Errorf("dismissing report: %w", err)
	}

	// Create moderation action record
	action := &model.ModerationAction{
		ID:          uuid.NewString(),
		ReportID:    reportID,
		ActionType:  string(model.ResolutionNoAction),
		TargetType:  string(model.TargetTypeReport),
		TargetID:    reportID,
		PerformedBy: adminID,
		Notes:       notes,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.actionRepo.Create(ctx, action); err != nil {
		return fmt.Errorf("creating action: %w", err)
	}

	return nil
}

// GetModerationHistory retrieves moderation actions with pagination.
func (s *ModerationService) GetModerationHistory(
	ctx context.Context,
	limit, offset int,
) ([]*model.ModerationAction, error) {
	return s.actionRepo.List(ctx, limit, offset)
}

// GetReportActions retrieves all actions for a specific report.
func (s *ModerationService) GetReportActions(
	ctx context.Context,
	reportID string,
) ([]*model.ModerationAction, error) {
	return s.actionRepo.GetByReportID(ctx, reportID)
}
