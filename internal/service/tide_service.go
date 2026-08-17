package service

import (
	"fmt"
	"log/slog"
	"time"

	"treblesurf-backend/internal/config"
)

type TideService struct {
	isDevelopment bool
}

func NewTideService(cfg *config.Config) (*TideService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	return &TideService{isDevelopment: cfg.IsDevelopment()}, nil
}

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

func (s *TideService) GetBeforeAfterTides(locationName string) (prevTide, nextTide map[string]interface{}) {
	tides := s.GetCurrentTides(locationName)
	now := time.Now()
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

func (s *TideService) GetDayTides(locationName, startDay string) map[string]interface{} {
	start, err := time.Parse("2006-01-02", startDay)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("invalid date format: %v", err)}
	}
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
	if !s.isDevelopment {
		slog.Warn("tide data not configured; returning empty response",
			slog.String("location", locationName),
			slog.String("date", date),
		)
		return nil
	}
	// Development-only placeholder data until DynamoDB integration is available.

	// Generate sample tide times for the given date
	sampleTides := []map[string]interface{}{
		{
			"time":     date + " 06:00:00",
			"height":   2.5,
			"type":     "high",
			"location": locationName,
		},
		{
			"time":     date + " 12:00:00",
			"height":   0.5,
			"type":     "low",
			"location": locationName,
		},
		{
			"time":     date + " 18:00:00",
			"height":   2.8,
			"type":     "high",
			"location": locationName,
		},
		{
			"time":     date + " 23:30:00",
			"height":   0.3,
			"type":     "low",
			"location": locationName,
		},
	}

	return map[string]interface{}{
		"location": locationName,
		"date":     date,
		"tides":    sampleTides,
	}
}
