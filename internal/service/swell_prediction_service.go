package service

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type SwellPredictionService struct {
	db *dynamodb.DynamoDB
}

func NewSwellPredictionService(db *dynamodb.DynamoDB) *SwellPredictionService {
	return &SwellPredictionService{db: db}
}

func (s *SwellPredictionService) GetSpotSwellPrediction(spotName, regionName, countryName string) ([]map[string]interface{}, error) {
	spotId := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotId),
			},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
		Limit:            aws.Int64(10),   // Get multiple predictions (up to 10)
	}

	result, err := s.db.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query swell predictions: %v", err)
	}

	var predictions []map[string]interface{}
	for _, item := range result.Items {
		var prediction map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &prediction)
		if err != nil {
			continue // Skip invalid items
		}
		
		// Extract the data field and add metadata
		if data, exists := prediction["data"]; exists {
			if dataMap, ok := data.(map[string]interface{}); ok {
				// Add metadata fields
				dataMap["spot_id"] = prediction["spot_id"]
				dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
				dataMap["generated_at"] = prediction["generated_at"]
				predictions = append(predictions, dataMap)
			}
		}
	}

	return predictions, nil
}

func (s *SwellPredictionService) GetListSpotsSwellPrediction(spots []string, regionName, countryName string) ([][]map[string]interface{}, error) {
	var allPredictions [][]map[string]interface{}
	
	for _, spot := range spots {
		predictions, err := s.GetSpotSwellPrediction(spot, regionName, countryName)
		if err != nil {
			return nil, fmt.Errorf("failed to get prediction for spot %s: %v", spot, err)
		}
		allPredictions = append(allPredictions, predictions)
	}
	
	return allPredictions, nil
}

func (s *SwellPredictionService) GetRegionSwellPrediction(regionName, countryName string) ([]map[string]interface{}, error) {
	// Query all spots in the region by scanning with filter
	input := &dynamodb.ScanInput{
		TableName: aws.String("SwellPredictions"),
		FilterExpression: aws.String("begins_with(spot_id, :region_prefix)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":region_prefix": {
				S: aws.String(fmt.Sprintf("%s#%s#", countryName, regionName)),
			},
		},
		Limit: aws.Int64(200), // Increased limit to get more predictions
	}

	result, err := s.db.Scan(input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan swell predictions: %v", err)
	}

	var predictions []map[string]interface{}
	spotPredictionsMap := make(map[string][]map[string]interface{}) // Track multiple predictions per spot
	
	for _, item := range result.Items {
		var prediction map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &prediction)
		if err != nil {
			continue
		}
		
		spotId := prediction["spot_id"].(string)
		
		if data, exists := prediction["data"]; exists {
			if dataMap, ok := data.(map[string]interface{}); ok {
				dataMap["spot_id"] = prediction["spot_id"]
				dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
				dataMap["generated_at"] = prediction["generated_at"]
				
				// Add to spot's predictions (limit to 3 per spot to avoid too much data)
				if len(spotPredictionsMap[spotId]) < 3 {
					spotPredictionsMap[spotId] = append(spotPredictionsMap[spotId], dataMap)
				}
			}
		}
	}
	
	// Convert map to slice and flatten
	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}
	
	return predictions, nil
}

func (s *SwellPredictionService) GetSpotSwellPredictionRange(spotName, regionName, countryName string, startTime, endTime time.Time) ([]map[string]interface{}, error) {
	spotId := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	startTimestamp := fmt.Sprintf("%d", startTime.Unix())
	endTimestamp := fmt.Sprintf("%d", endTime.Unix())
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotId),
			},
			":start": {
				S: aws.String(startTimestamp),
			},
			":end": {
				S: aws.String(endTimestamp),
			},
		},
		ScanIndexForward: aws.Bool(true), // Ascending order by time
	}

	result, err := s.db.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query swell predictions range: %v", err)
	}

	var predictions []map[string]interface{}
	for _, item := range result.Items {
		var prediction map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &prediction)
		if err != nil {
			continue
		}
		
		if data, exists := prediction["data"]; exists {
			if dataMap, ok := data.(map[string]interface{}); ok {
				dataMap["spot_id"] = prediction["spot_id"]
				dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
				dataMap["generated_at"] = prediction["generated_at"]
				predictions = append(predictions, dataMap)
			}
		}
	}

	return predictions, nil
}

func (s *SwellPredictionService) GetRecentSwellPredictions(hoursBack int) ([]map[string]interface{}, error) {
	cutoffTime := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	cutoffTimestamp := fmt.Sprintf("%d", cutoffTime.Unix())
	
	input := &dynamodb.ScanInput{
		TableName: aws.String("SwellPredictions"),
		FilterExpression: aws.String("generated_at >= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cutoff": {
				S: aws.String(cutoffTimestamp),
			},
		},
		Limit: aws.Int64(200), // Limit to prevent large scans
	}

	result, err := s.db.Scan(input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan recent swell predictions: %v", err)
	}

	var predictions []map[string]interface{}
	spotPredictionsMap := make(map[string][]map[string]interface{}) // Track multiple predictions per spot
	
	for _, item := range result.Items {
		var prediction map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &prediction)
		if err != nil {
			continue
		}
		
		spotId := prediction["spot_id"].(string)
		
		if data, exists := prediction["data"]; exists {
			if dataMap, ok := data.(map[string]interface{}); ok {
				dataMap["spot_id"] = prediction["spot_id"]
				dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
				dataMap["generated_at"] = prediction["generated_at"]
				
				// Add to spot's predictions (limit to 3 per spot to avoid too much data)
				if len(spotPredictionsMap[spotId]) < 3 {
					spotPredictionsMap[spotId] = append(spotPredictionsMap[spotId], dataMap)
				}
			}
		}
	}
	
	// Convert map to slice and flatten
	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}
	
	return predictions, nil
}
