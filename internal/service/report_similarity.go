package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// buoyDataCache provides efficient lookup of pre-fetched buoy data by buoy name and time.
type buoyDataCache struct {
	data map[string][]*model.BuoyData // keyed by buoy name
}

type spotLocation struct {
	Latitude  float64
	Longitude float64
}

type buoyLocation struct {
	Name      string
	Latitude  float64
	Longitude float64
}

// newBuoyDataCache creates a cache from batch-fetched buoy data.
func newBuoyDataCache(data map[string][]*model.BuoyData) *buoyDataCache {
	return &buoyDataCache{data: data}
}

// getDataAtTime finds the closest buoy data entry to the target time (within 6 hours).
func (c *buoyDataCache) getDataAtTime(buoyName string, targetTime time.Time) *model.BuoyData {
	entries, ok := c.data[buoyName]
	if !ok || len(entries) == 0 {
		return nil
	}

	// Find the entry closest to target time within 6 hours
	var closest *model.BuoyData
	var minDiff = 6 * time.Hour

	for _, entry := range entries {
		diff := entry.Timestamp.Sub(targetTime)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closest = entry
		}
	}

	return closest
}

// GetSurfReportsWithSimilarBuoyData retrieves surf reports that had similar buoy conditions.
// It takes buoy data parameters (waveHeight, waveDirection, period), a specific buoy name,
// and optionally spot parameters. Returns a list of surf reports with similarity scores.
//
//nolint:gocyclo,funlen // Complex business logic with multiple conditional branches
func (s *ReportService) GetSurfReportsWithSimilarBuoyData(
	ctx context.Context,
	waveHeight float64,
	waveDirection float64,
	period float64,
	buoyName string,
	countryName string,
	regionName string,
	spotName string,
	daysBack int,
	maxResults int,
) ([]map[string]interface{}, error) {
	// Default values
	if daysBack == 0 {
		daysBack = 365 // Default to 1 year
	}
	if maxResults == 0 {
		maxResults = 20 // Default to 20 results
	}

	// Build spot filter
	var countryRegionSpot string
	if countryName != "" && regionName != "" && spotName != "" {
		countryRegionSpot = fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)
	}

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)

	// Query surf reports
	var err error
	var reports []map[string]interface{}

	if countryRegionSpot != "" {
		reportsBySpot, repoErr := s.reportRepo.GetBySpotAndTimeRange(
			ctx,
			countryName,
			regionName,
			spotName,
			cutoffTime,
			time.Now(),
		)
		if repoErr != nil {
			return nil, fmt.Errorf("failed to query surf reports: %w", repoErr)
		}
		reports, err = s.convertReportsToMaps(reportsBySpot)
		if err != nil {
			return nil, err
		}
	} else {
		// Scan all reports (filtered by time)
		reportList, repoErr := s.reportRepo.ScanSince(ctx, cutoffTime, maxResults*10)
		if repoErr != nil {
			return nil, fmt.Errorf("failed to scan surf reports: %w", repoErr)
		}
		reports, err = s.convertReportsToMaps(reportList)
		if err != nil {
			return nil, err
		}
	}

	if len(reports) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Validate buoy name
	if buoyName == "" {
		return nil, fmt.Errorf("buoyName is required")
	}

	// Get the specified buoy location
	buoyLocations := s.getBuoyLocations(ctx)
	buoyLoc, ok := buoyLocations[buoyName]
	if !ok {
		return nil, fmt.Errorf("buoy %s not found", buoyName)
	}

	buoyLat := buoyLoc.Latitude
	buoyLon := buoyLoc.Longitude

	// Get spot location if provided
	var spotLat, spotLon float64
	if countryName != "" && regionName != "" && spotName != "" {
		spotLoc, spotErr := s.getSpotLocation(ctx, countryName, regionName, spotName)
		if spotErr == nil {
			spotLat = spotLoc.Latitude
			spotLon = spotLoc.Longitude
		}
	}

	// Phase 1: Collect all time ranges needed for batch fetching
	type reportTimeInfo struct {
		report          map[string]interface{}
		reportTime      time.Time
		targetBuoyTime  time.Time
		travelTimeHours float64
	}

	var reportInfos []reportTimeInfo
	var minTime, maxTime time.Time
	first := true

	for _, report := range reports {
		timeStr, ok := report["time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, parseErr := parseReportTime(timeStr)
		if parseErr != nil {
			slog.Warn("failed to parse report time", slog.String("time", timeStr), slog.Any("error", parseErr))
			continue
		}

		// Calculate travel time and target buoy time
		var travelTimeHours float64
		targetBuoyTime := reportTime

		// Try to get spot location from report if not provided via parameters
		reportSpotLat, reportSpotLon := spotLat, spotLon
		if reportSpotLat == 0 && reportSpotLon == 0 {
			if crs, ok := report["country_region_spot"].(string); ok {
				parts := strings.Split(crs, "_")
				if len(parts) == 3 {
					spotLoc, spotErr := s.getSpotLocation(ctx, parts[0], parts[1], parts[2])
					if spotErr == nil {
						reportSpotLat = spotLoc.Latitude
						reportSpotLon = spotLoc.Longitude
					}
				}
			}
		}

		// Calculate travel time based on swell direction and spot location
		if reportSpotLat != 0 && reportSpotLon != 0 {
			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, reportSpotLat, reportSpotLon)
			swellTravelDirection := math.Mod(waveDirection+180, 360)
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff
			}

			if angleDiff < 45.0 {
				distance := s.calculateDistance(buoyLat, buoyLon, reportSpotLat, reportSpotLon)
				phaseVelocity := 1.56 * math.Sqrt(period)
				travelTimeHours = distance / (phaseVelocity * 3600)
				travelTimeHours = math.Min(8.0, math.Max(1.0, travelTimeHours))
				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			}
		}

		// Track min/max times for batch fetch
		lookupStart := targetBuoyTime.Add(-6 * time.Hour)
		lookupEnd := targetBuoyTime.Add(6 * time.Hour)
		if first {
			minTime = lookupStart
			maxTime = lookupEnd
			first = false
		} else {
			if lookupStart.Before(minTime) {
				minTime = lookupStart
			}
			if lookupEnd.After(maxTime) {
				maxTime = lookupEnd
			}
		}

		reportInfos = append(reportInfos, reportTimeInfo{
			report:          report,
			reportTime:      reportTime,
			targetBuoyTime:  targetBuoyTime,
			travelTimeHours: travelTimeHours,
		})
	}

	if len(reportInfos) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Phase 2: Batch fetch all buoy data
	batchRequests := []repository.BuoyDataRequest{{
		BuoyName: buoyName,
		Start:    minTime,
		End:      maxTime,
	}}

	batchData, err := s.buoyRepo.GetBatchDataRanges(ctx, batchRequests)
	if err != nil {
		slog.Warn("failed to batch fetch buoy data", slog.Any("error", err))
	}
	cache := newBuoyDataCache(batchData)

	// Phase 3: Calculate similarities using cached data
	type reportWithSimilarity struct {
		report     map[string]interface{}
		similarity float64
	}
	var reportsWithSimilarity []reportWithSimilarity

	for _, info := range reportInfos {
		buoyData := cache.getDataAtTime(buoyName, info.targetBuoyTime)
		if buoyData == nil {
			continue
		}

		buoyDataMap := buoyDataToMap(buoyData)
		similarity := s.calculateBuoyConditionSimilarity(waveHeight, waveDirection, period, buoyDataMap)

		if similarity > 0.7 {
			delete(info.report, "user_email")
			delete(info.report, "UserEmail")
			info.report["similarity"] = similarity
			info.report["buoy_wave_height"] = buoyDataMap["WaveHeight"]
			info.report["buoy_wave_direction"] = buoyDataMap["MeanWaveDirection"]
			info.report["buoy_period"] = buoyDataMap["MaxPeriod"]
			if info.travelTimeHours > 0 {
				info.report["travel_time_hours"] = info.travelTimeHours
			}

			reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
				report:     info.report,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity (highest first)
	sort.Slice(reportsWithSimilarity, func(i, j int) bool {
		return reportsWithSimilarity[i].similarity > reportsWithSimilarity[j].similarity
	})

	// Limit results
	if len(reportsWithSimilarity) > maxResults {
		reportsWithSimilarity = reportsWithSimilarity[:maxResults]
	}

	// Convert back to map slice
	var finalReports []map[string]interface{}
	for _, rws := range reportsWithSimilarity {
		finalReports = append(finalReports, rws.report)
	}

	return finalReports, nil
}

func (s *ReportService) calculateBuoyConditionSimilarity(
	predHeight float64,
	predDirection float64,
	predPeriod float64,
	buoyData map[string]interface{},
) float64 {
	// Extract buoy measurements
	buoyHeight := 0.0
	buoyDirection := 0.0
	buoyPeriod := 0.0

	switch v := buoyData["WaveHeight"].(type) {
	case float64:
		buoyHeight = v
	case string:
		if h, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
			buoyHeight = h
		}
	}

	switch v := buoyData["MeanWaveDirection"].(type) {
	case float64:
		buoyDirection = v
	case string:
		if d, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
			buoyDirection = d
		}
	}

	switch v := buoyData["MaxPeriod"].(type) {
	case float64:
		buoyPeriod = v
	case string:
		if p, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
			buoyPeriod = p
		}
	}

	// Calculate height similarity (within 50% is considered similar)
	maxHeight := predHeight
	if buoyHeight > maxHeight {
		maxHeight = buoyHeight
	}
	if maxHeight < 0.1 {
		maxHeight = 0.1 // Avoid division by zero
	}
	heightDiff := absFloat(predHeight-buoyHeight) / maxHeight
	heightSimilarity := maxFloat(0.0, 1.0-heightDiff/0.5)

	// Calculate direction similarity (within 30 degrees is considered similar)
	directionDiff := absFloat(predDirection - buoyDirection)
	if directionDiff > 180 {
		directionDiff = 360 - directionDiff // Handle wraparound
	}
	directionSimilarity := maxFloat(0.0, 1.0-directionDiff/30.0)

	// Calculate period similarity (within 2 seconds is considered similar)
	periodDiff := absFloat(predPeriod - buoyPeriod)
	periodSimilarity := maxFloat(0.0, 1.0-periodDiff/2.0)

	// Combined similarity (weighted average)
	// Height and direction are more important than period
	return 0.5*heightSimilarity + 0.4*directionSimilarity + 0.1*periodSimilarity
}

