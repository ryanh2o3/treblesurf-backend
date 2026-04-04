package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/logging"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var _ repository.ForecastRepository = (*ForecastRepo)(nil)

const (
	forecastSourceStormglass    = "stormglass"
	forecastSourceIrelandMerged = "imi_swan+weatherkit"
)

// defaultForecastPartitionSources matches what the forecaster writes (see treblesurf-forecaster app.py).
var defaultForecastPartitionSources = []string{
	forecastSourceStormglass,
	forecastSourceIrelandMerged,
}

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

func partitionSpotID(country, region, spot, source string) string {
	return fmt.Sprintf("%s#%s#%s#%s", country, region, spot, source)
}

func itemTimestampTS(item map[string]*dynamodb.AttributeValue) (int64, bool) {
	av := item["timestamp_ts"]
	if av == nil || av.N == nil {
		return 0, false
	}
	n, err := strconv.ParseInt(aws.StringValue(av.N), 10, 64)
	return n, err == nil
}

func itemSourceString(item map[string]*dynamodb.AttributeValue) string {
	if av := item["source"]; av != nil && av.S != nil {
		return aws.StringValue(av.S)
	}
	return ""
}

func sortForecastItems(items []map[string]*dynamodb.AttributeValue) {
	sort.Slice(items, func(i, j int) bool {
		ti, _ := itemTimestampTS(items[i])
		tj, _ := itemTimestampTS(items[j])
		if ti != tj {
			return ti < tj
		}
		return itemSourceString(items[i]) < itemSourceString(items[j])
	})
}

func (r *ForecastRepo) queryPartitionRange(
	ctx context.Context,
	partitionKey string,
	startInclusive, endInclusive int64,
) ([]map[string]*dynamodb.AttributeValue, error) {
	var all []map[string]*dynamodb.AttributeValue
	var eks map[string]*dynamodb.AttributeValue
	for {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.tableName),
			KeyConditionExpression: aws.String("spot_id = :pk AND timestamp_ts BETWEEN :start AND :end"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":pk": {
					S: aws.String(partitionKey),
				},
				":start": {
					N: aws.String(strconv.FormatInt(startInclusive, 10)),
				},
				":end": {
					N: aws.String(strconv.FormatInt(endInclusive, 10)),
				},
			},
			ScanIndexForward: aws.Bool(true),
		}
		if len(eks) > 0 {
			input.ExclusiveStartKey = eks
		}
		result, err := r.client.QueryWithContext(ctx, input)
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if len(result.LastEvaluatedKey) == 0 {
			break
		}
		eks = result.LastEvaluatedKey
	}
	return all, nil
}

func (r *ForecastRepo) queryMergedSpotPartitions(
	ctx context.Context,
	country, region, spot string,
	startEpoch, endEpoch int64,
) ([]map[string]*dynamodb.AttributeValue, error) {
	var merged []map[string]*dynamodb.AttributeValue
	for _, src := range defaultForecastPartitionSources {
		pk := partitionSpotID(country, region, spot, src)
		items, err := r.queryPartitionRange(ctx, pk, startEpoch, endEpoch)
		if err != nil {
			return nil, err
		}
		merged = append(merged, items...)
	}
	sortForecastItems(merged)
	return merged, nil
}

func (r *ForecastRepo) GetSpotForecast(
	ctx context.Context,
	country, region, spot string,
) ([]*model.Forecast, error) {
	now := time.Now()
	startEpoch := now.Unix()
	endEpoch := now.Add(7 * 24 * time.Hour).Unix()

	dynamoStart := time.Now()
	items, err := r.queryMergedSpotPartitions(ctx, country, region, spot, startEpoch, endEpoch)
	dynamoElapsed := time.Since(dynamoStart)
	if err != nil {
		return nil, fmt.Errorf("querying spot forecast: %w", err)
	}

	parseStart := time.Now()
	forecasts := make([]*model.Forecast, 0, len(items))
	for _, item := range items {
		forecast, err := parseForecastItem(item, country, region, spot)
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, forecast)
	}
	parseElapsed := time.Since(parseStart)

	baseID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	logging.FromContext(ctx).Info("forecast timing: dynamodb get_spot_forecast",
		slog.String("spot_id", baseID),
		slog.Int64("dynamo_query_ms", dynamoElapsed.Milliseconds()),
		slog.Int64("parse_items_ms", parseElapsed.Milliseconds()),
		slog.Int("dynamo_item_count", len(items)),
		slog.Bool("dynamo_result_truncated", false),
		slog.Int64("range_start_epoch", startEpoch),
		slog.Int64("range_end_epoch", endEpoch),
	)

	return forecasts, nil
}

func (r *ForecastRepo) GetCurrentConditions(
	ctx context.Context,
	country, region, spot string,
) (*model.Forecast, error) {
	now := time.Now()
	startEpoch := now.Unix()
	endEpoch := now.Add(48 * time.Hour).Unix()

	items, err := r.queryMergedSpotPartitions(ctx, country, region, spot, startEpoch, endEpoch)
	if err != nil {
		return nil, fmt.Errorf("querying current conditions: %w", err)
	}
	if len(items) == 0 {
		return nil, repository.ErrNotFound
	}
	if len(items) > 100 {
		items = items[:100]
	}

	forecast, err := parseForecastItem(items[0], country, region, spot)
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
	targetEpoch := t.Unix()

	items, err := r.queryMergedSpotPartitions(ctx, country, region, spot, targetEpoch, targetEpoch)
	if err != nil {
		return nil, fmt.Errorf("querying forecast at time: %w", err)
	}
	if len(items) == 0 {
		return nil, repository.ErrNotFound
	}

	forecast, err := parseForecastItem(items[0], country, region, spot)
	if err != nil {
		return nil, err
	}

	return forecast, nil
}

