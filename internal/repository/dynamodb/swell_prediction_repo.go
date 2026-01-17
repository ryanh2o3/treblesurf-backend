package dynamodb

import (
	"context"
	"fmt"
	"strconv"
	"time"

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
) ([]map[string]interface{}, error) {
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
) ([][]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 25
	}

	allPredictions := make([][]map[string]interface{}, 0, len(spotIDs))
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
) ([]map[string]interface{}, error) {
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
) ([]map[string]interface{}, error) {
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
) ([]map[string]interface{}, error) {
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
) (map[string]interface{}, error) {
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

	closest, err := findClosestPrediction(result.Items, now.Unix())
	if err != nil || closest == nil {
		return nil, fmt.Errorf("no valid AI predictions found for spot %s", spotID)
	}

	return closest, nil
}

func (r *SwellPredictionRepo) unmarshalPredictions(
	items []map[string]*dynamodb.AttributeValue,
) []map[string]interface{} {
	predictions := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		var prediction map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
			continue
		}

		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
				continue
			}

			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]
			predictions = append(predictions, dataMap)
		}
	}

	return predictions
}

func (r *SwellPredictionRepo) groupPredictionsBySpot(
	items []map[string]*dynamodb.AttributeValue,
	perSpotLimit int,
) []map[string]interface{} {
	if perSpotLimit <= 0 {
		perSpotLimit = 3
	}

	predictions := make([]map[string]interface{}, 0, len(items))
	spotPredictionsMap := make(map[string][]map[string]interface{})

	for _, item := range items {
		var prediction map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
			continue
		}

		spotID, ok := prediction["spot_id"].(string)
		if !ok {
			continue
		}

		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
				continue
			}

			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]

			if len(spotPredictionsMap[spotID]) < perSpotLimit {
				spotPredictionsMap[spotID] = append(spotPredictionsMap[spotID], dataMap)
			}
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
) (map[string]interface{}, error) {
	var closestPrediction map[string]interface{}
	var closestTimeDiff int64 = 999999999999

	for _, item := range items {
		var prediction map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
			continue
		}

		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
				continue
			}

			if forecastTimestampStr, ok := prediction["forecast_timestamp"].(string); ok {
				if forecastTimestamp, err := strconv.ParseInt(forecastTimestampStr, 10, 64); err == nil {
					timeDiff := abs(currentTimestamp - forecastTimestamp)
					if timeDiff < closestTimeDiff {
						closestTimeDiff = timeDiff
						closestPrediction = dataMap
						closestPrediction["spot_id"] = prediction["spot_id"]
						closestPrediction["forecast_timestamp"] = prediction["forecast_timestamp"]
						closestPrediction["generated_at"] = prediction["generated_at"]
					}
				}
			}
		}
	}

	return closestPrediction, nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