// Helper functions
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func parseReportTime(timeStr string) (time.Time, error) {
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try Go time format
	if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr); err == nil {
		return t, nil
	}

	// Try simplified format
	if t, err := time.Parse("2006-01-02 15:04:05 -0700", timeStr); err == nil {
		return t, nil
	}

	// Try ISO format
	if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}

func (s *ReportService) getSpotLocation(ctx context.Context, countryName, regionName, spotName string) (spotLocation, error) {
	location, err := s.locationRepo.GetLocationInfo(ctx, countryName, regionName, spotName)
	if err != nil {
		return spotLocation{}, fmt.Errorf("failed to query location: %w", err)
	}
	if location == nil {
		return spotLocation{}, fmt.Errorf("no location found")
	}

	return spotLocation{
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
	}, nil
}

func (s *ReportService) getBuoyLocations(ctx context.Context) map[string]buoyLocation {
	buoyLocations := make(map[string]buoyLocation)
	locations, err := s.buoyRepo.GetLocations(ctx)
	if err != nil {
		slog.Warn("error loading buoy locations", slog.Any("error", err))
		return buoyLocations
	}

	for name, location := range locations {
		if location == nil {
			continue
		}
		buoyLocations[name] = buoyLocation{
			Name:      location.Name,
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
		}
	}

	return buoyLocations
}

