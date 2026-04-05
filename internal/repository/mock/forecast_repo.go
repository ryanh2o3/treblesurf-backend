package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.ForecastRepository = (*ForecastRepo)(nil)

type ForecastRepo struct {
	GetSpotForecastFn       func(ctx context.Context, country, region, spot string, partitionSources []string, granularity string) ([]*model.Forecast, error)
	GetCurrentConditionsFn  func(ctx context.Context, country, region, spot string) (*model.Forecast, error)
	GetForecastAtTimeFn    func(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error)
	QuerySinceFn           func(ctx context.Context, spotID string, since time.Time, limit int) ([]*model.ForecastDataPoint, error)
	QueryBetweenFn         func(ctx context.Context, spotID string, start, end time.Time, limit int) ([]*model.ForecastDataPoint, error)
}

func (m *ForecastRepo) GetSpotForecast(ctx context.Context, country, region, spot string, partitionSources []string, granularity string) ([]*model.Forecast, error) {
	if m.GetSpotForecastFn != nil {
		return m.GetSpotForecastFn(ctx, country, region, spot, partitionSources, granularity)
	}
	return []*model.Forecast{}, nil
}

func (m *ForecastRepo) GetCurrentConditions(ctx context.Context, country, region, spot string) (*model.Forecast, error) {
	if m.GetCurrentConditionsFn != nil {
		return m.GetCurrentConditionsFn(ctx, country, region, spot)
	}
	return nil, repository.ErrNotFound
}

func (m *ForecastRepo) GetForecastAtTime(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error) {
	if m.GetForecastAtTimeFn != nil {
		return m.GetForecastAtTimeFn(ctx, country, region, spot, t)
	}
	return nil, repository.ErrNotFound
}

func (m *ForecastRepo) QuerySince(ctx context.Context, spotID string, since time.Time, limit int) ([]*model.ForecastDataPoint, error) {
	if m.QuerySinceFn != nil {
		return m.QuerySinceFn(ctx, spotID, since, limit)
	}
	return []*model.ForecastDataPoint{}, nil
}

func (m *ForecastRepo) QueryBetween(
	ctx context.Context,
	spotID string,
	start, end time.Time,
	limit int,
) ([]*model.ForecastDataPoint, error) {
	if m.QueryBetweenFn != nil {
		return m.QueryBetweenFn(ctx, spotID, start, end, limit)
	}
	return []*model.ForecastDataPoint{}, nil
}
