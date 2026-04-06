// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/logging"
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
	locations repository.LocationRepository
}

// NewForecastService creates a new ForecastService with the given repositories.
// Returns an error if the forecast repository is nil.
func NewForecastService(forecasts repository.ForecastRepository, locations repository.LocationRepository) (*ForecastService, error) {
	if forecasts == nil {
		return nil, fmt.Errorf("forecast repository is required")
	}
	if locations == nil {
		return nil, fmt.Errorf("location repository is required")
	}
	return &ForecastService{forecasts: forecasts, locations: locations}, nil
}

// primarySourceForRegion returns the metadata primary_source label for grouped API responses.
// Ireland default primary is always imi_swan+weatherkit (persisted row); elsewhere Stormglass.
func primarySourceForRegion(country, _ string) string {
	if strings.EqualFold(country, "ireland") {
		return sourceComposedIreland
	}
	return sourceStormglass
}

// partitionSourcesForPreferred selects which DynamoDB partitions to query for spot forecast reads.
// Default: Ireland → imi_swan+weatherkit only; elsewhere → stormglass only. Explicit source= queries one partition.
func partitionSourcesForPreferred(country, preferredSource string) []string {
	if preferredSource != "" {
		return []string{preferredSource}
	}
	if strings.EqualFold(country, "ireland") {
		return []string{sourceComposedIreland}
	}
	return []string{sourceStormglass}
}

func partitionSourcesAll() []string {
	return []string{sourceStormglass, sourceComposedIreland}
}

// primaryForecastFromGroup returns the forecast to use as "primary" for this group.
// If preferredSource is set, that source is used when present (e.g. source=stormglass).
//
// For Ireland with no preferred source, primary is only the persisted imi_swan+weatherkit row.
// Partial hours (Stormglass-only, legacy split rows, etc.) yield no primary row.
// Stormglass is never default for Ireland; use source=stormglass (if rows exist).
//
// Other countries use PrimarySource for the region, then any remaining source.
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
		return irelandDefaultPrimary(g)
	}
	if f := g.Sources[g.PrimarySource]; f != nil {
		return f
	}
	for _, f := range g.Sources {
		return f
	}
	return nil
}

