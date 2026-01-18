package dynamodb

import (
	"context"
	"fmt"
	"strconv"
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

func (r *ForecastRepo) GetSpotForecast(
	ctx context.Context,
	country, region, spot string,
) ([]*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	currentEpoch := time.Now().Unix()

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

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying spot forecast: %w", err)
	}

	forecasts := make([]*model.Forecast, 0, len(result.Items))
	for _, item := range result.Items {
		var forecastRecord forecastItem
		if err := dynamodbattribute.UnmarshalMap(item, &forecastRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling forecast: %w", err)
		}
		forecasts = append(forecasts, forecastRecord.toModel())
	}

	return forecasts, nil
}

func (r *ForecastRepo) GetCurrentConditions(
	ctx context.Context,
	country, region, spot string,
) (*model.Forecast, error) {
	spotID := fmt.Sprintf("%s#%s#%s", country, region, spot)
	currentEpoch := time.Now().Unix()

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
		Limit:            aws.Int64(1),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying current conditions: %w", err)
	}
	if len(result.Items) == 0 {
		return nil, model.ErrForecastNotFound
	}

	var forecastRecord forecastItem
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &forecastRecord); err != nil {
		return nil, fmt.Errorf("unmarshaling forecast: %w", err)
	}

	return forecastRecord.toModel(), nil
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
		return nil, model.ErrForecastNotFound
	}

	var forecastRecord forecastItem
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &forecastRecord); err != nil {
		return nil, fmt.Errorf("unmarshaling forecast: %w", err)
	}

	return forecastRecord.toModel(), nil
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
		return nil, nil
	}

	forecasts := make([]*model.Forecast, 0, len(result.Items))
	for _, item := range result.Items {
		var forecastRecord forecastItem
		if err := dynamodbattribute.UnmarshalMap(item, &forecastRecord); err != nil {
			return nil, fmt.Errorf("unmarshaling forecast: %w", err)
		}
		forecasts = append(forecasts, forecastRecord.toModel())
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