func (s *ReportService) getNearestBuoys(ctx context.Context, spotLat, spotLon float64, numBuoys int) []map[string]interface{} {
	if numBuoys <= 0 {
		numBuoys = 2 // Default to 2 nearest buoys
	}

	allBuoys := s.getBuoyLocations(ctx)
	type buoyWithDistance struct {
		name     string
		buoy     buoyLocation
		distance float64
	}

	buoysWithDistance := make([]buoyWithDistance, 0, len(allBuoys))

	for name, buoy := range allBuoys {
		distance := s.calculateDistance(spotLat, spotLon, buoy.Latitude, buoy.Longitude)

		buoysWithDistance = append(buoysWithDistance, buoyWithDistance{
			buoy:     buoy,
			name:     name,
			distance: distance,
		})
	}

	// Sort by distance (closest first)
	sort.Slice(buoysWithDistance, func(i, j int) bool {
		return buoysWithDistance[i].distance < buoysWithDistance[j].distance
	})

	result := []map[string]interface{}{}
	for i := 0; i < numBuoys && i < len(buoysWithDistance); i++ {
		result = append(result, map[string]interface{}{
			"Name":      buoysWithDistance[i].name,
			"Latitude":  buoysWithDistance[i].buoy.Latitude,
			"Longitude": buoysWithDistance[i].buoy.Longitude,
		})
	}

	return result
}

