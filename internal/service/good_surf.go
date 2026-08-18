package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
)

// EvaluateGoodSurf returns the earliest qualifying prediction in the next 24 hours
// that has not already been notified (deduped on arrival_time).
func EvaluateGoodSurf(predictions []model.SwellPrediction, now time.Time, lastNotifiedKey string) (bool, string, string) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	horizon := now.Add(goodSurfHorizon)

	var bestTime time.Time
	var bestKey, bestBody string
	found := false

	for _, prediction := range predictions {
		data := prediction.Data
		if data == nil {
			continue
		}
		quality := predictionFloat(data, "direction_quality", "directionQuality")
		size := predictionFloat(data, "surf_size", "surfSize")
		confidence := predictionFloat(data, "confidence")
		if quality < goodSurfDirectionQuality || size < goodSurfMinSize || confidence < goodSurfMinConfidence {
			continue
		}

		arrival, ok := parseArrival(data, prediction.ForecastTimestamp, now)
		if !ok {
			continue
		}
		if arrival.Before(now) || arrival.After(horizon) {
			continue
		}
		key := arrivalKey(data, prediction.ForecastTimestamp)
		if key == "" || key == lastNotifiedKey {
			continue
		}
		spot := spotNameFromID(prediction.SpotID)
		body := formatGoodSurfBody(spot, arrival)
		if !found || arrival.Before(bestTime) {
			found = true
			bestTime = arrival
			bestKey = key
			bestBody = body
		}
	}
	return found, bestKey, bestBody
}

func formatGoodSurfBody(spot string, arrival time.Time) string {
	if spot == "" {
		spot = "your spot"
	}
	return fmt.Sprintf("Good surf predicted at %s around %s", spot, formatAroundTime(arrival))
}

func formatAroundTime(t time.Time) string {
	loc, err := time.LoadLocation("Europe/Dublin")
	if err != nil {
		loc = time.UTC
	}
	local := t.In(loc)
	hour := local.Hour()
	ampm := "am"
	if hour >= 12 {
		ampm = "pm"
	}
	h12 := hour % 12
	if h12 == 0 {
		h12 = 12
	}
	return fmt.Sprintf("%d%s", h12, ampm)
}

func spotNameFromID(spotID string) string {
	parts := strings.Split(spotID, "#")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func arrivalKey(data map[string]interface{}, forecastTimestamp string) string {
	if raw := predictionString(data, "arrival_time", "arrivalTime"); raw != "" {
		return raw
	}
	return forecastTimestamp
}

func parseArrival(data map[string]interface{}, forecastTimestamp string, now time.Time) (time.Time, bool) {
	if raw := predictionString(data, "arrival_time", "arrivalTime"); raw != "" {
		if t, ok := parseTimeValue(raw); ok {
			return t, true
		}
	}
	if hoursAhead, ok := optionalPredictionFloat(data, "hours_ahead", "hoursAhead"); ok {
		return now.Add(time.Duration(hoursAhead * float64(time.Hour))), true
	}
	if forecastTimestamp != "" {
		if t, ok := parseTimeValue(forecastTimestamp); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

func parseTimeValue(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), true
		}
	}
	if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if unix > 1_000_000_000_000 {
			unix /= 1000
		}
		if unix > 0 {
			return time.Unix(unix, 0).UTC(), true
		}
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil && f > 0 {
		return time.Unix(int64(f), 0).UTC(), true
	}
	return time.Time{}, false
}

func predictionString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v
			}
		case fmt.Stringer:
			if s := v.String(); strings.TrimSpace(s) != "" {
				return s
			}
		default:
			s := fmt.Sprintf("%v", v)
			if strings.TrimSpace(s) != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func predictionFloat(data map[string]interface{}, keys ...string) float64 {
	value, ok := optionalPredictionFloat(data, keys...)
	if !ok {
		return 0
	}
	return value
}

func optionalPredictionFloat(data map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case string:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}
