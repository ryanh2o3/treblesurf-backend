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
)

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
			return nil, fmt.Errorf("failed to query surf reports: %v", repoErr)
		}
		reports, err = s.convertReportsToMaps(reportsBySpot)
		if err != nil {
			return nil, err
		}
	} else {
		// Scan all reports (filtered by time)
		reportList, repoErr := s.reportRepo.ScanSince(ctx, cutoffTime, maxResults*10)
		if repoErr != nil {
			return nil, fmt.Errorf("failed to scan surf reports: %v", repoErr)
		}
		reports, err = s.convertReportsToMaps(reportList)
		if err != nil {
			return nil, err
		}
	}

	if len(reports) == 0 {
		return []map[string]interface{}{}, nil
	}

	// For each report, get buoy data at that time and calculate similarity
	type reportWithSimilarity struct {
		report     map[string]interface{}
		similarity float64
	}

	var reportsWithSimilarity []reportWithSimilarity

	// Validate buoy name
	if buoyName == "" {
		return nil, fmt.Errorf("buoyName is required")
	}

	// Get the specified buoy location
	buoyLocations := s.getBuoyLocations(ctx)
	buoyLocation, ok := buoyLocations[buoyName]
	if !ok {
		return nil, fmt.Errorf("buoy %s not found", buoyName)
	}

	buoyLat, ok := buoyLocation["Latitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid latitude for buoy %s", buoyName)
	}

	buoyLon, ok := buoyLocation["Longitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid longitude for buoy %s", buoyName)
	}

	// Use only the specified buoy for data lookup
	buoyPriority := []string{buoyName}

	for _, report := range reports {
		// Parse report time
		timeStr, ok := report["Time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, err := parseReportTime(timeStr)
		if err != nil {
			slog.Warn("failed to parse report time", slog.String("time", timeStr), slog.Any("error", err))
			continue
		}

		// Calculate travel time if spot location is available
		var travelTimeHours float64
		var targetBuoyTime = reportTime

		// Try to get spot location from report
		// First try from report fields, then from query parameters
		var spotLat, spotLon float64

		// Extract spot location from report country_region_spot
		if countryRegionSpot, ok := report["country_region_spot"].(string); ok {
			// Format is "Country_Region_Spot"
			parts := strings.Split(countryRegionSpot, "_")
			if len(parts) == 3 {
				spotLoc, err := s.getSpotLocation(ctx, parts[0], parts[1], parts[2])
				if err == nil && spotLoc != nil {
					if lat, ok := spotLoc["Latitude"].(float64); ok {
						spotLat = lat
					}
					if lon, ok := spotLoc["Longitude"].(float64); ok {
						spotLon = lon
					}
				}
			}
		}

		// Fallback to query parameters if not found in report
		if spotLat == 0 && spotLon == 0 && countryName != "" && regionName != "" && spotName != "" {
			spotLoc, err := s.getSpotLocation(ctx, countryName, regionName, spotName)
			if err == nil && spotLoc != nil {
				if lat, ok := spotLoc["Latitude"].(float64); ok {
					spotLat = lat
				}
				if lon, ok := spotLoc["Longitude"].(float64); ok {
					spotLon = lon
				}
			}
		}

		// Calculate travel time based on swell direction and spot location
		if spotLat != 0 && spotLon != 0 {
			// Calculate bearing from buoy to spot
			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, spotLat, spotLon)

			// Check if spot is downwave from buoy (within reasonable angle of swell direction)
			// Swell direction is where waves are coming FROM, so we need to check if the spot
			// is in the direction the swell is traveling TO
			swellTravelDirection := math.Mod(waveDirection+180, 360) // Direction waves are traveling TO

			// Calculate angle difference between swell travel direction and bearing to spot
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff // Handle wraparound
			}

			// If angle difference is less than 45 degrees, spot is downwave - use travel time
			// If angle difference is greater, spot is not in swell path - minimal/no travel time
			if angleDiff < 45.0 {
				// Spot is downwave - calculate normal travel time
				distance := s.calculateDistance(buoyLat, buoyLon, spotLat, spotLon)

				// Calculate travel time using phase velocity
				// Phase velocity = 1.56 * sqrt(period) in m/s for deep water waves
				phaseVelocity := 1.56 * math.Sqrt(period) // m/s

				// Travel time in hours = distance (meters) / (phase_velocity * 3600)
				travelTimeHours = distance / (phaseVelocity * 3600)

				// Cap travel time between 1-8 hours (matching prediction service)
				if travelTimeHours > 8.0 {
					travelTimeHours = 8.0
				}
				if travelTimeHours < 1.0 {
					travelTimeHours = 1.0
				}

				// Look for buoy data at (report_time - travel_time)
				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			} else {
				// Spot is not directly downwave - use minimal travel time (or same time)
				// For perpendicular/non-direct swells, look at buoy data closer to report time
				targetBuoyTime = reportTime
				travelTimeHours = 0.0
			}
		}

		// Get buoy data at target time (accounting for travel time if calculated)
		buoyData := s.getBuoyDataAtTime(ctx, targetBuoyTime, buoyPriority)
		if buoyData == nil {
			continue // Skip if no buoy data found
		}

		// Calculate similarity
		similarity := s.calculateBuoyConditionSimilarity(
			waveHeight, waveDirection, period,
			buoyData,
		)

		// Only include reports with similarity > 0.7
		if similarity > 0.7 {
			// Remove sensitive fields
			delete(report, "UserEmail")

			// Add similarity score and buoy data info
			report["similarity"] = similarity
			report["buoy_wave_height"] = buoyData["WaveHeight"]
			report["buoy_wave_direction"] = buoyData["MeanWaveDirection"]
			report["buoy_period"] = buoyData["MaxPeriod"]
			if travelTimeHours > 0 {
				report["travel_time_hours"] = travelTimeHours
			}

			reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
				report:     report,
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

func (s *ReportService) getBuoyDataAtTime(ctx context.Context, targetTime time.Time, buoyPriority []string) map[string]interface{} {
	// Look for data within 6 hours of target time
	startTime := targetTime.Add(-6 * time.Hour)
	endTime := targetTime.Add(6 * time.Hour)

	// Try multiple buoys in order of priority
	for _, buoyName := range buoyPriority {
		data, err := s.buoyRepo.GetDataRange(ctx, buoyName, startTime, endTime)
		if err != nil {
			slog.Warn("error querying buoy data", slog.String("buoy", buoyName), slog.Time("target_time", targetTime), slog.Any("error", err))
			continue
		}
		if len(data) > 0 {
			return buoyDataToMap(data[0])
		}
	}

	return nil
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

func (s *ReportService) getSpotLocation(ctx context.Context, countryName, regionName, spotName string) (map[string]interface{}, error) {
	location, err := s.locationRepo.GetLocationInfo(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, fmt.Errorf("failed to query location: %v", err)
	}
	if location == nil {
		return nil, fmt.Errorf("no location found")
	}

	return map[string]interface{}{
		"Latitude":  location.Latitude,
		"Longitude": location.Longitude,
	}, nil
}

func (s *ReportService) getBuoyLocations(ctx context.Context) map[string]map[string]interface{} {
	buoyLocations := make(map[string]map[string]interface{})
	locations, err := s.buoyRepo.GetLocations(ctx)
	if err != nil {
		slog.Warn("error loading buoy locations", slog.Any("error", err))
		return buoyLocations
	}

	for name, location := range locations {
		if location == nil {
			continue
		}
		buoyLocations[name] = map[string]interface{}{
			"Name":      location.Name,
			"Latitude":  location.Latitude,
			"Longitude": location.Longitude,
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
		buoy     map[string]interface{}
		name     string
		distance float64
	}

	buoysWithDistance := make([]buoyWithDistance, 0, len(allBuoys))

	for name, buoy := range allBuoys {
		buoyLat, ok1 := buoy["Latitude"].(float64)
		buoyLon, ok2 := buoy["Longitude"].(float64)

		if !ok1 || !ok2 {
			continue // Skip if coordinates are missing or invalid
		}

		distance := s.calculateDistance(spotLat, spotLon, buoyLat, buoyLon)

		// Create a copy of the buoy map with the name added
		buoyCopy := make(map[string]interface{})
		for k, v := range buoy {
			buoyCopy[k] = v
		}
		buoyCopy["Name"] = name

		buoysWithDistance = append(buoysWithDistance, buoyWithDistance{
			buoy:     buoyCopy,
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
		result = append(result, buoysWithDistance[i].buoy)
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
) map[string]interface{} {
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
		return nil, fmt.Errorf("failed to get spot location: %v", err)
	}

	spotLat, ok := spotLoc["Latitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid latitude for spot")
	}

	spotLon, ok := spotLoc["Longitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid longitude for spot")
	}

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
		return nil, fmt.Errorf("failed to query surf reports: %v", err)
	}

	reports, err := s.convertReportsToMaps(reportsBySpot)
	if err != nil {
		return nil, err
	}

	// Step 6: Process each report and calculate similarity
	type reportWithSimilarity struct {
		report             map[string]interface{}
		matchedBuoy        string
		buoySimilarity     float64
		windSimilarity     float64
		combinedSimilarity float64
		travelTimeHours    float64
	}

	var reportsWithSimilarity []reportWithSimilarity

	for _, report := range reports {
		// Parse report time
		timeStr, ok := report["Time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, err := parseReportTime(timeStr)
		if err != nil {
			slog.Warn("failed to parse report time", slog.String("time", timeStr), slog.Any("error", err))
			continue
		}

		// Try each buoy and find the best match
		bestMatch := struct {
			historicalData map[string]interface{}
			buoyName       string
			similarity     float64
			travelTime     float64
		}{
			similarity: 0.0,
		}

		for _, buoyInfo := range buoyDataList {
			buoyLat, ok1 := buoyInfo.location["Latitude"].(float64)
			buoyLon, ok2 := buoyInfo.location["Longitude"].(float64)
			if !ok1 || !ok2 {
				continue
			}

			// Calculate travel time independently for this buoy
			var travelTimeHours float64
			targetBuoyTime := reportTime

			// Calculate bearing from buoy to spot
			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, spotLat, spotLon)

			// Check if spot is downwave from buoy
			swellTravelDirection := math.Mod(buoyInfo.waveDirection+180, 360) // Direction waves are traveling TO
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff // Handle wraparound
			}

			if angleDiff < 45.0 {
				// Spot is downwave - calculate travel time
				distance := s.calculateDistance(buoyLat, buoyLon, spotLat, spotLon)
				phaseVelocity := 1.56 * math.Sqrt(buoyInfo.period) // m/s
				travelTimeHours = distance / (phaseVelocity * 3600)

				// Cap travel time between 1-8 hours
				if travelTimeHours > 8.0 {
					travelTimeHours = 8.0
				}
				if travelTimeHours < 1.0 {
					travelTimeHours = 1.0
				}

				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			} else {
				// Spot is not directly downwave (targetBuoyTime already set to reportTime)
				travelTimeHours = 0.0
			}

			// Get historical buoy data at target time
			historicalBuoyData := s.getBuoyDataAtTime(ctx, targetBuoyTime, []string{buoyInfo.name})
			if historicalBuoyData == nil {
				continue // Skip if no buoy data found
			}

			// Calculate buoy similarity
			buoySimilarity := s.calculateBuoyConditionSimilarity(
				buoyInfo.waveHeight, buoyInfo.waveDirection, buoyInfo.period,
				historicalBuoyData,
			)

			// Track the best matching buoy
			if buoySimilarity > bestMatch.similarity {
				bestMatch.buoyName = buoyInfo.name
				bestMatch.similarity = buoySimilarity
				bestMatch.travelTime = travelTimeHours
				bestMatch.historicalData = historicalBuoyData
			}
		}

		// Only continue if buoy similarity is high enough
		if bestMatch.similarity < 0.7 {
			continue
		}

		// Get historical forecast data at report time
		historicalForecast := s.getForecastDataAtTime(ctx, countryName, regionName, spotName, reportTime)
		if historicalForecast == nil {
			// If no forecast data, skip wind matching but still include if buoy matches
			continue
		}

		// Extract historical wind data
		data, ok := historicalForecast["data"].(map[string]interface{})
		if !ok {
			continue
		}

		historicalWindSpeed := 0.0
		historicalWindDirection := 0.0

		switch v := data["windSpeed"].(type) {
		case float64:
			historicalWindSpeed = v
		case string:
			if ws, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				historicalWindSpeed = ws
			}
		}

		switch v := data["windDirection"].(type) {
		case float64:
			historicalWindDirection = v
		case string:
			if wd, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				historicalWindDirection = wd
			}
		}

		// Calculate wind similarity
		var windSimilarity float64
		if currentWindSpeed > 0 || currentWindDirection > 0 {
			windSimilarity = s.calculateWindSimilarity(
				currentWindSpeed, currentWindDirection,
				historicalWindSpeed, historicalWindDirection,
			)
		} else {
			// If we don't have current wind data, skip wind matching
			windSimilarity = 1.0 // Neutral score
		}

		// Only include if wind similarity is reasonable (>= 0.5)
		if windSimilarity < 0.5 {
			continue
		}

		// Calculate combined similarity (70% buoy, 30% wind)
		combinedSimilarity := 0.7*bestMatch.similarity + 0.3*windSimilarity

		// Remove sensitive fields
		delete(report, "UserEmail")

		// Add similarity scores and metadata
		report["buoy_similarity"] = bestMatch.similarity
		report["wind_similarity"] = windSimilarity
		report["combined_similarity"] = combinedSimilarity
		report["matched_buoy"] = bestMatch.buoyName
		report["historical_buoy_wave_height"] = bestMatch.historicalData["WaveHeight"]
		report["historical_buoy_wave_direction"] = bestMatch.historicalData["MeanWaveDirection"]
		report["historical_buoy_period"] = bestMatch.historicalData["MaxPeriod"]
		report["historical_wind_speed"] = historicalWindSpeed
		report["historical_wind_direction"] = historicalWindDirection
		if bestMatch.travelTime > 0 {
			report["travel_time_hours"] = bestMatch.travelTime
		}

		reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
			report:             report,
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
