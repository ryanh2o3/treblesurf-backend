package model

import "time"

type Forecast struct {
	Date              time.Time              `json:"date"`
	Data              map[string]interface{} `json:"data,omitempty"`
	Region            string                 `json:"region"`
	ForecastDate      string                 `json:"forecast_date"`
	Location          string                 `json:"location"`
	DateForecastedFor string                 `json:"dateForecastedFor"`
	Spot              string                 `json:"spot"`
	Country           string                 `json:"country"`
	Conditions        string                 `json:"conditions"`
	ForecastTimestamp string                 `json:"forecast_timestamp,omitempty"`
	SpotID            string                 `json:"spot_id,omitempty"`
	Source            string                 `json:"source,omitempty"`
	GeneratedAt       string                 `json:"generated_at,omitempty"`
	CountryRegionSpot string                 `json:"country_region_spot"`
	WaveDirection     float64                `json:"wave_direction"`
	Temperature       float64                `json:"temperature"`
	WindDirection     float64                `json:"wind_direction"`
	WindSpeed         float64                `json:"wind_speed"`
	MaxPeriod         float64                `json:"max_period"`
	WavePeriod        float64                `json:"wave_period"`
	WaveHeight        float64                `json:"wave_height"`
	Hour              int                    `json:"hour"`
}
