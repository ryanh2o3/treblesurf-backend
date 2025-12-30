package model

import "time"

type Forecast struct {
	Date              time.Time `json:"date" dynamodbav:"date"`
	Conditions        string    `json:"conditions" dynamodbav:"conditions"`
	Country           string    `json:"country" dynamodbav:"country"`
	Region            string    `json:"region" dynamodbav:"region"`
	Spot              string    `json:"spot" dynamodbav:"spot"`
	DateForecastedFor string    `json:"dateForecastedFor" dynamodbav:"dateForecastedFor"`
	Location          string    `json:"location" dynamodbav:"location"`
	Hour              int       `json:"hour" dynamodbav:"hour"`
	WindSpeed         float64   `json:"wind_speed" dynamodbav:"wind_speed"`
	WindDirection     float64   `json:"wind_direction" dynamodbav:"wind_direction"`
	WaveHeight        float64   `json:"wave_height" dynamodbav:"wave_height"`
	WavePeriod        float64   `json:"wave_period" dynamodbav:"wave_period"`
	MaxPeriod         float64   `json:"max_period" dynamodbav:"max_period"`
	WaveDirection     float64   `json:"wave_direction" dynamodbav:"wave_direction"`
	Temperature       float64   `json:"temperature" dynamodbav:"temperature"`
}
