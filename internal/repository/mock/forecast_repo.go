package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.ForecastRepository = (*ForecastRepo)(nil)

type ForecastRepo struct {
	GetSpotForecastFn    func(ctx context.Context, country, region, spot string) ([]*model.Forecast, error)
	GetCurrentConditionsFn func(ctx context.Context, country, region, spot string) (*model.Forecast, error)
	GetForecastAtTimeFn  func(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error)
}

func (m *ForecastRepo) GetSpotForecast(ctx context.Context, country, region, spot string) ([]*model.Forecast, error) {
	if m.GetSpotForecastFn != nil {
		return m.GetSpotForecastFn(ctx, country, region, spot)
	}
	return nil, nil
}

func (m *ForecastRepo) GetCurrentConditions(ctx context.Context, country, region, spot string) (*model.Forecast, error) {
	if m.GetCurrentConditionsFn != nil {
		return m.GetCurrentConditionsFn(ctx, country, region, spot)
	}
	return nil, nil
}

func (m *ForecastRepo) GetForecastAtTime(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error) {
	if m.GetForecastAtTimeFn != nil {
		return m.GetForecastAtTimeFn(ctx, country, region, spot, t)
	}
	return nil, nil
}
