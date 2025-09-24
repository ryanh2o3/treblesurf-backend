package model

import "time"

// Buoy represents a buoy in the system
type Buoy struct {
	RegionBuoy string `json:"region_buoy"`
	Latitude    float64   `json:"latitude" dynamodbav:"latitude"`
	Longitude   float64   `json:"longitude" dynamodbav:"longitude"`
	Name string `json:"name"`
}

// BuoyData represents buoy measurement data
type BuoyData struct {
	BuoyName     string    `json:"buoy_name" dynamodbav:"buoy_name"`
	Timestamp    time.Time `json:"timestamp" dynamodbav:"timestamp"`
	WaveHeight   float64   `json:"wave_height" dynamodbav:"wave_height"`
	WavePeriod   float64   `json:"wave_period" dynamodbav:"wave_period"`
	MaxPeriod    float64   `json:"max_period" dynamodbav:"max_period"`
	WaveDirection float64  `json:"wave_direction" dynamodbav:"wave_direction"`
	WindSpeed    float64   `json:"wind_speed" dynamodbav:"wind_speed"`
	WindDirection float64  `json:"wind_direction" dynamodbav:"wind_direction"`
	Temperature  float64   `json:"temperature" dynamodbav:"temperature"`
	Pressure     float64   `json:"pressure" dynamodbav:"pressure"`
}

// BuoyLocation represents buoy location information
type BuoyLocation struct {
	Name      string  `json:"name" dynamodbav:"name"`
	Country   string  `json:"country" dynamodbav:"country"`
	Region    string  `json:"region" dynamodbav:"region"`
	Spot      string  `json:"spot" dynamodbav:"spot"`
	Latitude  float64 `json:"latitude" dynamodbav:"latitude"`
	Longitude float64 `json:"longitude" dynamodbav:"longitude"`
}
