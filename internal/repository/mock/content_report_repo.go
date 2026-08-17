package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.ContentReportRepository = (*ContentReportRepo)(nil)

// ContentReportRepo is a mock implementation of ContentReportRepository.
type ContentReportRepo struct {
	CreateFn               func(ctx context.Context, report *model.ContentReport) error
	GetByIDFn              func(ctx context.Context, id string) (*model.ContentReport, error)
	GetBySurfReportIDFn    func(ctx context.Context, surfReportID string) ([]*model.ContentReport, error)
	GetByReporterIDFn      func(ctx context.Context, userID string) ([]*model.ContentReport, error)
	GetPendingReportsFn    func(ctx context.Context, limit, offset int) ([]*model.ContentReport, error)
	UpdateStatusFn         func(ctx context.Context, id, status, reviewedBy string) error
	ResolveFn              func(ctx context.Context, id, resolution, notes, reviewedBy string) error
	CountByReporterSinceFn func(ctx context.Context, userID string, since time.Time) (int, error)
}

func (m *ContentReportRepo) Create(ctx context.Context, report *model.ContentReport) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, report)
	}
	return nil
}

func (m *ContentReportRepo) GetByID(ctx context.Context, id string) (*model.ContentReport, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *ContentReportRepo) GetBySurfReportID(
	ctx context.Context,
	surfReportID string,
) ([]*model.ContentReport, error) {
	if m.GetBySurfReportIDFn != nil {
		return m.GetBySurfReportIDFn(ctx, surfReportID)
	}
	return []*model.ContentReport{}, nil
}

func (m *ContentReportRepo) GetByReporterID(
	ctx context.Context,
	userID string,
) ([]*model.ContentReport, error) {
	if m.GetByReporterIDFn != nil {
		return m.GetByReporterIDFn(ctx, userID)
	}
	return []*model.ContentReport{}, nil
}

func (m *ContentReportRepo) GetPendingReports(
	ctx context.Context,
	limit, offset int,
) ([]*model.ContentReport, error) {
	if m.GetPendingReportsFn != nil {
		return m.GetPendingReportsFn(ctx, limit, offset)
	}
	return []*model.ContentReport{}, nil
}

func (m *ContentReportRepo) UpdateStatus(
	ctx context.Context,
	id, status, reviewedBy string,
) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status, reviewedBy)
	}
	return nil
}

func (m *ContentReportRepo) Resolve(
	ctx context.Context,
	id, resolution, notes, reviewedBy string,
) error {
	if m.ResolveFn != nil {
		return m.ResolveFn(ctx, id, resolution, notes, reviewedBy)
	}
	return nil
}

func (m *ContentReportRepo) CountByReporterSince(
	ctx context.Context,
	userID string,
	since time.Time,
) (int, error) {
	if m.CountByReporterSinceFn != nil {
		return m.CountByReporterSinceFn(ctx, userID, since)
	}
	return 0, nil
}
