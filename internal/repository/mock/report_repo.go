package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.ReportRepository = (*ReportRepo)(nil)

type ReportRepo struct {
	CreateFn              func(ctx context.Context, report *model.SurfReport) error
	GetBySpotFn           func(ctx context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error)
	GetBySpotAndTimeRangeFn func(ctx context.Context, country, region, spot string, start, end time.Time) ([]*model.SurfReport, error)
	ScanSinceFn             func(ctx context.Context, since time.Time, limit int) ([]*model.SurfReport, error)
	AnonymizeByUserEmailFn  func(ctx context.Context, email string) error
}

func (m *ReportRepo) Create(ctx context.Context, report *model.SurfReport) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, report)
	}
	return nil
}

func (m *ReportRepo) GetBySpot(ctx context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error) {
	if m.GetBySpotFn != nil {
		return m.GetBySpotFn(ctx, country, region, spot, limit)
	}
	return []*model.SurfReport{}, nil
}

func (m *ReportRepo) GetBySpotAndTimeRange(
	ctx context.Context,
	country, region, spot string,
	start, end time.Time,
) ([]*model.SurfReport, error) {
	if m.GetBySpotAndTimeRangeFn != nil {
		return m.GetBySpotAndTimeRangeFn(ctx, country, region, spot, start, end)
	}
	return []*model.SurfReport{}, nil
}

func (m *ReportRepo) ScanSince(ctx context.Context, since time.Time, limit int) ([]*model.SurfReport, error) {
	if m.ScanSinceFn != nil {
		return m.ScanSinceFn(ctx, since, limit)
	}
	return []*model.SurfReport{}, nil
}

func (m *ReportRepo) AnonymizeByUserEmail(ctx context.Context, email string) error {
	if m.AnonymizeByUserEmailFn != nil {
		return m.AnonymizeByUserEmailFn(ctx, email)
	}
	return nil
}
