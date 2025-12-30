package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

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

//nolint:unparam // Error return maintained for API consistency
func findClosestPrediction(
	items []map[string]*dynamodb.AttributeValue,
	currentTimestamp int64,
	_ string,
) (map[string]interface{}, error) {
	var closestPrediction map[string]interface{}
	var closestTimeDiff int64 = 999999999999
	validItems := 0

	for i, item := range items {
		var prediction map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &prediction); err != nil {
			fmt.Printf("Failed to unmarshal item %d: %v\n", i, err)
			continue
		}

		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			if err := dynamodbattribute.Unmarshal(dataAttr, &dataMap); err != nil {
				fmt.Printf("Failed to unmarshal data field for item %d: %v\n", i, err)
				continue
			}

			if forecastTimestampStr, ok := prediction["forecast_timestamp"].(string); ok {
				if forecastTimestamp, err := strconv.ParseInt(forecastTimestampStr, 10, 64); err == nil {
					timeDiff := abs(currentTimestamp - forecastTimestamp)
					validItems++
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

	fmt.Printf("Valid items processed: %d\n", validItems)
	return closestPrediction, nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

