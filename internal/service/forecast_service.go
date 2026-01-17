package service

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

type ForecastService struct {
	forecasts repository.ForecastRepository
}

func NewForecastService(forecasts repository.ForecastRepository) *ForecastService {
	return &ForecastService{forecasts: forecasts}
}

func (s *ForecastService) GetSpotForecast(
	ctx context.Context,
	spotName, regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName)
}

func (s *ForecastService) GetListSpotsForecast(
	ctx context.Context,
	spots []string,
	regionName, countryName string,
) ([][]*model.Forecast, error) {
	results := make([][]*model.Forecast, 0, len(spots))
	for _, spot := range spots {
		forecast, err := s.forecasts.GetSpotForecast(ctx, countryName, regionName, spot)
		if err != nil {
			return nil, err
		}
		results = append(results, forecast)
	}

	return results, nil
}

func (s *ForecastService) GetRegionForecast(
	ctx context.Context,
	regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.forecasts.GetRegionForecast(ctx, countryName, regionName, time.Now())
}

func (s *ForecastService) GetCurrentWeather(
	ctx context.Context,
	spotName, regionName, countryName string,
) ([]*model.Forecast, error) {
	current, err := s.forecasts.GetCurrentConditions(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}
	return []*model.Forecast{current}, nil
}
