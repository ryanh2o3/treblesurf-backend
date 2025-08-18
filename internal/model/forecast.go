package model

import "time"

// Forecast represents weather forecast data
type Forecast struct {
	Location      string    `json:"location" dynamodbav:"location"`
	Country       string    `json:"country" dynamodbav:"country"`
	Region        string    `json:"region" dynamodbav:"region"`
	Spot          string    `json:"spot" dynamodbav:"spot"`
	Date          time.Time `json:"date" dynamodbav:"date"`
	DateForecastedFor string `json:"dateForecastedFor" dynamodbav:"dateForecastedFor"`
	Hour          int       `json:"hour" dynamodbav:"hour"`
	Temperature   float64   `json:"temperature" dynamodbav:"temperature"`
	WindSpeed     float64   `json:"wind_speed" dynamodbav:"wind_speed"`
	WindDirection float64   `json:"wind_direction" dynamodbav:"wind_direction"`
	WaveHeight    float64   `json:"wave_height" dynamodbav:"wave_height"`
	WavePeriod    float64   `json:"wave_period" dynamodbav:"wave_period"`
	WaveDirection float64   `json:"wave_direction" dynamodbav:"wave_direction"`
	Conditions    string    `json:"conditions" dynamodbav:"conditions"`
}