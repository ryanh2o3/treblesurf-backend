package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.ForecastRepository = (*ForecastRepo)(nil)

type ForecastRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewForecastRepo(client *dynamodb.DynamoDB, tableName string) *ForecastRepo {
	return &ForecastRepo{
		client:    client,
		tableName: tableName,
	}
}

// forecastRangeEnd returns the sort key value that includes all keys with the given
// Unix timestamp (including any "{ts}#{source}" suffixes). Lexicographic order ensures
// "{ts}" < "{ts}#stormglass" < "{ts}#weatherkit" < "{ts+1}".
func forecastRangeEnd(endUnix int64) string {
	return fmt.Sprintf("%d~", endUnix)
}

func (r *ForecastRepo) GetSpotForecast(
	ctx context.Context,
	country, region, spot string,
) ([]*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	now := time.Now()
	startEpoch := now.Unix()
	endEpoch := now.Add(7 * 24 * time.Hour).Unix()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":start": {
				S: aws.String(fmt.Sprintf("%d", startEpoch)),
			},
			":end": {
				S: aws.String(forecastRangeEnd(endEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying spot forecast: %w", err)
	}

	forecasts := make([]*model.Forecast, 0, len(result.Items))
	for _, item := range result.Items {
		forecast, err := parseForecastItem(item, country, region, spot)
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

func (r *ForecastRepo) GetCurrentConditions(
	ctx context.Context,
	country, region, spot string,
) (*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	now := time.Now()
	startEpoch := now.Unix()
	endEpoch := now.Add(48 * time.Hour).Unix()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":start": {
				S: aws.String(fmt.Sprintf("%d", startEpoch)),
			},
			":end": {
				S: aws.String(forecastRangeEnd(endEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(100),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying current conditions: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, repository.ErrNotFound
	}

	// Return first item (primary source selection happens in service when grouping)
	forecast, err := parseForecastItem(result.Items[0], country, region, spot)
	if err != nil {
		return nil, err
	}

	return forecast, nil
}

func (r *ForecastRepo) GetForecastAtTime(
	ctx context.Context,
	country, region, spot string,
	t time.Time,
) (*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	targetEpoch := t.Unix()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :target AND :target_end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":target": {
				S: aws.String(fmt.Sprintf("%d", targetEpoch)),
			},
			":target_end": {
				S: aws.String(forecastRangeEnd(targetEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(10),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast at time: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, repository.ErrNotFound
	}

	// Return first item (primary source selection in service if needed)
	forecast, err := parseForecastItem(result.Items[0], country, region, spot)
	if err != nil {
		return nil, err
	}

	return forecast, nil
}

func (r *ForecastRepo) GetRegionForecast(
	ctx context.Context,
	country, region string,
	forecastDate time.Time,
) ([]*model.Forecast, error) {
	dateKey := forecastDate.Format("2006-01-02")
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("ForecastDate = :date AND begins_with(country_region_spot, :location)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":date": {
				S: aws.String(dateKey),
			},
			":location": {
				S: aws.String(fmt.Sprintf("%s_%s_", country, region)),
			},
		},
		ScanIndexForward: aws.Bool(false),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying region forecast: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, repository.ErrNotFound
	}

	forecasts := make([]*model.Forecast, 0, len(result.Items))
	for _, item := range result.Items {
		forecast, err := parseForecastItem(item, country, region, "")
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, forecast)
	}

	return forecasts, nil
}

func (r *ForecastRepo) QuerySince(
	ctx context.Context,
	spotID string,
	since time.Time,
	limit int,
) ([]*model.ForecastDataPoint, error) {
	sinceEpoch := since.UTC().Unix()
	endEpoch := since.Add(30 * 24 * time.Hour).Unix()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :since AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":since": {
				S: aws.String(fmt.Sprintf("%d", sinceEpoch)),
			},
			":end": {
				S: aws.String(forecastRangeEnd(endEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if limit > 0 {
		input.Limit = aws.Int64(int64(limit * 10)) // fetch extra in case of multiple sources per time
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data: %w", err)
	}

	points, err := unmarshalForecastDataPoints(result.Items)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(points) > limit {
		points = points[:limit]
	}
	return points, nil
}

func (r *ForecastRepo) QueryBetween(
	ctx context.Context,
	spotID string,
	start, end time.Time,
	limit int,
) ([]*model.ForecastDataPoint, error) {
	startEpoch := start.UTC().Unix()
	endEpoch := end.UTC().Unix()

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":start": {
				S: aws.String(fmt.Sprintf("%d", startEpoch)),
			},
			":end": {
				S: aws.String(forecastRangeEnd(endEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if limit > 0 {
		input.Limit = aws.Int64(int64(limit * 10))
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data range: %w", err)
	}

	points, err := unmarshalForecastDataPoints(result.Items)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(points) > limit {
		points = points[:limit]
	}
	return points, nil
}

type forecastDataItem struct {
	Data              map[string]interface{} `dynamodbav:"data"`
	SpotID            string                 `dynamodbav:"spot_id"`
	ForecastTimestamp string                 `dynamodbav:"forecast_timestamp"`
	Source            string                 `dynamodbav:"source"`
	GeneratedAt       string                 `dynamodbav:"generated_at"`
}

func unmarshalForecastDataPoints(items []map[string]*dynamodb.AttributeValue) ([]*model.ForecastDataPoint, error) {
	forecasts := make([]*model.ForecastDataPoint, 0, len(items))
	for _, item := range items {
		var forecastItem forecastDataItem
		if err := dynamodbattribute.UnmarshalMap(item, &forecastItem); err != nil {
			return nil, fmt.Errorf("unmarshaling forecast data: %w", err)
		}
		forecastTime, err := parseForecastTimestamp(forecastItem.ForecastTimestamp)
		if err != nil {
			return nil, fmt.Errorf("parsing forecast timestamp: %w", err)
		}
		forecasts = append(forecasts, &model.ForecastDataPoint{
			SpotID:            forecastItem.SpotID,
			ForecastTimestamp: forecastTime,
			Data:              forecastItem.Data,
			Source:            forecastItem.Source,
			GeneratedAt:       forecastItem.GeneratedAt,
		})
	}
	return forecasts, nil
}

// parseForecastTimestamp parses the sort key value, which may be "{unix_ts}" or "{unix_ts}#{source}".
func parseForecastTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	base := value
	if idx := strings.Index(value, "#"); idx >= 0 {
		base = value[:idx]
	}
	if unixSeconds, err := strconv.ParseInt(base, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC(), nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognized forecast timestamp: %s", value)
}

func parseForecastItem(
	item map[string]*dynamodb.AttributeValue,
	fallbackCountry, fallbackRegion, fallbackSpot string,
) (*model.Forecast, error) {
	var flat forecastItem
	if err := dynamodbattribute.UnmarshalMap(item, &flat); err != nil {
		return nil, fmt.Errorf("unmarshaling forecast: %w", err)
	}
	if flat.isPopulated() {
		return flat.toModel(), nil
	}

	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		return flat.toModel(), nil
	}

	dataMap := extractForecastData(item, raw)
	if len(dataMap) == 0 {
		return flat.toModel(), nil
	}

	dataItem := forecastDataItem{
		Data:              dataMap,
		SpotID:            stringValue(raw["spot_id"]),
		ForecastTimestamp: stringValue(raw["forecast_timestamp"]),
		Source:            stringValue(raw["source"]),
		GeneratedAt:       stringValue(raw["generated_at"]),
	}

	return forecastFromData(raw, dataItem, fallbackCountry, fallbackRegion, fallbackSpot), nil
}

func extractForecastData(
	item map[string]*dynamodb.AttributeValue,
	raw map[string]interface{},
) map[string]interface{} {
	for _, key := range []string{"data", "Data"} {
		if attr, ok := item[key]; ok {
			var data map[string]interface{}
			if err := dynamodbattribute.Unmarshal(attr, &data); err == nil && len(data) > 0 {
				return data
			}
		}
	}

	for _, key := range []string{"data", "Data"} {
		if value, ok := raw[key]; ok {
			switch v := value.(type) {
			case map[string]interface{}:
				if len(v) > 0 {
					return v
				}
			case string:
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(v), &data); err == nil && len(data) > 0 {
					return data
				}
			}
		}
	}

	return nil
}

func (f *forecastItem) isPopulated() bool {
	return f.hasTextFields() || f.hasDateFields() || f.hasNumericFields()
}

func (f *forecastItem) hasTextFields() bool {
	return f.CountryRegionSpot != "" ||
		f.ForecastDate != "" ||
		f.Conditions != "" ||
		f.Country != "" ||
		f.Region != "" ||
		f.Spot != "" ||
		f.DateForecastedFor != "" ||
		f.Location != ""
}

func (f *forecastItem) hasDateFields() bool {
	return !f.Date.IsZero()
}

func (f *forecastItem) hasNumericFields() bool {
	return f.Hour != 0 ||
		f.WindSpeed != 0 ||
		f.WindDirection != 0 ||
		f.WaveHeight != 0 ||
		f.WavePeriod != 0 ||
		f.MaxPeriod != 0 ||
		f.WaveDirection != 0 ||
		f.Temperature != 0
}

func forecastFromData(
	raw map[string]interface{},
	item forecastDataItem,
	fallbackCountry, fallbackRegion, fallbackSpot string,
) *model.Forecast {
	country := fallbackCountry
	region := fallbackRegion
	spot := fallbackSpot

	if spotID := stringValue(raw["spot_id"]); spotID != "" {
		if parsedCountry, parsedRegion, parsedSpot, ok := splitSpotID(spotID); ok {
			country = chooseString(country, parsedCountry)
			region = chooseString(region, parsedRegion)
			spot = chooseString(spot, parsedSpot)
		}
	}

	countryRegionSpot := stringValue(raw["country_region_spot"])
	if countryRegionSpot == "" && country != "" && region != "" && spot != "" {
		countryRegionSpot = fmt.Sprintf("%s_%s_%s", country, region, spot)
	}

	dateForecastedFor := mapString(item.Data, "dateForecastedFor", "date_forecasted_for")
	forecastDate := stringValue(raw["ForecastDate"])
	if forecastDate == "" {
		forecastDate = mapString(item.Data, "forecast_date", "ForecastDate")
	}

	date := parseForecastDate(mapString(item.Data, "date"), dateForecastedFor, item.ForecastTimestamp)
	hour := mapInt(item.Data, "hour")
	if hour == 0 && !date.IsZero() {
		hour = date.Hour()
	}

	return &model.Forecast{
		ForecastTimestamp: item.ForecastTimestamp,
		SpotID:            chooseString(stringValue(raw["spot_id"]), item.SpotID),
		Source:            item.Source,
		GeneratedAt:       chooseString(item.GeneratedAt, mapString(item.Data, "generated_at", "generatedAt")),
		CountryRegionSpot: countryRegionSpot,
		ForecastDate:      chooseString(forecastDate, date.Format("2006-01-02")),
		Date:              date,
		Conditions:        mapString(item.Data, "conditions", "surfMessiness"),
		Country:           country,
		Region:            region,
		Spot:              spot,
		DateForecastedFor: dateForecastedFor,
		Location:          mapString(item.Data, "location"),
		Hour:              hour,
		WindSpeed:         mapFloat(item.Data, "wind_speed", "windSpeed"),
		WindDirection:     mapFloat(item.Data, "wind_direction", "windDirection"),
		WaveHeight:        mapFloat(item.Data, "wave_height", "waveHeight", "swellHeight"),
		WavePeriod:        mapFloat(item.Data, "wave_period", "wavePeriod", "swellPeriod"),
		MaxPeriod:         mapFloat(item.Data, "max_period", "maxPeriod"),
		WaveDirection:     mapFloat(item.Data, "wave_direction", "waveDirection", "swellDirection"),
		Temperature:       mapFloat(item.Data, "temperature", "waterTemperature"),
		Data:              item.Data,
	}
}

func parseForecastDate(dateValue, dateForecastedFor, forecastTimestamp string) time.Time {
	if parsed := parseTimeValue(dateValue); !parsed.IsZero() {
		return parsed
	}
	if parsed := parseTimeValue(dateForecastedFor); !parsed.IsZero() {
		return parsed
	}
	if forecastTimestamp != "" {
		if parsed, err := parseForecastTimestamp(forecastTimestamp); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseTimeValue(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed.UTC()
	}
	if unixSeconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(unixSeconds, 0).UTC()
	}
	return time.Time{}
}

func splitSpotID(spotID string) (country, region, spot string, ok bool) {
	parts := strings.SplitN(spotID, "#", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func chooseString(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func mapString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str := stringValue(value); str != "" {
				return str
			}
		}
	}
	return ""
}

func mapFloat(data map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if out, ok := floatValue(value); ok {
				return out
			}
		}
	}
	return 0
}

func mapInt(data map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if out, ok := intValue(value); ok {
				return out
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
	case string:
		if v == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func intValue(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		if v == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