// irelandDefaultPrimary selects the default (non–source-param) primary for Ireland.
func irelandDefaultPrimary(g *ForecastGroup) *model.Forecast {
	return g.Sources[sourceComposedIreland]
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

// filterIrelandReadRows drops rows the API no longer uses for Ireland once the forecaster writes only
// imi_swan+weatherkit (no standalone imi_swan/weatherkit; no stormglass for in-bounds Irish shelf).
// Dynamo still returns legacy rows until TTL; this avoids grouping/parsing them on the default path.
// When preferredSource is set, the client asked for that source — do not filter.
// For sources=all, standalone imi_swan and weatherkit are omitted; stormglass + imi_swan+weatherkit remain.
func filterIrelandReadRows(rows []*model.Forecast, country, preferredSource string, sourcesAll bool) []*model.Forecast {
	if !strings.EqualFold(country, "ireland") || preferredSource != "" {
		return rows
	}
	if len(rows) == 0 {
		return rows
	}
	out := make([]*model.Forecast, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		switch r.Source {
		case sourceIMISwan, sourceWeatherKit:
			continue
		}
		if !sourcesAll && r.Source != sourceComposedIreland {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (s *ForecastService) GetSpotForecast(
	ctx context.Context,
	spotName, regionName, countryName string,
	preferredSource string,
	granularity string,
) ([]*model.Forecast, error) {
	raw, err := s.forecasts.GetSpotForecast(
		ctx,
		countryName,
		regionName,
		spotName,
		partitionSourcesForPreferred(countryName, preferredSource),
		granularity,
	)
	if err != nil {
		return nil, err
	}
	raw = filterIrelandReadRows(raw, countryName, preferredSource, false)

	groupStart := time.Now()
	groups := s.groupForecastsByTime(raw, countryName, regionName)
	groupElapsed := time.Since(groupStart)

	selectStart := time.Now()
	out := make([]*model.Forecast, 0, len(groups))
	for i := range groups {
		if f := primaryForecastFromGroup(&groups[i], countryName, preferredSource); f != nil {
			out = append(out, f)
		}
	}
	selectElapsed := time.Since(selectStart)

	logging.FromContext(ctx).Info("forecast timing: service get_spot_forecast",
		slog.String("spot", spotName),
		slog.String("region", regionName),
		slog.String("country", countryName),
		slog.String("preferred_source", preferredSource),
		slog.Int("raw_row_count", len(raw)),
		slog.Int("group_count", len(groups)),
		slog.Int("output_row_count", len(out)),
		slog.Int64("group_by_time_ms", groupElapsed.Milliseconds()),
		slog.Int64("primary_per_group_ms", selectElapsed.Milliseconds()),
	)

	return out, nil
}

// GetSpotForecastGrouped returns forecasts grouped by time with all sources (for sources=all).
func (s *ForecastService) GetSpotForecastGrouped(
	ctx context.Context,
	spotName, regionName, countryName string,
	granularity string,
) ([]ForecastGroup, error) {
	raw, err := s.forecasts.GetSpotForecast(ctx, countryName, regionName, spotName, partitionSourcesAll(), granularity)
	if err != nil {
		return nil, err
	}
	raw = filterIrelandReadRows(raw, countryName, "", true)
	groupStart := time.Now()
	groups := s.groupForecastsByTime(raw, countryName, regionName)
	logging.FromContext(ctx).Info("forecast timing: service get_spot_forecast_grouped",
		slog.String("spot", spotName),
		slog.String("region", regionName),
		slog.String("country", countryName),
		slog.Int("raw_row_count", len(raw)),
		slog.Int("group_count", len(groups)),
		slog.Int64("group_by_time_ms", time.Since(groupStart).Milliseconds()),
	)
	return groups, nil
}

func (s *ForecastService) GetListSpotsForecast(
	ctx context.Context,
	spots []string,
	regionName, countryName string,
	preferredSource string,
	granularity string,
) ([][]*model.Forecast, error) {
	results := make([][]*model.Forecast, 0, len(spots))
	for _, spot := range spots {
		forecast, err := s.GetSpotForecast(ctx, spot, regionName, countryName, preferredSource, granularity)
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
	spots, err := s.locations.GetSpots(ctx, countryName, regionName)
	if err != nil {
		return nil, fmt.Errorf("listing spots for region: %w", err)
	}
	if len(spots) == 0 {
		return nil, repository.ErrNotFound
	}
	var out []*model.Forecast
	for _, loc := range spots {
		if loc == nil {
			continue
		}
		parts := strings.Split(loc.CountryRegionSpot, "/")
		if len(parts) < 3 {
			continue
		}
		spotName := parts[2]
		rows, err := s.GetSpotForecast(ctx, spotName, regionName, countryName, "", repository.ForecastGranularityHourly)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	if len(out) == 0 {
		return nil, repository.ErrNotFound
	}
	return out, nil
}

func (s *ForecastService) GetCurrentWeather(
	ctx context.Context,
	spotName, regionName, countryName string,
	preferredSource string,
) ([]*model.Forecast, error) {
	f, err := s.forecasts.GetCurrentConditions(
		ctx,
		countryName,
		regionName,
		spotName,
		partitionSourcesForPreferred(countryName, preferredSource),
	)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, repository.ErrNotFound
	}
	// Ireland: preserve behavior where default responses only return the persisted imi_swan+weatherkit row.
	rows := filterIrelandReadRows([]*model.Forecast{f}, countryName, preferredSource, false)
	if len(rows) == 0 {
		return nil, repository.ErrNotFound
	}
	return rows, nil
}
