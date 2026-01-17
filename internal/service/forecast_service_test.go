package service

import (
	"context"
	"testing"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestForecastService_GetSpotForecast_PropagatesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "ctx-key", "ctx-value")
	expected := []*model.Forecast{{CountryRegionSpot: "US_CA_Spot"}}
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(callCtx context.Context, country, region, spot string) ([]*model.Forecast, error) {
			if callCtx.Value("ctx-key") != "ctx-value" {
				t.Fatalf("expected context value to be propagated")
			}
			if country != "US" || region != "CA" || spot != "Spot" {
				t.Fatalf("unexpected args: %s %s %s", country, region, spot)
			}
			return expected, nil
		},
	}

	service := NewForecastService(repo)
	got, err := service.GetSpotForecast(ctx, "Spot", "CA", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d forecasts, got %d", len(expected), len(got))
	}
}

func TestForecastService_GetListSpotsForecast_PropagatesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "ctx-key", "ctx-value")
	calls := 0
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(callCtx context.Context, country, region, spot string) ([]*model.Forecast, error) {
			calls++
			if callCtx.Value("ctx-key") != "ctx-value" {
				t.Fatalf("expected context value to be propagated")
			}
			return []*model.Forecast{{CountryRegionSpot: country + "_" + region + "_" + spot}}, nil
		},
	}

	service := NewForecastService(repo)
	spots := []string{"Spot1", "Spot2"}
	_, err := service.GetListSpotsForecast(ctx, spots, "CA", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != len(spots) {
		t.Fatalf("expected %d calls, got %d", len(spots), calls)
	}
}
