package model

import "time"

type Forecast struct {
	CountryRegionSpot string    `json:"country_region_spot"`
	ForecastDate      string    `json:"forecast_date"`
	Date              time.Time `json:"date"`
	Conditions        string    `json:"conditions"`
	Country           string    `json:"country"`
	Region            string    `json:"region"`
	Spot              string    `json:"spot"`
	DateForecastedFor string    `json:"dateForecastedFor"`
	Location          string    `json:"location"`
	Hour              int       `json:"hour"`
	WindSpeed         float64   `json:"wind_speed"`
	WindDirection     float64   `json:"wind_direction"`
	WaveHeight        float64   `json:"wave_height"`
	WavePeriod        float64   `json:"wave_period"`
	MaxPeriod         float64   `json:"max_period"`
	WaveDirection     float64   `json:"wave_direction"`
	Temperature       float64   `json:"temperature"`
}