// Returns distance in meters
func (s *ReportService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (s *ReportService) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)

	bearing := math.Atan2(y, x)
	bearingDegrees := bearing * 180 / math.Pi

	// Convert to 0-360 range
	bearingDegrees = (bearingDegrees + 360)
	return math.Mod(bearingDegrees, 360)
}

func (s *ReportService) getCurrentBuoyData(ctx context.Context, buoyName string) map[string]interface{} {
	data, err := s.buoyRepo.GetLiveData(ctx, buoyName)
	if err != nil {
		slog.Warn("error querying current buoy data", slog.String("buoy", buoyName), slog.Any("error", err))
		return nil
	}

	return buoyDataToMap(data)
}

func buoyDataToMap(data *model.BuoyData) map[string]interface{} {
	if data == nil {
		return nil
	}

	return map[string]interface{}{
		"WaveHeight":        data.WaveHeight,
		"MeanWaveDirection": data.WaveDirection,
		"MaxPeriod":         data.MaxPeriod,
		"dataDateTime":      data.Timestamp.UTC().Format(time.RFC3339),
	}
}

func (s *ReportService) getCurrentWindConditions(
	ctx context.Context,
	countryName, regionName, spotName string,
) (windSpeed, windDirection float64, err error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	forecast, err := s.queryCurrentForecast(ctx, spotID)
	if err != nil {
		return 0, 0, err
	}
	if forecast == nil {
		forecast, err = s.queryHistoricalForecast(ctx, spotID)
		if err != nil || forecast == nil {
			return 0, 0, fmt.Errorf("no forecast data found for spot")
		}
	}

	return s.extractWindData(forecast)
}

func (s *ReportService) getForecastDataAtTime(
	ctx context.Context,
	countryName, regionName, spotName string, targetTime time.Time,
) *model.ForecastDataPoint {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	// Search within ±3 hours window
	startTime := targetTime.Add(-3 * time.Hour)
	endTime := targetTime.Add(3 * time.Hour)

	forecasts, err := s.forecastDataRepo.QueryBetween(ctx, spotID, startTime, endTime, 1)
	if err != nil {
		slog.Warn("error querying forecast data", slog.String("spot", spotID), slog.Any("error", err))
		return nil
	}
	if len(forecasts) == 0 {
		return nil
	}

	return forecasts[0]
}

