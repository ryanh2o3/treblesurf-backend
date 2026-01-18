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

var _ repository.SwellPredictionRepository = (*SwellPredictionRepo)(nil)

type SwellPredictionRepo struct {
	client    *dynamodb.DynamoDB
	tableName string
}

func NewSwellPredictionRepo(client *dynamodb.DynamoDB, tableName string) *SwellPredictionRepo {
	return &SwellPredictionRepo{
		client:    client,
		tableName: tableName,
	}
}

func (r *SwellPredictionRepo) GetSpotPredictions(
	ctx context.Context,
	spotID string,
	start time.Time,
	limit int,
) ([]model.SwellPrediction, error) {
	if limit <= 0 {
		limit = 25
	}

	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp >= :current_hour"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":current_hour": {
				S: aws.String(fmt.Sprintf("%d", start.UTC().Unix())),
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(int64(limit)),
	}

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying swell predictions: %w", err)
	}

	return r.unmarshalPredictions(result.Items), nil
}

func (r *SwellPredictionRepo) GetListSpotsPredictions(
	ctx context.Context,
	spotIDs []string,
	start time.Time,
	limit int,
) ([][]model.SwellPrediction, error) {
	if limit <= 0 {
		limit = 25
	}

	allPredictions := make([][]model.SwellPrediction, 0, len(spotIDs))
	for _, spotID := range spotIDs {
		predictions, err := r.GetSpotPredictions(ctx, spotID, start, limit)
		if err != nil {
			return nil, err
		}
		allPredictions = append(allPredictions, predictions)
	}

	return allPredictions, nil
}

func (r *SwellPredictionRepo) GetRegionPredictions(
	ctx context.Context,
	country, region string,
	start time.Time,
	perSpotLimit int,
) ([]model.SwellPrediction, error) {
	if perSpotLimit <= 0 {
		perSpotLimit = 3
	}

	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		FilterExpression: aws.String("begins_with(spot_id, :region_prefix) AND forecast_timestamp >= :current_hour"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":region_prefix": {
				S: aws.String(fmt.Sprintf("%s#%s#", country, region)),
			},
			":current_hour": {
				S: aws.String(fmt.Sprintf("%d", start.UTC().Unix())),
			},
		},
		Limit: aws.Int64(500),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning swell predictions: %w", err)
	}

	return r.groupPredictionsBySpot(result.Items, perSpotLimit), nil
}

func (r *SwellPredictionRepo) GetSpotPredictionRange(
	ctx context.Context,
	spotID string,
	start, end time.Time,
) ([]model.SwellPrediction, error) {
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

	result, err := r.client.QueryWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("querying swell prediction range: %w", err)
	}

	return r.unmarshalPredictions(result.Items), nil
}

func (r *SwellPredictionRepo) GetRecentPredictions(
	ctx context.Context,
	cutoff time.Time,
	perSpotLimit int,
) ([]model.SwellPrediction, error) {
	if perSpotLimit <= 0 {
		perSpotLimit = 3
	}

	input := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
		FilterExpression: aws.String("generated_at >= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cutoff": {
				S: aws.String(fmt.Sprintf("%d", cutoff.UTC().Unix())),
			},
		},
		Limit: aws.Int64(200),
	}

	result, err := r.client.ScanWithContext(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("scanning recent swell predictions: %w", err)
	}

	return r.groupPredictionsBySpot(result.Items, perSpotLimit), nil
}

func (r *SwellPredictionRepo) GetClosestPrediction(
	ctx context.Context,
	spotID string,
	now time.Time,
) (*model.SwellPrediction, error) {
	startTime := now.Add(-12 * time.Hour)
	endTime := now.Add(48 * time.Hour)

	result, err := r.client.QueryWithContext(ctx, buildPredictionQuery(spotID, startTime, endTime))
	if err != nil {
		return nil, fmt.Errorf("querying closest AI prediction: %w", err)
	}

	if len(result.Items) == 0 {
		fallbackStartTime := now.Add(-7 * 24 * time.Hour)
		fallbackResult, queryErr := r.client.QueryWithContext(ctx, buildFallbackPredictionQuery(spotID, fallbackStartTime))
		if queryErr != nil {
			return nil, fmt.Errorf("querying fallback AI prediction: %w", queryErr)
		}
		if len(fallbackResult.Items) == 0 {
			return nil, fmt.Errorf("no AI predictions found for spot %s", spotID)
		}
		result = fallbackResult
	}

	closest := findClosestPrediction(result.Items, now.Unix())
	if closest == nil {
		return nil, fmt.Errorf("no valid AI predictions found for spot %s", spotID)
	}

	return closest, nil
}

