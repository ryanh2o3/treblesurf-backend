package model

type Location struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LocationInfo struct {
	IdealSwellDirection string  `json:"IdealSwellDirection"`
	Image               string  `json:"Image"`
	Type                string  `json:"Type"`
	CountryRegionSpot   string  `json:"country_region_spot"`
	ImageString         string  `json:"ImageString,omitempty"`
	ImageURL            string  `json:"image_url,omitempty"`
	BeachDirection      int     `json:"BeachDirection"`
	Elevation           int     `json:"Elevation"`
	Latitude            float64 `json:"Latitude"`
	Longitude           float64 `json:"Longitude"`
}
