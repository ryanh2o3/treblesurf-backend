// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

const (
	sourceStormglass          = "stormglass"
	sourceIMISwan             = "imi_swan"
	sourceWeatherKit          = "weatherkit"
	sourceComposedIreland     = "imi_swan+weatherkit"
)

// ForecastGroup holds forecasts for one timestamp, keyed by source.
type ForecastGroup struct {
	Time              time.Time
	ForecastTimestamp string
	Sources           map[string]*model.Forecast
	PrimarySource     string
}

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

// primarySourceForRegion returns the preferred forecast source name for a region.
// For Ireland the API composes primary from imi_swan (waves) + weatherkit (weather); this returns the label used when no compose is done.
func primarySourceForRegion(country, _ string) string {
	if strings.EqualFold(country, "ireland") {
		return sourceIMISwan // Ireland primary is composed (SWAN + WeatherKit); this is the wave source name
	}
	return sourceStormglass
}

// Weather-related data keys (from WeatherKit). When composing Ireland primary we take wave data from
// IMI SWAN (all of swan.Data) and overlay these weather fields from WeatherKit.
var weatherDataKeys = []string{
	"temperature", "windSpeed", "wind_speed", "windDirection", "wind_direction",
	"pressure", "humidity", "precipitation", "relativeWindDirection", "relative_wind_direction",
	"generated_at", "generatedAt",
}

// composeIrelandForecast merges wave data from IMI SWAN and weather data from Apple WeatherKit into one forecast.
// If either source is nil, returns the non-nil one; if both nil, returns nil.
func composeIrelandForecast(swan, weatherkit *model.Forecast) *model.Forecast {
	if swan == nil && weatherkit == nil {
		return nil
	}
	if swan == nil {
		return weatherkit
	}
	if weatherkit == nil {
		return swan
	}
	// Merge Data: wave keys from SWAN, weather keys from WeatherKit. Prefer SWAN for shared keys in wave set.
	data := make(map[string]interface{})
	for k, v := range swan.Data {
		data[k] = v
	}
	for _, k := range weatherDataKeys {
		if v, ok := weatherkit.Data[k]; ok {
			data[k] = v
		}
	}
	// Copy base from SWAN, overlay weather-only fields from WeatherKit
	out := *swan
	out.Data = data
	out.Source = sourceComposedIreland
	if weatherkit.GeneratedAt != "" {
		out.GeneratedAt = weatherkit.GeneratedAt
	}
	return &out
}

// primaryForecastFromGroup returns the forecast to use as "primary" for this group.
// If preferredSource is set, that source is used when present. For Ireland, when both imi_swan and weatherkit
// exist, returns a composed forecast (SWAN waves + WeatherKit weather). Otherwise uses PrimarySource or first.
func primaryForecastFromGroup(g *ForecastGroup, country, preferredSource string) *model.Forecast {
	if g == nil {
		return nil
	}
	if preferredSource != "" {
		if f := g.Sources[preferredSource]; f != nil {
			return f
		}
	}
	if strings.EqualFold(country, "ireland") {
		if f := composeIrelandForecast(g.Sources[sourceIMISwan], g.Sources[sourceWeatherKit]); f != nil {
			return f
		}
	}
	if f := g.Sources[g.PrimarySource]; f != nil {
		return f
	}
	for _, f := range g.Sources {
		return f
	}
	return nil
}

// baseTimestamp returns the sort-key prefix (unix ts) from forecast_timestamp, which may be "ts" or "ts#source".
func baseTimestamp(forecastTimestamp string) string {
	if idx := strings.Index(forecastTimestamp, "#"); idx >= 0 {
		return forecastTimestamp[:idx]
	}
	return forecastTimestamp
}

// groupForecastsByTime groups forecasts by base timestamp and sets PrimarySource per group.
func (s *ForecastService) groupForecastsByTime(forecasts []*model.Forecast, country, region string) []ForecastGroup {
	byTime := make(map[string]*ForecastGroup)
	primary := primarySourceForRegion(country, region)

	for _, f := range forecasts {
		if f == nil {
			continue
		}
		base := baseTimestamp(f.ForecastTimestamp)
		if byTime[base] == nil {
			byTime[base] = &ForecastGroup{
				Time:              f.Date,
				ForecastTimestamp: base,
				Sources:           make(map[string]*model.Forecast),
				PrimarySource:     primary,
			}
			if !f.Date.IsZero() {
				byTime[base].Time = f.Date
			} else if base != "" {
				if unix, err := parseInt64(base); err == nil {
					byTime[base].Time = time.Unix(unix, 0).UTC()
				}
			}
		}
		grp := byTime[base]
		src := f.Source
		if src == "" {
			src = "default"
		}
		grp.Sources[src] = f
	}

	order := make([]string, 0, len(byTime))
	for k := range byTime {
		order = append(order, k)
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	out := make([]ForecastGroup, 0, len(order))
	for _, base := range order {
		out = append(out, *byTime[base])
	}
	return out
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func (s *ForecastService) GetSpotForecast(
	ctx context.Context,
	spotName, regionName, countryName string,
	preferredSource string,
) ([]*model.Forecast, error) {
	raw, err := s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}
	groups := s.groupForecastsByTime(raw, countryName, regionName)
	out := make([]*model.Forecast, 0, len(groups))
	for i := range groups {
		if f := primaryForecastFromGroup(&groups[i], countryName, preferredSource); f != nil {
			out = append(out, f)
		}
	}
	return out, nil
}

// GetSpotForecastGrouped returns forecasts grouped by time with all sources (for sources=all).
func (s *ForecastService) GetSpotForecastGrouped(
	ctx context.Context,
	spotName, regionName, countryName string,
) ([]ForecastGroup, error) {
	raw, err := s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}
	return s.groupForecastsByTime(raw, countryName, regionName), nil
}

func (s *ForecastService) GetListSpotsForecast(
	ctx context.Context,
	spots []string,
	regionName, countryName string,
	preferredSource string,
) ([][]*model.Forecast, error) {
	results := make([][]*model.Forecast, 0, len(spots))
	for _, spot := range spots {
		forecast, err := s.GetSpotForecast(ctx, spot, regionName, countryName, preferredSource)
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
	preferredSource string,
) ([]*model.Forecast, error) {
	// Get all sources for the current window so we can apply preferred/compose logic
	raw, err := s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}
	groups := s.groupForecastsByTime(raw, countryName, regionName)
	if len(groups) == 0 {
		return nil, repository.ErrNotFound
	}
	f := primaryForecastFromGroup(&groups[0], countryName, preferredSource)
	if f == nil {
		return nil, repository.ErrNotFound
	}
	return []*model.Forecast{f}, nil
}