func (r *SwellPredictionRepo) unmarshalPredictions(
	items []map[string]*dynamodb.AttributeValue,
) []model.SwellPrediction {
	predictions := make([]model.SwellPrediction, 0, len(items))
	for _, item := range items {
		prediction, ok := r.extractPrediction(item)
		if !ok {
			continue
		}
		predictions = append(predictions, prediction)
	}

	return predictions
}

func (r *SwellPredictionRepo) groupPredictionsBySpot(
	items []map[string]*dynamodb.AttributeValue,
	perSpotLimit int,
) []model.SwellPrediction {
	if perSpotLimit <= 0 {
		perSpotLimit = 3
	}

	predictions := make([]model.SwellPrediction, 0, len(items))
	spotPredictionsMap := make(map[string][]model.SwellPrediction)

	for _, item := range items {
		prediction, ok := r.extractPrediction(item)
		if !ok {
			continue
		}

		spotID := prediction.SpotID
		if spotID == "" {
			continue
		}

		if len(spotPredictionsMap[spotID]) < perSpotLimit {
			spotPredictionsMap[spotID] = append(spotPredictionsMap[spotID], prediction)
		}
	}

	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}

	return predictions
}

func buildPredictionQuery(spotID string, startTime, endTime time.Time) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {S: aws.String(spotID)},
			":start":   {S: aws.String(fmt.Sprintf("%d", startTime.Unix()))},
			":end":     {S: aws.String(fmt.Sprintf("%d", endTime.Unix()))},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(100),
	}
}

func buildFallbackPredictionQuery(spotID string, startTime time.Time) *dynamodb.QueryInput {
	return &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp >= :start"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {S: aws.String(spotID)},
			":start":   {S: aws.String(fmt.Sprintf("%d", startTime.Unix()))},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(50),
	}
}

func findClosestPrediction(
	items []map[string]*dynamodb.AttributeValue,
	currentTimestamp int64,
) *model.SwellPrediction {
	var closestPrediction *model.SwellPrediction
	var closestTimeDiff int64 = 999999999999

	for _, item := range items {
		prediction, ok := extractPrediction(item)
		if !ok {
			continue
		}

		forecastTimestamp, err := strconv.ParseInt(prediction.ForecastTimestamp, 10, 64)
		if err != nil {
			continue
		}
		timeDiff := abs(currentTimestamp - forecastTimestamp)
		if timeDiff < closestTimeDiff {
			closestTimeDiff = timeDiff
			predictionCopy := prediction
			closestPrediction = &predictionCopy
			}
	}

	return closestPrediction
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (r *SwellPredictionRepo) extractPrediction(
	item map[string]*dynamodb.AttributeValue,
) (model.SwellPrediction, bool) {
	var prediction map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
		return model.SwellPrediction{}, false
	}
	dataAttr, exists := item["data"]
	if !exists {
		return model.SwellPrediction{}, false
	}

	var dataMap map[string]interface{}
	if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
		return model.SwellPrediction{}, false
	}

	return model.SwellPrediction{
		SpotID:            stringValue(prediction["spot_id"]),
		ForecastTimestamp: stringValue(prediction["forecast_timestamp"]),
		GeneratedAt:       stringValue(prediction["generated_at"]),
		Data:              dataMap,
	}, true
}

func extractPrediction(
	item map[string]*dynamodb.AttributeValue,
) (model.SwellPrediction, bool) {
	var prediction map[string]interface{}
	if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
		return model.SwellPrediction{}, false
	}
	dataAttr, exists := item["data"]
	if !exists {
		return model.SwellPrediction{}, false
	}

	var dataMap map[string]interface{}
	if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
		return model.SwellPrediction{}, false
	}

	return model.SwellPrediction{
		SpotID:            stringValue(prediction["spot_id"]),
		ForecastTimestamp: stringValue(prediction["forecast_timestamp"]),
		GeneratedAt:       stringValue(prediction["generated_at"]),
		Data:              dataMap,
	}, true
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case *string:
		if v == nil {
			return ""
		}
		return *v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
