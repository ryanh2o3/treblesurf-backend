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


func (s *SwellPredictionService) GetSpotSwellPrediction(
	spotName, regionName, countryName string,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	
	// Get current time rounded to the hour (UTC)
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	currentHourTimestamp := fmt.Sprintf("%d", currentHour.Unix())
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp >= :current_hour"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":current_hour": {
				S: aws.String(currentHourTimestamp),
			},
		},
		ScanIndexForward: aws.Bool(true), // Ascending order (earliest first)
		Limit:            aws.Int64(25),  // Get up to 25 hours of predictions
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
		
		// Extract the data field and properly unmarshal it
		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			err := dynamodbattribute.Unmarshal(dataAttr, &dataMap)
			if err != nil {
				continue // Skip invalid data
			}
			
			// Add metadata fields
			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]
			predictions = append(predictions, dataMap)
		}
	}

	return predictions, nil
}

func (s *SwellPredictionService) GetListSpotsSwellPrediction(
	spots []string, regionName, countryName string,
) ([][]map[string]interface{}, error) {
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

func (s *SwellPredictionService) GetRegionSwellPrediction(
	regionName, countryName string,
) ([]map[string]interface{}, error) {
	// Get current time rounded to the hour (UTC)
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	currentHourTimestamp := fmt.Sprintf("%d", currentHour.Unix())
	
	// Query all spots in the region by scanning with filter
	input := &dynamodb.ScanInput{
		TableName: aws.String("SwellPredictions"),
		FilterExpression: aws.String("begins_with(spot_id, :region_prefix) AND forecast_timestamp >= :current_hour"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":region_prefix": {
				S: aws.String(fmt.Sprintf("%s#%s#", countryName, regionName)),
			},
			":current_hour": {
				S: aws.String(currentHourTimestamp),
			},
		},
		Limit: aws.Int64(500), // Increased limit to get more predictions
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
		
		spotID := prediction["spot_id"].(string)
		
		// Extract the data field and properly unmarshal it
		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			err := dynamodbattribute.Unmarshal(dataAttr, &dataMap)
			if err != nil {
				continue // Skip invalid data
			}
			
			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]
			
			// Add to spot's predictions (limit to 3 per spot to avoid too much data)
			if len(spotPredictionsMap[spotID]) < 3 {
				spotPredictionsMap[spotID] = append(spotPredictionsMap[spotID], dataMap)
			}
		}
	}
	
	// Convert map to slice and flatten
	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}
	
	return predictions, nil
}

func (s *SwellPredictionService) GetSpotSwellPredictionRange(
	spotName, regionName, countryName string,
	startTime, endTime time.Time,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	startTimestamp := fmt.Sprintf("%d", startTime.Unix())
	endTimestamp := fmt.Sprintf("%d", endTime.Unix())
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start AND :end"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
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
		
		// Extract the data field and properly unmarshal it
		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			err := dynamodbattribute.Unmarshal(dataAttr, &dataMap)
			if err != nil {
				continue // Skip invalid data
			}
			
			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]
			predictions = append(predictions, dataMap)
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
		
		spotID := prediction["spot_id"].(string)
		
		// Extract the data field and properly unmarshal it
		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			err := dynamodbattribute.Unmarshal(dataAttr, &dataMap)
			if err != nil {
				continue // Skip invalid data
			}
			
			dataMap["spot_id"] = prediction["spot_id"]
			dataMap["forecast_timestamp"] = prediction["forecast_timestamp"]
			dataMap["generated_at"] = prediction["generated_at"]
			
			// Add to spot's predictions (limit to 3 per spot to avoid too much data)
			if len(spotPredictionsMap[spotID]) < 3 {
				spotPredictionsMap[spotID] = append(spotPredictionsMap[spotID], dataMap)
			}
		}
	}
	
	// Convert map to slice and flatten
	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}
	
	return predictions, nil
}

func (s *SwellPredictionService) GetClosestAIPredictionForSpot(
	spotName, regionName, countryName string,
) (map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	now := time.Now().UTC()
	startTime := now.Add(-12 * time.Hour)
	endTime := now.Add(48 * time.Hour)

	result, err := s.db.Query(buildPredictionQuery(spotID, startTime, endTime))
	if err != nil {
		return nil, fmt.Errorf("failed to query closest AI prediction: %w", err)
	}

	if len(result.Items) == 0 {
		fallbackStartTime := now.Add(-7 * 24 * time.Hour)
		fmt.Printf("No predictions found in time window, trying broader search from %s\n",
			fallbackStartTime.Format(time.RFC3339))

		fallbackResult, err := s.db.Query(buildFallbackPredictionQuery(spotID, fallbackStartTime))
		if err != nil {
			return nil, fmt.Errorf("failed to query fallback AI prediction: %w", err)
		}

		if len(fallbackResult.Items) == 0 {
			return nil, fmt.Errorf("no AI predictions found for spot %s", spotID)
		}

		result = fallbackResult
		fmt.Printf("Found %d predictions in fallback search\n", len(result.Items))
	}

	fmt.Printf("Found %d raw items for spot %s\n", len(result.Items), spotID)

	closestPrediction, err := findClosestPrediction(result.Items, now.Unix(), spotID)
	if err != nil || closestPrediction == nil {
		return nil, fmt.Errorf("no valid AI predictions found for spot %s", spotID)
	}

	return closestPrediction, nil
}