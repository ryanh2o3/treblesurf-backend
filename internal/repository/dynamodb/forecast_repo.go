package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

func normalizeForecastQuerySource(source string) string {
	return strings.ToLower(strings.TrimSpace(source))
}

func forecastRowSource(f *model.Forecast) string {
	if f == nil || f.Data == nil {
		return ""
	}
	return normalizeForecastQuerySource(mapString(f.Data, "source", "forecast_source", "forecastSource"))
}

func forecastMatchesSource(forecast *model.Forecast, wantNormalized string) bool {
	if wantNormalized == "" {
		return true
	}
	got := forecastRowSource(forecast)
	if got == "" {
		return false
	}
	return got == wantNormalized
}

func mergeTopLevelSourceFromItem(item map[string]*dynamodb.AttributeValue, forecast *model.Forecast) {
	if forecast == nil {
		return
	}
	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		return
	}
	src := strings.TrimSpace(stringValue(raw["source"]))
	if src == "" {
		return
	}
	if forecast.Data == nil {
		forecast.Data = map[string]interface{}{}
	}
	if existing := mapString(forecast.Data, "source", "forecast_source", "forecastSource"); existing == "" {
		forecast.Data["source"] = src
	}
}

func (r *ForecastRepo) GetSpotForecast(
	ctx context.Context,
	country, region, spot, source string,
) ([]*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	currentEpoch := time.Now().Unix()
	wantSource := normalizeForecastQuerySource(source)
	log := logging.FromContext(ctx)

	baseInput := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":current_time": {
				S: aws.String(fmt.Sprintf("%d", currentEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}

	repoStart := time.Now()
	var (
		dynamoQueryDur time.Duration
		parseDur       time.Duration
		pages          int
		itemsRead      int
	)

	var forecasts []*model.Forecast
	var lastKey map[string]*dynamodb.AttributeValue
	for {
		pages++
		input := *baseInput
		if lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}
		tq := time.Now()
		result, err := r.client.QueryWithContext(ctx, &input)
		dynamoQueryDur += time.Since(tq)
		if err != nil {
			log.Warn("forecast repo GetSpotForecast dynamo query failed",
				slog.String("spot_id", spotID),
				slog.String("source_filter", source),
				slog.Int("dynamo_pages_completed", pages-1),
				slog.Duration("dynamo_query_total", dynamoQueryDur),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("querying spot forecast: %w", err)
		}
		itemsRead += len(result.Items)
		for _, item := range result.Items {
			tp := time.Now()
			forecast, err := parseForecastItem(item, country, region, spot)
			parseDur += time.Since(tp)
			if err != nil {
				return nil, err
			}
			if wantSource != "" && !forecastMatchesSource(forecast, wantSource) {
				continue
			}
			forecasts = append(forecasts, forecast)
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}

	log.Info("forecast repo GetSpotForecast timing",
		slog.String("spot_id", spotID),
		slog.String("source_filter", source),
		slog.Bool("source_filter_active", wantSource != ""),
		slog.Int("dynamo_pages", pages),
		slog.Int("dynamo_items_read", itemsRead),
		slog.Int("items_returned", len(forecasts)),
		slog.Duration("dynamo_query_total", dynamoQueryDur),
		slog.Duration("parse_and_filter_total", parseDur),
		slog.Duration("repo_total", time.Since(repoStart)),
	)

	return forecasts, nil
}

func (r *ForecastRepo) GetCurrentConditions(
	ctx context.Context,
	country, region, spot, source string,
) (*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	currentEpoch := time.Now().Unix()
	wantSource := normalizeForecastQuerySource(source)
	log := logging.FromContext(ctx)

	repoStart := time.Now()
	var (
		dynamoQueryDur time.Duration
		parseDur       time.Duration
		roundTrips     int
		itemsRead      int
	)

	var lastKey map[string]*dynamodb.AttributeValue
	for {
		roundTrips++
		input := &dynamodb.QueryInput{
			TableName:              aws.String(r.tableName),
			KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":spot_id": {
					S: aws.String(spotID),
				},
				":current_time": {
					S: aws.String(fmt.Sprintf("%d", currentEpoch)),
				},
			},
			ScanIndexForward: aws.Bool(true),
		}
		if lastKey != nil {
			input.ExclusiveStartKey = lastKey
		}
		if wantSource == "" {
			input.Limit = aws.Int64(1)
		}

		tq := time.Now()
		result, err := r.client.QueryWithContext(ctx, input)
		dynamoQueryDur += time.Since(tq)
		if err != nil {
			log.Warn("forecast repo GetCurrentConditions dynamo query failed",
				slog.String("spot_id", spotID),
				slog.String("source_filter", source),
				slog.Int("dynamo_round_trips", roundTrips),
				slog.Duration("dynamo_query_total", dynamoQueryDur),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("querying current conditions: %w", err)
		}
		itemsRead += len(result.Items)
		if len(result.Items) == 0 {
			if result.LastEvaluatedKey == nil {
				log.Info("forecast repo GetCurrentConditions timing",
					slog.String("spot_id", spotID),
					slog.String("source_filter", source),
					slog.Bool("source_filter_active", wantSource != ""),
					slog.Int("dynamo_round_trips", roundTrips),
					slog.Int("dynamo_items_read", itemsRead),
					slog.Bool("found", false),
					slog.Duration("dynamo_query_total", dynamoQueryDur),
					slog.Duration("parse_total", parseDur),
					slog.Duration("repo_total", time.Since(repoStart)),
				)
				return nil, repository.ErrNotFound
			}
			lastKey = result.LastEvaluatedKey
			continue
		}

		for _, item := range result.Items {
			tp := time.Now()
			forecast, err := parseForecastItem(item, country, region, spot)
			parseDur += time.Since(tp)
			if err != nil {
				return nil, err
			}
			if wantSource != "" && !forecastMatchesSource(forecast, wantSource) {
				continue
			}
			log.Info("forecast repo GetCurrentConditions timing",
				slog.String("spot_id", spotID),
				slog.String("source_filter", source),
				slog.Bool("source_filter_active", wantSource != ""),
				slog.Int("dynamo_round_trips", roundTrips),
				slog.Int("dynamo_items_read", itemsRead),
				slog.Bool("found", true),
				slog.Duration("dynamo_query_total", dynamoQueryDur),
				slog.Duration("parse_total", parseDur),
				slog.Duration("repo_total", time.Since(repoStart)),
			)
			return forecast, nil
		}

		if wantSource == "" || result.LastEvaluatedKey == nil {
			log.Info("forecast repo GetCurrentConditions timing",
				slog.String("spot_id", spotID),
				slog.String("source_filter", source),
				slog.Bool("source_filter_active", wantSource != ""),
				slog.Int("dynamo_round_trips", roundTrips),
				slog.Int("dynamo_items_read", itemsRead),
				slog.Bool("found", false),
				slog.Duration("dynamo_query_total", dynamoQueryDur),
				slog.Duration("parse_total", parseDur),
				slog.Duration("repo_total", time.Since(repoStart)),
			)
			return nil, repository.ErrNotFound
		}
		lastKey = result.LastEvaluatedKey
	}
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
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp >= :target_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":target_time": {
				S: aws.String(fmt.Sprintf("%d", targetEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(1),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast at time: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, repository.ErrNotFound
	}

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
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :since"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":since": {
				S: aws.String(fmt.Sprintf("%d", since.UTC().Unix())),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if limit > 0 {
		input.Limit = aws.Int64(int64(limit))
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data: %w", err)
	}

	return unmarshalForecastDataPoints(result.Items)
}

func (r *ForecastRepo) QueryBetween(
	ctx context.Context,
	spotID string,
	start, end time.Time,
	limit int,
) ([]*model.ForecastDataPoint, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":start": {
				S: aws.String(fmt.Sprintf("%d", start.UTC().Unix())),
			},
			":end": {
				S: aws.String(fmt.Sprintf("%d", end.UTC().Unix())),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if limit > 0 {
		input.Limit = aws.Int64(int64(limit))
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying forecast data range: %w", err)
	}

	return unmarshalForecastDataPoints(result.Items)
}

type forecastDataItem struct {
	Data              map[string]interface{} `dynamodbav:"data"`
	SpotID            string                 `dynamodbav:"spot_id"`
	ForecastTimestamp string                 `dynamodbav:"forecast_timestamp"`
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
		})
	}
	return forecasts, nil
}

func parseForecastTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	if unixSeconds, err := strconv.ParseInt(value, 10, 64); err == nil {
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
		out := flat.toModel()
		mergeTopLevelSourceFromItem(item, out)
		return out, nil
	}

	var raw map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &raw); err != nil {
		out := flat.toModel()
		mergeTopLevelSourceFromItem(item, out)
		return out, nil
	}

	dataMap := extractForecastData(item, raw)
	if len(dataMap) == 0 {
		out := flat.toModel()
		mergeTopLevelSourceFromItem(item, out)
		return out, nil
	}

	dataItem := forecastDataItem{
		Data:              dataMap,
		SpotID:            stringValue(raw["spot_id"]),
		ForecastTimestamp: stringValue(raw["forecast_timestamp"]),
	}

	out := forecastFromData(raw, dataItem, fallbackCountry, fallbackRegion, fallbackSpot)
	mergeTopLevelSourceFromItem(item, out)
	return out, nil
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
