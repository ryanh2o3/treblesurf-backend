package model

import "time"

// ForecastDataPoint represents a raw forecast snapshot with its timestamp and payload.
// This keeps repository shapes consistent across storage backends.
type ForecastDataPoint struct {
	SpotID            string                 `json:"spot_id"`
	ForecastTimestamp time.Time              `json:"forecast_timestamp"`
	Data              map[string]interface{} `json:"data"`
}
