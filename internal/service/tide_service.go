package service

import (
	"time"
)

type TideService struct {
	// Add any dependencies here if needed
}

// NewTideService creates a new tide service instance.
func NewTideService() *TideService {
	return &TideService{}
}

// GetCurrentTides returns current tide information for the specified location.
func (s *TideService) GetCurrentTides(locationName string) []map[string]interface{} {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	queryToday := s.getTides(locationName, today)
	queryYesterday := s.getTides(locationName, yesterday)
	queryTomorrow := s.getTides(locationName, tomorrow)

	// Collect all tide data into a slice
	var result []map[string]interface{}
	
	// Add yesterday's tides if they exist
	if queryYesterday != nil {
		if tides, ok := queryYesterday["tides"].([]map[string]interface{}); ok {
			result = append(result, tides...)
		}
	}
	
	// Add today's tides if they exist
	if queryToday != nil {
		if tides, ok := queryToday["tides"].([]map[string]interface{}); ok {
			result = append(result, tides...)
		}
	}
	
	// Add tomorrow's tides if they exist
	if queryTomorrow != nil {
		if tides, ok := queryTomorrow["tides"].([]map[string]interface{}); ok {
			result = append(result, tides...)
		}
	}

	return result
}

// GetBeforeAfterTides returns the previous and next tide information for the specified location.
func (s *TideService) GetBeforeAfterTides(locationName string) (map[string]interface{}, map[string]interface{}) {
	tides := s.GetCurrentTides(locationName)
	now := time.Now()

	var prevTide, nextTide map[string]interface{}
	for _, tide := range tides {
		// Parse the tide time string
		tideTimeStr, ok := tide["time"].(string)
		if !ok {
			continue // Skip if time field is not a string
		}
		
		tideTime, err := time.Parse("2006-01-02 15:04:05", tideTimeStr)
		if err != nil {
			continue // Skip if time parsing fails
		}
		
		// Find the most recent tide before now
		if tideTime.Before(now) {
			if prevTide == nil {
				prevTide = tide
			} else {
				// Parse the previous tide time for comparison
				prevTimeStr, ok := prevTide["time"].(string)
				if ok {
					if prevTime, err := time.Parse("2006-01-02 15:04:05", prevTimeStr); err == nil {
						if tideTime.After(prevTime) {
							prevTide = tide
						}
					}
				}
			}
		}
		
		// Find the earliest tide after now
		if tideTime.After(now) {
			if nextTide == nil {
				nextTide = tide
			} else {
				// Parse the next tide time for comparison
				nextTimeStr, ok := nextTide["time"].(string)
				if ok {
					if nextTime, err := time.Parse("2006-01-02 15:04:05", nextTimeStr); err == nil {
						if tideTime.Before(nextTime) {
							nextTide = tide
						}
					}
				}
			}
		}
	}

	return prevTide, nextTide
}

// GetDayTides returns tide information for a specific day at the specified location.
func (s *TideService) GetDayTides(locationName, startDay string) map[string]interface{} {
	start, _ := time.Parse("2006-01-02", startDay)
	end := start.AddDate(0, 0, 10)

	tideData := make(map[string]interface{})
	for start.Before(end) {
		day := start.Format("2006-01-02")
		tides := s.getTides(locationName, day)
		if tides != nil {
			tideData[day] = tides
		} else {
			tideData[day] = map[string]interface{}{
				"location": locationName,
				"date":     day,
				"tides":    []map[string]interface{}{},
			}
		}
		start = start.AddDate(0, 0, 1)
	}

	return tideData
}

func (s *TideService) getTides(locationName, date string) map[string]interface{} {
	// TODO: Implement the logic to get tides from DynamoDB
	// This is a placeholder implementation that returns sample tide data
	// In a real implementation, this would query DynamoDB for tide information
	
	// Generate sample tide times for the given date
	sampleTides := []map[string]interface{}{
		{
			"time":        date + " 06:00:00",
			"height":      2.5,
			"type":        "high",
			"location":    locationName,
		},
		{
			"time":        date + " 12:00:00",
			"height":      0.5,
			"type":        "low",
			"location":    locationName,
		},
		{
			"time":        date + " 18:00:00",
			"height":      2.8,
			"type":        "high",
			"location":    locationName,
		},
		{
			"time":        date + " 23:30:00",
			"height":      0.3,
			"type":        "low",
			"location":    locationName,
		},
	}
	
	return map[string]interface{}{
		"location": locationName,
		"date":     date,
		"tides":    sampleTides,
	}
}


