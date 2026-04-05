package repository

import "fmt"

// DynamoDB surf_forecasts partition key suffix (country#region#spot#source#<granularity>).
const (
	ForecastGranularityHourly    = "hourly"
	ForecastGranularityMultiHour = "multiHour"
)

// ParseForecastGranularity normalizes the API query value. Empty defaults to multiHour.
func ParseForecastGranularity(s string) (string, error) {
	if s == "" {
		return ForecastGranularityMultiHour, nil
	}
	if s == ForecastGranularityHourly || s == ForecastGranularityMultiHour {
		return s, nil
	}
	return "", fmt.Errorf("invalid granularity %q: use %q or %q", s, ForecastGranularityHourly, ForecastGranularityMultiHour)
}
