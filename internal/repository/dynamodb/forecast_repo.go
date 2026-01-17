package dynamodb

import (
	"context"
	"fmt"
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
		var forecast model.Forecast
		if err := dynamodbattribute.UnmarshalMap(item, &forecast); err != nil {
			return nil, fmt.Errorf("unmarshaling forecast: %w", err)
		}
		forecasts = append(forecasts, &forecast)
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

	var forecast model.Forecast
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &forecast); err != nil {
		return nil, fmt.Errorf("unmarshaling forecast: %w", err)
	}

	return &forecast, nil
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

	var forecast model.Forecast
	if err := dynamodbattribute.UnmarshalMap(result.Items[0], &forecast); err != nil {
		return nil, fmt.Errorf("unmarshaling forecast: %w", err)
	}

	return &forecast, nil
}
