package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

type forecastCtxKey string

const (
	forecastCtxKeyValue   forecastCtxKey = "ctx-key"
	forecastCtxValue      string         = "ctx-value"
)

func TestForecastService_GetSpotForecast_PropagatesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), forecastCtxKeyValue, forecastCtxValue)
	expected := []*model.Forecast{{CountryRegionSpot: "US_CA_Spot"}}
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(callCtx context.Context, country, region, spot string) ([]*model.Forecast, error) {
			if callCtx.Value(forecastCtxKeyValue) != forecastCtxValue {
				t.Fatalf("expected context value to be propagated")
			}
			if country != "US" || region != "CA" || spot != "Spot" {
				t.Fatalf("unexpected args: %s %s %s", country, region, spot)
			}
			return expected, nil
		},
	}

	service, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpotForecast(ctx, "Spot", "CA", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d forecasts, got %d", len(expected), len(got))
	}
}

func TestForecastService_GetListSpotsForecast_PropagatesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), forecastCtxKeyValue, forecastCtxValue)
	calls := 0
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(callCtx context.Context, country, region, spot string) ([]*model.Forecast, error) {
			calls++
			if callCtx.Value(forecastCtxKeyValue) != forecastCtxValue {
				t.Fatalf("expected context value to be propagated")
			}
			return []*model.Forecast{{CountryRegionSpot: country + "_" + region + "_" + spot}}, nil
		},
	}

	service, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	spots := []string{"Spot1", "Spot2"}
	_, err = service.GetListSpotsForecast(ctx, spots, "CA", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != len(spots) {
		t.Fatalf("expected %d calls, got %d", len(spots), calls)
	}
}

func TestForecastService_GetRegionForecast(t *testing.T) {
	ctx := context.Background()
	expected := []*model.Forecast{
		{CountryRegionSpot: "Ireland_Donegal_Bundoran"},
		{CountryRegionSpot: "Ireland_Donegal_Rossnowlagh"},
	}
	repo := &mockrepo.ForecastRepo{
		GetRegionForecastFn: func(_ context.Context, country, region string, _ time.Time) ([]*model.Forecast, error) {
			if country != "Ireland" || region != "Donegal" {
				t.Fatalf("unexpected args: %s %s", country, region)
			}
			return expected, nil
		},
	}

	service, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegionForecast(ctx, "Donegal", "Ireland")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d forecasts, got %d", len(expected), len(got))
	}
}

func TestForecastService_GetCurrentWeather(t *testing.T) {
	t.Run("returns current weather forecast", func(t *testing.T) {
		ctx := context.Background()
		expected := &model.Forecast{
			CountryRegionSpot: "Ireland_Donegal_Bundoran",
			Temperature:       18.5,
			WindSpeed:         15.0,
		}
		repo := &mockrepo.ForecastRepo{
			GetCurrentConditionsFn: func(_ context.Context, country, region, spot string) (*model.Forecast, error) {
				if country != "Ireland" || region != "Donegal" || spot != "Bundoran" {
					t.Fatalf("unexpected args: %s %s %s", country, region, spot)
				}
				return expected, nil
			},
		}

		service, err := NewForecastService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetCurrentWeather(ctx, "Bundoran", "Donegal", "Ireland")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 forecast, got %d", len(got))
		}
		if got[0].CountryRegionSpot != expected.CountryRegionSpot {
			t.Fatalf("expected %s, got %s", expected.CountryRegionSpot, got[0].CountryRegionSpot)
		}
	})

	t.Run("handles nil current conditions", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.ForecastRepo{
			GetCurrentConditionsFn: func(_ context.Context, _, _, _ string) (*model.Forecast, error) {
				return nil, nil
			},
		}

		service, err := NewForecastService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetCurrentWeather(ctx, "Unknown", "Donegal", "Ireland")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil && len(got) != 0 {
			t.Fatalf("expected nil or empty, got %d items", len(got))
		}
	})
}

func TestNewForecastService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewForecastService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}
