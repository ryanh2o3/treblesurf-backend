package service

import (
	"context"
	"testing"

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

func TestNewForecastService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewForecastService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}
