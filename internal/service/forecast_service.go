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
	spotName, regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.GetSpotForecastWithContext(context.Background(), spotName, regionName, countryName)
}

func (s *ForecastService) GetSpotForecastWithContext(
	ctx context.Context,
	spotName, regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName)
}

func (s *ForecastService) GetListSpotsForecast(
	spots []string,
	regionName, countryName string,
) ([][]*model.Forecast, error) {
	return s.GetListSpotsForecastWithContext(context.Background(), spots, regionName, countryName)
}

func (s *ForecastService) GetListSpotsForecastWithContext(
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

func (s *ForecastService) GetRegionForecast(regionName, countryName string) ([]*model.Forecast, error) {
	return s.GetRegionForecastWithContext(context.Background(), regionName, countryName)
}

func (s *ForecastService) GetRegionForecastWithContext(
	ctx context.Context,
	regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.forecasts.GetRegionForecast(ctx, countryName, regionName, time.Now())
}

func (s *ForecastService) GetCurrentWeather(
	spotName, regionName, countryName string,
) ([]*model.Forecast, error) {
	return s.GetCurrentWeatherWithContext(context.Background(), spotName, regionName, countryName)
}

func (s *ForecastService) GetCurrentWeatherWithContext(
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
