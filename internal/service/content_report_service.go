package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/google/uuid"
)

// Rate limiting constants as specified in requirements.
const (
	maxReportsPerHour = 5
	maxReportsPerDay  = 20
	maxDescriptionLen = 500
)

// ContentReportService errors.
var (
	ErrRateLimitExceeded   = errors.New("rate limit exceeded")
	ErrInvalidReportReason = errors.New("invalid report reason")
	ErrDescriptionTooLong  = errors.New("description exceeds maximum length")
	ErrSurfReportNotFound  = errors.New("surf report not found")
	ErrSelfReport          = errors.New("cannot report your own content")
)

// ContentReportService handles content report business logic.
type ContentReportService struct {
	reportRepo     repository.ContentReportRepository
	surfReportRepo repository.ReportRepository
	userRepo       repository.UserRepository
}

// NewContentReportService creates a new ContentReportService.
func NewContentReportService(
	reportRepo repository.ContentReportRepository,
	surfReportRepo repository.ReportRepository,
	userRepo repository.UserRepository,
) (*ContentReportService, error) {
	switch {
	case reportRepo == nil:
		return nil, fmt.Errorf("content report repository is required")
	case surfReportRepo == nil:
		return nil, fmt.Errorf("surf report repository is required")
	case userRepo == nil:
		return nil, fmt.Errorf("user repository is required")
	}
	return &ContentReportService{
		reportRepo:     reportRepo,
		surfReportRepo: surfReportRepo,
		userRepo:       userRepo,
	}, nil
}

// SubmitReportInput contains the input for submitting a report.
type SubmitReportInput struct {
	SurfReportID string
	Reason       string
	Description  string
	ReporterID   string
}

// SubmitReport validates and creates a new content report.
func (s *ContentReportService) SubmitReport(
	ctx context.Context,
	input SubmitReportInput,
) (*model.ContentReport, error) {
	// Validate reason
	if !model.IsValidReportReason(input.Reason) {
		return nil, ErrInvalidReportReason
	}

	// Validate description length
	if len(input.Description) > maxDescriptionLen {
		return nil, ErrDescriptionTooLong
	}

	// Check rate limits
	if err := s.checkRateLimits(ctx, input.ReporterID); err != nil {
		return nil, err
	}

	// Create the report
	now := time.Now().UTC()
	report := &model.ContentReport{
		ID:             uuid.NewString(),
		SurfReportID:   input.SurfReportID,
		ReporterUserID: input.ReporterID,
		Reason:         input.Reason,
		Description:    input.Description,
		Status:         string(model.ReportStatusPending),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("creating report: %w", err)
	}

	return report, nil
}

// checkRateLimits verifies the user hasn't exceeded rate limits.
func (s *ContentReportService) checkRateLimits(ctx context.Context, userID string) error {
	now := time.Now().UTC()

	// Check hourly limit
	oneHourAgo := now.Add(-1 * time.Hour)
	hourlyCount, err := s.reportRepo.CountByReporterSince(ctx, userID, oneHourAgo)
	if err != nil {
		return fmt.Errorf("checking hourly rate: %w", err)
	}
	if hourlyCount >= maxReportsPerHour {
		return ErrRateLimitExceeded
	}

	// Check daily limit
	oneDayAgo := now.Add(-24 * time.Hour)
	dailyCount, err := s.reportRepo.CountByReporterSince(ctx, userID, oneDayAgo)
	if err != nil {
		return fmt.Errorf("checking daily rate: %w", err)
	}
	if dailyCount >= maxReportsPerDay {
		return ErrRateLimitExceeded
	}

	return nil
}

// GetReportByID retrieves a report by its ID.
func (s *ContentReportService) GetReportByID(
	ctx context.Context,
	id string,
) (*model.ContentReport, error) {
	return s.reportRepo.GetByID(ctx, id)
}

// GetReportsBySurfReport retrieves all reports for a specific surf report.
func (s *ContentReportService) GetReportsBySurfReport(
	ctx context.Context,
	surfReportID string,
) ([]*model.ContentReport, error) {
	return s.reportRepo.GetBySurfReportID(ctx, surfReportID)
}

// GetReportsByReporter retrieves all reports submitted by a user.
func (s *ContentReportService) GetReportsByReporter(
	ctx context.Context,
	userID string,
) ([]*model.ContentReport, error) {
	return s.reportRepo.GetByReporterID(ctx, userID)
}
