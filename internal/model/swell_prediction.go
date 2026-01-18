package model

import (
	"encoding/json"
	"strconv"
)

// SwellPrediction represents a swell prediction with inline data fields.
type SwellPrediction struct {
	Data              map[string]interface{} `json:"-"`
	SpotID            string                 `json:"spot_id"`
	ForecastTimestamp string                 `json:"forecast_timestamp"`
	GeneratedAt       string                 `json:"generated_at,omitempty"`
}

// MarshalJSON flattens the Data map with the base fields for backward-compatible responses.
func (p SwellPrediction) MarshalJSON() ([]byte, error) {
	payload := make(map[string]interface{}, len(p.Data)+3)
	for key, value := range p.Data {
		payload[key] = value
	}
	if p.SpotID != "" {
		payload["spot_id"] = p.SpotID
	}
	if p.ForecastTimestamp != "" {
		payload["forecast_timestamp"] = p.ForecastTimestamp
	}
	if p.GeneratedAt != "" {
		payload["generated_at"] = p.GeneratedAt
	}
	return json.Marshal(payload)
}

// ForecastTimestampValue converts the forecast timestamp to an int64 for sorting.
func (p SwellPrediction) ForecastTimestampValue() int64 {
	value, err := strconv.ParseInt(p.ForecastTimestamp, 10, 64)
	if err != nil {
		return 0
	}
	return value
}