func (r *ForecastRepo) QuerySince(
	ctx context.Context,
	spotID string,
	since time.Time,
	limit int,
) ([]*model.ForecastDataPoint, error) {
	country, region, spot, ok := splitBaseSpotID(spotID)
	if !ok {
		return nil, fmt.Errorf("invalid spot_id for forecast query: %q", spotID)
	}
	sinceEpoch := since.UTC().Unix()
	endEpoch := since.Add(30 * 24 * time.Hour).Unix()

	items, err := r.queryMergedSpotPartitions(ctx, country, region, spot, sinceEpoch, endEpoch)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data: %w", err)
	}

	points, err := unmarshalForecastDataPoints(items, spotID)
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
	country, region, spot, ok := splitBaseSpotID(spotID)
	if !ok {
		return nil, fmt.Errorf("invalid spot_id for forecast query: %q", spotID)
	}
	startEpoch := start.UTC().Unix()
	endEpoch := end.UTC().Unix()

	items, err := r.queryMergedSpotPartitions(ctx, country, region, spot, startEpoch, endEpoch)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data range: %w", err)
	}

	points, err := unmarshalForecastDataPoints(items, spotID)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(points) > limit {
		points = points[:limit]
	}
	return points, nil
}

func splitBaseSpotID(spotID string) (country, region, spot string, ok bool) {
	parts := strings.SplitN(spotID, "#", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

type forecastDataItem struct {
	Data        map[string]interface{} `dynamodbav:"data"`
	SpotID      string                 `dynamodbav:"spot_id"`
	TimestampTs int64                  `dynamodbav:"timestamp_ts"`
	Source      string                 `dynamodbav:"source"`
	GeneratedAt string                 `dynamodbav:"generated_at"`
}

func (item *forecastDataItem) forecastTimestampString() string {
	if item == nil || item.TimestampTs == 0 {
		return ""
	}
	return strconv.FormatInt(item.TimestampTs, 10)
}

func unmarshalForecastDataPoints(items []map[string]*dynamodb.AttributeValue, baseSpotID string) ([]*model.ForecastDataPoint, error) {
	forecasts := make([]*model.ForecastDataPoint, 0, len(items))
	for _, item := range items {
		var forecastItem forecastDataItem
		if err := dynamodbattribute.UnmarshalMap(item, &forecastItem); err != nil {
			return nil, fmt.Errorf("unmarshaling forecast data: %w", err)
		}
		forecastTime := time.Unix(forecastItem.TimestampTs, 0).UTC()
		forecasts = append(forecasts, &model.ForecastDataPoint{
			SpotID:            baseSpotID,
			ForecastTimestamp: forecastTime,
			Data:              forecastItem.Data,
			Source:            forecastItem.Source,
			GeneratedAt:       forecastItem.GeneratedAt,
		})
	}
	return forecasts, nil
}

// parseForecastTimestamp parses a legacy string sort key or decimal epoch string.
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
		Data:        dataMap,
		SpotID:      stringValue(raw["spot_id"]),
		TimestampTs: int64FromInterface(raw["timestamp_ts"]),
		Source:      stringValue(raw["source"]),
		GeneratedAt: stringValue(raw["generated_at"]),
	}

	return forecastFromData(raw, dataItem, fallbackCountry, fallbackRegion, fallbackSpot), nil
}

func int64FromInterface(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case float32:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	case json.Number:
		n, _ := x.Int64()
		return n
	default:
		return 0
	}
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

	baseSpotID := fmt.Sprintf("%s#%s#%s", country, region, spot)

	countryRegionSpot := stringValue(raw["country_region_spot"])
	if countryRegionSpot == "" && country != "" && region != "" && spot != "" {
		countryRegionSpot = fmt.Sprintf("%s_%s_%s", country, region, spot)
	}

	dateForecastedFor := mapString(item.Data, "dateForecastedFor", "date_forecasted_for")
	forecastDate := stringValue(raw["ForecastDate"])
	if forecastDate == "" {
		forecastDate = mapString(item.Data, "forecast_date", "ForecastDate")
	}

	ftStr := item.forecastTimestampString()
	date := parseForecastDate(mapString(item.Data, "date"), dateForecastedFor, ftStr)
	hour := mapInt(item.Data, "hour")
	if hour == 0 && !date.IsZero() {
		hour = date.Hour()
	}

	return &model.Forecast{
		ForecastTimestamp: ftStr,
		SpotID:            baseSpotID,
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

// splitSpotID parses partition spot_id (country#region#spot or country#region#spot#source).
func splitSpotID(spotID string) (country, region, spot string, ok bool) {
	parts := strings.Split(spotID, "#")
	if len(parts) < 3 {
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