func (s *ReportService) calculateWindSimilarity(
	currentSpeed, currentDirection float64,
	historicalSpeed, historicalDirection float64,
) float64 {
	// Calculate wind speed similarity (within 20% or 5 m/s, whichever is larger)
	maxSpeed := currentSpeed
	if historicalSpeed > maxSpeed {
		maxSpeed = historicalSpeed
	}
	if maxSpeed < 1.0 {
		maxSpeed = 1.0 // Avoid division by zero
	}

	speedDiff := absFloat(currentSpeed - historicalSpeed)
	speedThreshold := maxFloat(maxSpeed*0.2, 5.0) // 20% or 5 m/s
	speedSimilarity := maxFloat(0.0, 1.0-speedDiff/speedThreshold)

	// Calculate wind direction similarity (within 30 degrees)
	directionDiff := absFloat(currentDirection - historicalDirection)
	if directionDiff > 180 {
		directionDiff = 360 - directionDiff // Handle wraparound
	}
	directionSimilarity := maxFloat(0.0, 1.0-directionDiff/30.0)

	// Combined similarity (equal weight for speed and direction)
	return 0.5*speedSimilarity + 0.5*directionSimilarity
}

// GetSurfReportsWithMatchingConditions retrieves surf reports for a spot where:
// 1. Buoy data at the report time (accounting for travel time) matches current buoy data from nearest buoys
// 2. Wind conditions from forecast data at the report time are similar to current wind conditions
//
//nolint:gocyclo,funlen // Complex business logic with multiple conditional branches
func (s *ReportService) GetSurfReportsWithMatchingConditions(
	ctx context.Context,
	countryName string,
	regionName string,
	spotName string,
	daysBack int,
	maxResults int,
) ([]map[string]interface{}, error) {
	// Default values
	if daysBack == 0 {
		daysBack = 365 // Default to 1 year
	}
	if maxResults == 0 {
		maxResults = 20 // Default to 20 results
	}

	// Step 1: Get spot location
	spotLoc, err := s.getSpotLocation(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, fmt.Errorf("failed to get spot location: %w", err)
	}

	spotLat := spotLoc.Latitude
	spotLon := spotLoc.Longitude

	// Step 2: Find 2 nearest buoys
	nearestBuoys := s.getNearestBuoys(ctx, spotLat, spotLon, 2)
	if len(nearestBuoys) == 0 {
		return nil, fmt.Errorf("no buoys found")
	}

	// Step 3: Get current buoy data for all nearest buoys
	type buoyData struct {
		location      map[string]interface{}
		currentData   map[string]interface{}
		name          string
		waveHeight    float64
		waveDirection float64
		period        float64
	}

	buoyDataList := []buoyData{}
	for _, buoy := range nearestBuoys {
		buoyName, ok := buoy["Name"].(string)
		if !ok {
			continue
		}

		currentBuoyData := s.getCurrentBuoyData(ctx, buoyName)
		if currentBuoyData == nil {
			slog.Warn("no current buoy data found for buoy", slog.String("buoy", buoyName))
			continue
		}

		waveHeight := 0.0
		waveDirection := 0.0
		period := 0.0

		switch v := currentBuoyData["WaveHeight"].(type) {
		case float64:
			waveHeight = v
		case string:
			if h, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				waveHeight = h
			}
		}

		switch v := currentBuoyData["MeanWaveDirection"].(type) {
		case float64:
			waveDirection = v
		case string:
			if d, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				waveDirection = d
			}
		}

		switch v := currentBuoyData["MaxPeriod"].(type) {
		case float64:
			period = v
		case string:
			if p, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				period = p
			}
		}

		// Will be accessed from location map later
		if _, ok1 := buoy["Latitude"].(float64); !ok1 {
			continue
		}
		if _, ok2 := buoy["Longitude"].(float64); !ok2 {
			continue
		}

		buoyDataList = append(buoyDataList, buoyData{
			name:          buoyName,
			location:      buoy,
			currentData:   currentBuoyData,
			waveHeight:    waveHeight,
			waveDirection: waveDirection,
			period:        period,
		})
	}

	if len(buoyDataList) == 0 {
		return nil, fmt.Errorf("no current buoy data found for any nearest buoy")
	}

	// Step 4: Get current wind conditions
	currentWindSpeed, currentWindDirection, err := s.getCurrentWindConditions(ctx, countryName, regionName, spotName)
	if err != nil {
		slog.Warn("could not get current wind conditions", slog.Any("error", err))
		// Continue without wind matching if we can't get current wind data
		currentWindSpeed = 0
		currentWindDirection = 0
	}

	// Step 5: Query historical surf reports
	cutoffTime := time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)
	reportsBySpot, err := s.reportRepo.GetBySpotAndTimeRange(
		ctx,
		countryName,
		regionName,
		spotName,
		cutoffTime,
		time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query surf reports: %w", err)
	}

	reports, err := s.convertReportsToMaps(reportsBySpot)
	if err != nil {
		return nil, err
	}

	if len(reports) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Step 6: First pass - collect time windows for batch fetch
	type reportBuoyInfo struct {
		report          map[string]interface{}
		reportTime      time.Time
		buoyTargetTimes map[string]struct {
			targetTime time.Time
			travelTime float64
		}
	}

	var reportInfos []reportBuoyInfo
	var minTime, maxTime time.Time
	first := true

	for _, report := range reports {
		timeStr, ok := report["time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, parseErr := parseReportTime(timeStr)
		if parseErr != nil {
			slog.Warn("failed to parse report time", slog.String("time", timeStr), slog.Any("error", parseErr))
			continue
		}

		buoyTargetTimes := make(map[string]struct {
			targetTime time.Time
			travelTime float64
		})

		for _, buoyInfo := range buoyDataList {
			buoyLat, ok1 := buoyInfo.location["Latitude"].(float64)
			buoyLon, ok2 := buoyInfo.location["Longitude"].(float64)
			if !ok1 || !ok2 {
				continue
			}

			var travelTimeHours float64
			targetBuoyTime := reportTime

			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, spotLat, spotLon)
			swellTravelDirection := math.Mod(buoyInfo.waveDirection+180, 360)
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff
			}

			if angleDiff < 45.0 {
				distance := s.calculateDistance(buoyLat, buoyLon, spotLat, spotLon)
				phaseVelocity := 1.56 * math.Sqrt(buoyInfo.period)
				travelTimeHours = distance / (phaseVelocity * 3600)
				travelTimeHours = math.Min(8.0, math.Max(1.0, travelTimeHours))
				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			}

			buoyTargetTimes[buoyInfo.name] = struct {
				targetTime time.Time
				travelTime float64
			}{targetBuoyTime, travelTimeHours}

			// Track min/max for batch fetch
			lookupStart := targetBuoyTime.Add(-6 * time.Hour)
			lookupEnd := targetBuoyTime.Add(6 * time.Hour)
			if first {
				minTime = lookupStart
				maxTime = lookupEnd
				first = false
			} else {
				if lookupStart.Before(minTime) {
					minTime = lookupStart
				}
				if lookupEnd.After(maxTime) {
					maxTime = lookupEnd
				}
			}
		}

		reportInfos = append(reportInfos, reportBuoyInfo{
			report:          report,
			reportTime:      reportTime,
			buoyTargetTimes: buoyTargetTimes,
		})
	}

	if len(reportInfos) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Step 7: Batch fetch buoy data for all buoys
	var batchRequests []repository.BuoyDataRequest
	for _, buoyInfo := range buoyDataList {
		batchRequests = append(batchRequests, repository.BuoyDataRequest{
			BuoyName: buoyInfo.name,
			Start:    minTime,
			End:      maxTime,
		})
	}

	batchData, err := s.buoyRepo.GetBatchDataRanges(ctx, batchRequests)
	if err != nil {
		slog.Warn("failed to batch fetch buoy data", slog.Any("error", err))
	}
	cache := newBuoyDataCache(batchData)

	// Step 8: Process reports using cached data
	type reportWithSimilarity struct {
		report             map[string]interface{}
		matchedBuoy        string
		buoySimilarity     float64
		windSimilarity     float64
		combinedSimilarity float64
		travelTimeHours    float64
	}

	var reportsWithSimilarity []reportWithSimilarity

	for _, info := range reportInfos {
		bestMatch := struct {
			historicalData map[string]interface{}
			buoyName       string
			similarity     float64
			travelTime     float64
		}{similarity: 0.0}

		for _, buoyInfo := range buoyDataList {
			targetInfo, ok := info.buoyTargetTimes[buoyInfo.name]
			if !ok {
				continue
			}

			historicalBuoyData := cache.getDataAtTime(buoyInfo.name, targetInfo.targetTime)
			if historicalBuoyData == nil {
				continue
			}

			historicalBuoyMap := buoyDataToMap(historicalBuoyData)
			buoySimilarity := s.calculateBuoyConditionSimilarity(
				buoyInfo.waveHeight, buoyInfo.waveDirection, buoyInfo.period,
				historicalBuoyMap,
			)

			if buoySimilarity > bestMatch.similarity {
				bestMatch.buoyName = buoyInfo.name
				bestMatch.similarity = buoySimilarity
				bestMatch.travelTime = targetInfo.travelTime
				bestMatch.historicalData = historicalBuoyMap
			}
		}

		if bestMatch.similarity < 0.7 {
			continue
		}

		historicalForecast := s.getForecastDataAtTime(ctx, countryName, regionName, spotName, info.reportTime)
		if historicalForecast == nil {
			continue
		}

		if historicalForecast.Data == nil {
			continue
		}

		historicalWindSpeed := extractFloatFromData(historicalForecast.Data, "windSpeed")
		historicalWindDirection := extractFloatFromData(historicalForecast.Data, "windDirection")

		var windSimilarity float64
		if currentWindSpeed > 0 || currentWindDirection > 0 {
			windSimilarity = s.calculateWindSimilarity(
				currentWindSpeed, currentWindDirection,
				historicalWindSpeed, historicalWindDirection,
			)
		} else {
			windSimilarity = 1.0
		}

		if windSimilarity < 0.5 {
			continue
		}

		combinedSimilarity := 0.7*bestMatch.similarity + 0.3*windSimilarity

		delete(info.report, "user_email")
		delete(info.report, "UserEmail")
		info.report["buoy_similarity"] = bestMatch.similarity
		info.report["wind_similarity"] = windSimilarity
		info.report["combined_similarity"] = combinedSimilarity
		info.report["matched_buoy"] = bestMatch.buoyName
		info.report["historical_buoy_wave_height"] = bestMatch.historicalData["WaveHeight"]
		info.report["historical_buoy_wave_direction"] = bestMatch.historicalData["MeanWaveDirection"]
		info.report["historical_buoy_period"] = bestMatch.historicalData["MaxPeriod"]
		info.report["historical_wind_speed"] = historicalWindSpeed
		info.report["historical_wind_direction"] = historicalWindDirection
		if bestMatch.travelTime > 0 {
			info.report["travel_time_hours"] = bestMatch.travelTime
		}

		reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
			report:             info.report,
			buoySimilarity:     bestMatch.similarity,
			windSimilarity:     windSimilarity,
			combinedSimilarity: combinedSimilarity,
			matchedBuoy:        bestMatch.buoyName,
			travelTimeHours:    bestMatch.travelTime,
		})
	}

	// Sort by combined similarity (highest first)
	sort.Slice(reportsWithSimilarity, func(i, j int) bool {
		return reportsWithSimilarity[i].combinedSimilarity > reportsWithSimilarity[j].combinedSimilarity
	})

	// Limit results
	if len(reportsWithSimilarity) > maxResults {
		reportsWithSimilarity = reportsWithSimilarity[:maxResults]
	}

	// Convert back to map slice
	// Initialize as empty slice (not nil) to ensure JSON serialization returns [] instead of null
	finalReports := make([]map[string]interface{}, 0)
	for _, rws := range reportsWithSimilarity {
		finalReports = append(finalReports, rws.report)
	}

	return finalReports, nil
}
