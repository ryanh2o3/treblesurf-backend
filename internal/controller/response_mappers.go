package controller

import (
	"fmt"
	"strconv"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"
)

type clientForecastData struct {
	DateForecastedFor     string  `json:"dateForecastedFor"`
	SurfMessiness         string  `json:"surfMessiness"`
	RelativeWindDirection string  `json:"relativeWindDirection"`
	Temperature           float64 `json:"temperature"`
	SurfSize              float64 `json:"surfSize"`
	SwellDirection        float64 `json:"swellDirection"`
	SwellHeight           float64 `json:"swellHeight"`
	SwellPeriod           float64 `json:"swellPeriod"`
	DirectionQuality      float64 `json:"directionQuality"`
	WaterTemperature      float64 `json:"waterTemperature"`
	WaveEnergy            float64 `json:"waveEnergy"`
	WindDirection         float64 `json:"windDirection"`
	WindSpeed             float64 `json:"windSpeed"`
	Pressure              float64 `json:"pressure"`
	Precipitation         float64 `json:"precipitation"`
	Humidity              float64 `json:"humidity"`
}

type clientForecastResponse struct {
	ForecastTimestamp string             `json:"forecast_timestamp"`
	GeneratedAt       string             `json:"generated_at"`
	SpotID            string             `json:"spot_id"`
	Source            string             `json:"source,omitempty"`
	Data              clientForecastData `json:"data"`
}

// clientForecastGroupResponse is the multi-source shape (Option A: time + sources map).
type clientForecastGroupResponse struct {
	Sources           map[string]clientForecastData `json:"sources"`
	Time              string                        `json:"time"`
	ForecastTimestamp string                        `json:"forecast_timestamp"`
	PrimarySource     string                        `json:"primary_source"`
}

type clientBuoyResponse struct {
	AirTemperature      *float64 `json:"AirTemperature,omitempty"`
	AtmosphericPressure *float64 `json:"AtmosphericPressure,omitempty"`
	DewPoint            *float64 `json:"DewPoint,omitempty"`
	Gust                *float64 `json:"Gust,omitempty"`
	MaxHeight           *float64 `json:"MaxHeight,omitempty"`
	MaxPeriod           *float64 `json:"MaxPeriod,omitempty"`
	MeanWaveDirection   *float64 `json:"MeanWaveDirection,omitempty"`
	RelativeHumidity    *float64 `json:"RelativeHumidity,omitempty"`
	Salinity            *float64 `json:"Salinity,omitempty"`
	SeaTemperature      *float64 `json:"SeaTemperature,omitempty"`
	SprTp               *float64 `json:"SprTp,omitempty"`
	ThTp                *float64 `json:"ThTp,omitempty"`
	WaveHeight          *float64 `json:"WaveHeight,omitempty"`
	WavePeriod          *float64 `json:"WavePeriod,omitempty"`
	WindDirection       *float64 `json:"WindDirection,omitempty"`
	WindSpeed           *float64 `json:"WindSpeed,omitempty"`
	DataDateTime        string   `json:"dataDateTime"`
	Name                string   `json:"name"`
	RegionBuoy          string   `json:"region_buoy"`
}

func mapForecastsToClient(forecasts []*model.Forecast) []clientForecastResponse {
	if len(forecasts) == 0 {
		return []clientForecastResponse{}
	}
	results := make([]clientForecastResponse, 0, len(forecasts))
	for _, forecast := range forecasts {
		if forecast == nil {
			continue
		}
		results = append(results, mapForecastToClient(forecast))
	}
	return results
}

func mapForecastGroupsToClient(groups [][]*model.Forecast) [][]clientForecastResponse {
	if len(groups) == 0 {
		return [][]clientForecastResponse{}
	}
	results := make([][]clientForecastResponse, 0, len(groups))
	for _, group := range groups {
		results = append(results, mapForecastsToClient(group))
	}
	return results
}

func mapForecastToClient(forecast *model.Forecast) clientForecastResponse {
	spotID := forecast.SpotID
	if spotID == "" && forecast.Country != "" && forecast.Region != "" && forecast.Spot != "" {
		spotID = fmt.Sprintf("%s#%s#%s", forecast.Country, forecast.Region, forecast.Spot)
	}

	generatedAt := stringFromData(forecast.Data, "generated_at", "generatedAt")
	if generatedAt == "" {
		generatedAt = forecast.GeneratedAt
	}
	if generatedAt == "" && !forecast.Date.IsZero() {
		generatedAt = forecast.Date.UTC().Format(time.RFC3339)
	}

	forecastTimestamp := forecast.ForecastTimestamp
	if forecastTimestamp == "" && !forecast.Date.IsZero() {
		forecastTimestamp = fmt.Sprintf("%d", forecast.Date.Unix())
	}

	return clientForecastResponse{
		Source: forecast.Source,
		Data: clientForecastData{
			DateForecastedFor: firstString(
				stringFromData(forecast.Data, "dateForecastedFor", "date_forecasted_for"),
				forecast.DateForecastedFor,
			),
			DirectionQuality:      floatFromData(forecast.Data, "directionQuality", "direction_quality"),
			Humidity:              floatFromData(forecast.Data, "humidity"),
			Precipitation:         floatFromData(forecast.Data, "precipitation"),
			Pressure:              floatFromData(forecast.Data, "pressure"),
			RelativeWindDirection: stringFromData(forecast.Data, "relativeWindDirection", "relative_wind_direction"),
			SurfMessiness:         firstString(stringFromData(forecast.Data, "surfMessiness", "surf_messiness", "conditions"), forecast.Conditions),
			SurfSize:              floatFromData(forecast.Data, "surfSize", "surf_size"),
			SwellDirection:        floatFromData(forecast.Data, "swellDirection", "swell_direction", "wave_direction"),
			SwellHeight:           floatFromData(forecast.Data, "swellHeight", "swell_height", "wave_height"),
			SwellPeriod:           floatFromData(forecast.Data, "swellPeriod", "swell_period", "wave_period"),
			Temperature:           floatFromData(forecast.Data, "temperature"),
			WaterTemperature:      floatFromData(forecast.Data, "waterTemperature", "water_temperature"),
			WaveEnergy:            floatFromData(forecast.Data, "waveEnergy", "wave_energy"),
			WindDirection:         floatFromData(forecast.Data, "windDirection", "wind_direction"),
			WindSpeed:             floatFromData(forecast.Data, "windSpeed", "wind_speed"),
		},
		ForecastTimestamp: forecastTimestamp,
		GeneratedAt:       generatedAt,
		SpotID:            spotID,
	}
}

