package model

import "time"

type Weather struct {
	Timestamp     time.Time `json:"timestamp"`
	Location      string    `json:"location"`
	Country       string    `json:"country"`
	Region        string    `json:"region"`
	Spot          string    `json:"spot"`
	Conditions    string    `json:"conditions"`
	Temperature   float64   `json:"temperature"`
	Humidity      float64   `json:"humidity"`
	Pressure      float64   `json:"pressure"`
	WindSpeed     float64   `json:"wind_speed"`
	WindDirection float64   `json:"wind_direction"`
	Visibility    float64   `json:"visibility"`
}

type Tide struct {
	Date        time.Time `json:"date"`
	Time        time.Time `json:"time"`
	Location    string    `json:"location"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Height      float64   `json:"height"`
}
