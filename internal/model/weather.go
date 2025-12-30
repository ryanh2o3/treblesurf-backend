package model

import "time"

type Weather struct {
	Timestamp     time.Time `json:"timestamp" dynamodbav:"timestamp"`
	Location      string    `json:"location" dynamodbav:"location"`
	Country       string    `json:"country" dynamodbav:"country"`
	Region        string    `json:"region" dynamodbav:"region"`
	Spot          string    `json:"spot" dynamodbav:"spot"`
	Conditions    string    `json:"conditions" dynamodbav:"conditions"`
	Temperature   float64   `json:"temperature" dynamodbav:"temperature"`
	Humidity      float64   `json:"humidity" dynamodbav:"humidity"`
	Pressure      float64   `json:"pressure" dynamodbav:"pressure"`
	WindSpeed     float64   `json:"wind_speed" dynamodbav:"wind_speed"`
	WindDirection float64   `json:"wind_direction" dynamodbav:"wind_direction"`
	Visibility    float64   `json:"visibility" dynamodbav:"visibility"`
}

type Tide struct {
	Date        time.Time `json:"date" dynamodbav:"date"`
	Time        time.Time `json:"time" dynamodbav:"time"`
	Location    string    `json:"location" dynamodbav:"location"`
	Type        string    `json:"type" dynamodbav:"type"`
	Description string    `json:"description" dynamodbav:"description"`
	Height      float64   `json:"height" dynamodbav:"height"`
}