// mapForecastGroupToClient maps a service ForecastGroup to the multi-source API shape.
func mapForecastGroupToClient(g service.ForecastGroup) clientForecastGroupResponse {
	sources := make(map[string]clientForecastData)
	for name, f := range g.Sources {
		if f != nil {
			sources[name] = mapForecastToClient(f).Data
		}
	}
	timeStr := g.ForecastTimestamp
	if !g.Time.IsZero() {
		timeStr = g.Time.UTC().Format(time.RFC3339)
	}
	return clientForecastGroupResponse{
		Time:              timeStr,
		ForecastTimestamp: g.ForecastTimestamp,
		PrimarySource:     g.PrimarySource,
		Sources:           sources,
	}
}

func mapForecastGroupsToClientResponse(groups []service.ForecastGroup) []clientForecastGroupResponse {
	if len(groups) == 0 {
		return []clientForecastGroupResponse{}
	}
	out := make([]clientForecastGroupResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, mapForecastGroupToClient(g))
	}
	return out
}

func mapSpotReportsToClient(reports []map[string]interface{}) []map[string]interface{} {
	if len(reports) == 0 {
		return []map[string]interface{}{}
	}
	out := make([]map[string]interface{}, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		out = append(out, mapSpotReportToClient(report))
	}
	return out
}

func mapSpotReportToClient(report map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(report))

	copyIfExists := func(keys ...string) {
		for _, key := range keys {
			if value, ok := report[key]; ok {
				out[key] = value
			}
		}
	}

	out["Consistency"] = stringFromReport(report, "consistency", "Consistency")
	out["ImageKey"] = stringFromReport(report, "image_key", "ImageKey")
	out["VideoKey"] = stringFromReport(report, "video_key", "VideoKey")
	out["Messiness"] = stringFromReport(report, "messiness", "Messiness")
	out["Quality"] = stringFromReport(report, "quality", "Quality")
	out["Reporter"] = stringFromReport(report, "reporter", "Reporter")
	out["SurfSize"] = stringFromReport(report, "surf_size", "SurfSize")
	out["Time"] = stringFromReport(report, "time", "Time")
	out["WindAmount"] = stringFromReport(report, "wind_amount", "WindAmount")
	out["WindDirection"] = stringFromReport(report, "wind_direction", "WindDirection")
	out["country_region_spot"] = stringFromReport(report, "country_region_spot", "country_region_spot")
	out["dateReported"] = stringFromReport(report, "date_reported", "dateReported")
	out["MediaType"] = stringFromReport(report, "media_type", "MediaType")
	out["IOSValidated"] = boolFromReport(report, "ios_validated", "IOSValidated")

	// Similarity fields are expected in snake_case by the iOS app.
	copyIfExists(
		"buoy_similarity",
		"wind_similarity",
		"combined_similarity",
		"matched_buoy",
		"historical_buoy_wave_height",
		"historical_buoy_wave_direction",
		"historical_buoy_period",
		"historical_wind_speed",
		"historical_wind_direction",
		"travel_time_hours",
	)

	return out
}

func stringFromReport(report map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := report[key]; ok {
			if str := stringValue(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func boolFromReport(report map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if value, ok := report[key]; ok {
			switch v := value.(type) {
			case bool:
				return v
			case string:
				if parsed, err := strconv.ParseBool(v); err == nil {
					return parsed
				}
			case float64:
				return v != 0
			case int:
				return v != 0
			}
		}
	}
	return false
}

func stringFromData(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if data == nil {
			return ""
		}
		if value, ok := data[key]; ok {
			if str := stringValue(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func firstString(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func floatFromData(data map[string]interface{}, keys ...string) float64 {
	if data == nil {
		return 0
	}
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if parsed, ok := floatValue(value); ok {
				return parsed
			}
		}
	}
	return 0
}

func floatValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case jsonNumber:
		if parsed, err := v.Float64(); err == nil {
			return parsed, true
		}
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

type jsonNumber interface {
	Float64() (float64, error)
	String() string
}

func floatPtr(value float64) *float64 {
	return &value
}
