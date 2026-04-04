package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

type forecastCtxKey string

const (
	forecastCtxKeyValue   forecastCtxKey = "ctx-key"
	forecastCtxValue      string         = "ctx-value"
	forecastTestCountry                  = "Ireland"
	forecastTestRegion                   = "Donegal"
	forecastTestSpot                     = "Bundoran"
	forecastTestSpotID                   = "Ireland_Donegal_Bundoran"
	forecastTestSpotIDRoss               = "Ireland_Donegal_Rossnowlagh"
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

	got, err := service.GetSpotForecast(ctx, "Spot", "CA", "US", "")
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
	_, err = service.GetListSpotsForecast(ctx, spots, "CA", "US", "")
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
		{CountryRegionSpot: forecastTestSpotID},
		{CountryRegionSpot: forecastTestSpotIDRoss},
	}
	repo := &mockrepo.ForecastRepo{
		GetRegionForecastFn: func(_ context.Context, country, region string, _ time.Time) ([]*model.Forecast, error) {
			if country != forecastTestCountry || region != forecastTestRegion {
				t.Fatalf("unexpected args: %s %s", country, region)
			}
			return expected, nil
		},
	}

	service, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegionForecast(ctx, forecastTestRegion, forecastTestCountry)
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
		repo := &mockrepo.ForecastRepo{
			GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
				if country != forecastTestCountry || region != forecastTestRegion || spot != forecastTestSpot {
					t.Fatalf("unexpected args: %s %s %s", country, region, spot)
				}
				return []*model.Forecast{
					{
						ForecastTimestamp: "1700000000#imi_swan",
						Source:            sourceIMISwan,
						Country:           country,
						Region:            region,
						Spot:              spot,
						CountryRegionSpot: forecastTestSpotID,
						Data:              map[string]interface{}{"swellHeight": 2.0},
					},
					{
						ForecastTimestamp: "1700000000#weatherkit",
						Source:            sourceWeatherKit,
						Country:           country,
						Region:            region,
						Spot:              spot,
						CountryRegionSpot: forecastTestSpotID,
						Data: map[string]interface{}{
							"temperature": 18.5,
							"windSpeed":   15.0,
						},
					},
				}, nil
			},
		}

		service, err := NewForecastService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetCurrentWeather(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 forecast, got %d", len(got))
		}
		if got[0].CountryRegionSpot != forecastTestSpotID {
			t.Fatalf("expected %s, got %s", forecastTestSpotID, got[0].CountryRegionSpot)
		}
		if got[0].Source != sourceComposedIreland {
			t.Fatalf("expected composed Ireland primary, got %s", got[0].Source)
		}
	})

	t.Run("handles nil current conditions", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.ForecastRepo{
			GetSpotForecastFn: func(_ context.Context, _, _, _ string) ([]*model.Forecast, error) {
				return []*model.Forecast{}, nil
			},
		}

		service, err := NewForecastService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		_, err = service.GetCurrentWeather(ctx, "Unknown", forecastTestRegion, forecastTestCountry, "")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestNewForecastService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewForecastService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestForecastService_GetSpotForecast_PicksPrimarySource(t *testing.T) {
	ctx := context.Background()
	// Ireland: no imi_swan+weatherkit and no WeatherKit → no default primary (Stormglass ignored).
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: sourceStormglass, Country: country, Region: region, Spot: spot},
				{ForecastTimestamp: "1700000000#imi_swan", Source: sourceIMISwan, Country: country, Region: region, Spot: spot},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no Ireland primary without SWAN+WK merge, got %d", len(got))
	}
}

func TestForecastService_GetSpotForecastGrouped_ReturnsAllSources(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: sourceStormglass, Country: country, Region: region, Spot: spot},
				{ForecastTimestamp: "1700000000#imi_swan", Source: sourceIMISwan, Country: country, Region: region, Spot: spot},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	groups, err := svc.GetSpotForecastGrouped(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Sources) != 2 {
		t.Fatalf("expected 2 sources in group, got %d", len(groups[0].Sources))
	}
	if groups[0].Sources[sourceStormglass] == nil || groups[0].Sources[sourceIMISwan] == nil {
		t.Fatalf("expected both sources in group")
	}
	if groups[0].PrimarySource != sourceComposedIreland {
		t.Fatalf("expected primary_source %s for Ireland, got %s", sourceComposedIreland, groups[0].PrimarySource)
	}
}

