// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// ForecastService provides business logic for forecast operations.
type ForecastService struct {
	forecasts repository.ForecastRepository
}

// NewForecastService creates a new ForecastService with the given repository.
// Returns an error if the repository is nil.
func NewForecastService(forecasts repository.ForecastRepository) (*ForecastService, error) {
	if forecasts == nil {
		return nil, fmt.Errorf("forecast repository is required")
	}
	return &ForecastService{forecasts: forecasts}, nil
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
