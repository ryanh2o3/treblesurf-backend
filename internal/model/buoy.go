package model

import "time"

type Buoy struct {
	RegionBuoy string  `json:"region_buoy"`
	Name       string  `json:"name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

type BuoyData struct {
	Timestamp        time.Time `json:"timestamp"`
	BuoyName         string    `json:"buoy_name"`
	WaveHeight       float64   `json:"wave_height"`
	WavePeriod       float64   `json:"wave_period"`
	MaxPeriod        float64   `json:"max_period"`
	WaveDirection    float64   `json:"wave_direction"`
	WindSpeed        float64   `json:"wind_speed"`
	WindDirection    float64   `json:"wind_direction"`
	Temperature      float64   `json:"temperature"`
	Pressure         float64   `json:"pressure"`
	SprTp            float64   `json:"spr_tp"`
	ThTp             float64   `json:"th_tp"`
	MaxHeight        float64   `json:"max_height"`
	Gust             float64   `json:"gust"`
	AirTemperature   float64   `json:"air_temperature"`
	DewPoint         float64   `json:"dew_point"`
	RelativeHumidity float64   `json:"relative_humidity"`
	Salinity         float64   `json:"salinity"`
}

type BuoyLocation struct {
	Name      string  `json:"name"`
	RegionBuoy string `json:"region_buoy,omitempty"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	Spot      string  `json:"spot"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
