package model

// Location represents a geographic location with coordinates.
type Location struct {
    Name      string  `json:"name"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

// LocationInfo represents detailed information about a surf location.
type LocationInfo struct {
    BeachDirection      int     `json:"BeachDirection"`
    Elevation           int     `json:"Elevation"`
    IdealSwellDirection string  `json:"IdealSwellDirection"`
    Image               string  `json:"Image"`
    Latitude            float64 `json:"Latitude"`
    Longitude           float64 `json:"Longitude"`
    Type                string  `json:"Type"`
    CountryRegionSpot   string  `json:"country_region_spot"`
    ImageString        string  `json:"ImageString"`
}