func TestForecastService_GetSpotForecast_IrelandPrefersPremergedRow(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{
					ForecastTimestamp: "1700000000#imi_swan+weatherkit",
					Source:            sourceComposedIreland,
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"swellHeight": 9.0, "temperature": 99.0,
					},
				},
				{
					ForecastTimestamp: "1700000000#imi_swan",
					Source:            sourceIMISwan,
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"swellHeight": 1.0, "temperature": 10.0,
					},
				},
				{
					ForecastTimestamp: "1700000000#weatherkit",
					Source:            sourceWeatherKit,
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"temperature": 20.0, "windSpeed": 5.0,
					},
				},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 forecast, got %d", len(got))
	}
	if got[0].Source != sourceComposedIreland {
		t.Fatalf("expected pre-merged row, got source %s", got[0].Source)
	}
	if v := floatFromMap(got[0].Data, "swellHeight", "swell_height"); v != 9.0 {
		t.Fatalf("expected swell from pre-merged row 9, got %v", v)
	}
	if v := floatFromMap(got[0].Data, "temperature"); v != 99.0 {
		t.Fatalf("expected temperature from pre-merged row 99, got %v", v)
	}
}

func TestForecastService_GetSpotForecast_IrelandComposedPrimary(t *testing.T) {
	ctx := context.Background()
	const composedSource = "imi_swan+weatherkit"
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{
					ForecastTimestamp: "1700000000#imi_swan",
					Source:            "imi_swan",
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"swellHeight": 2.5, "wave_height": 2.5,
						"temperature": 10.0,
					},
				},
				{
					ForecastTimestamp: "1700000000#weatherkit",
					Source:            "weatherkit",
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"temperature": 18.0, "windSpeed": 15.0, "windDirection": 270.0,
						"swellHeight": 1.0,
					},
				},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 composed forecast, got %d", len(got))
	}
	f := got[0]
	if f.Source != composedSource {
		t.Fatalf("expected composed source %s, got %s", composedSource, f.Source)
	}
	// Wave data from SWAN
	if v := floatFromMap(f.Data, "swellHeight", "swell_height"); v != 2.5 {
		t.Fatalf("expected swellHeight from SWAN 2.5, got %v", v)
	}
	// Weather data from WeatherKit
	if v := floatFromMap(f.Data, "temperature"); v != 18.0 {
		t.Fatalf("expected temperature from WeatherKit 18, got %v", v)
	}
	if v := floatFromMap(f.Data, "windSpeed", "wind_speed"); v != 15.0 {
		t.Fatalf("expected windSpeed from WeatherKit 15, got %v", v)
	}
}

func TestForecastService_GetSpotForecast_IrelandPrimaryExcludesStormglass(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: sourceStormglass, Country: country, Region: region, Spot: spot},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no default primary when only Stormglass exists for Ireland, got %d rows", len(got))
	}
}

func TestForecastService_GetSpotForecast_IrelandStormglassStillViaSourceParam(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: sourceStormglass, Country: country, Region: region, Spot: spot},
				{ForecastTimestamp: "1700000000#imi_swan", Source: sourceIMISwan, Country: country, Region: region, Spot: spot},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, sourceStormglass)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 forecast, got %d", len(got))
	}
	if got[0].Source != sourceStormglass {
		t.Fatalf("expected stormglass when source=stormglass, got %s", got[0].Source)
	}
}

func TestForecastService_GetSpotForecast_IrelandSkipsStormglassOnlyHours(t *testing.T) {
	ctx := context.Background()
	const composedSource = "imi_swan+weatherkit"
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: sourceStormglass, Country: country, Region: region, Spot: spot},
				{
					ForecastTimestamp: "1700003600#imi_swan",
					Source:            sourceIMISwan,
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data:              map[string]interface{}{"swellHeight": 2.0},
				},
				{
					ForecastTimestamp: "1700003600#weatherkit",
					Source:            sourceWeatherKit,
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data:              map[string]interface{}{"temperature": 12.0},
				},
			}, nil
		},
	}
	svc, err := NewForecastService(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err := svc.GetSpotForecast(ctx, forecastTestSpot, forecastTestRegion, forecastTestCountry, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 composed hour (stormglass-only hour skipped), got %d", len(got))
	}
	if got[0].Source != composedSource {
		t.Fatalf("expected %s, got %s", composedSource, got[0].Source)
	}
}

func floatFromMap(m map[string]interface{}, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch x := v.(type) {
			case float64:
				return x
			case int:
				return float64(x)
			}
		}
	}
	return 0
}
