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
	
	// Get current time rounded to the hour (UTC)
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	currentHourTimestamp := fmt.Sprintf("%d", currentHour.Unix())
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SwellPredictions"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp >= :current_hour"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotId),
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
		
		spotId := prediction["spot_id"].(string)
		
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
			if len(spotPredictionsMap[spotId]) < 3 {
				spotPredictionsMap[spotId] = append(spotPredictionsMap[spotId], dataMap)
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
		
		spotId := prediction["spot_id"].(string)
		
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
			if len(spotPredictionsMap[spotId]) < 3 {
				spotPredictionsMap[spotId] = append(spotPredictionsMap[spotId], dataMap)
			}
		}
	}
	
	// Convert map to slice and flatten
	for _, spotPredictions := range spotPredictionsMap {
		predictions = append(predictions, spotPredictions...)
	}
	
	return predictions, nil
}

// GetClosestAIPredictionForSpot retrieves the closest AI prediction for a spot around the current time
func (s *SwellPredictionService) GetClosestAIPredictionForSpot(spotName, regionName, countryName string) (map[string]interface{}, error) {
	spotId := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	
	// Get current time
	now := time.Now().UTC()
	
	// Look for predictions within the last 6 hours and next 24 hours (30 hour window)
	// This gives us a wide range to find the closest arrival_time to current time
	startTime := now.Add(-6 * time.Hour)
	endTime := now.Add(24 * time.Hour)
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
		Limit:            aws.Int64(50),  // Limit to prevent too many results (increased for larger time window)
	}

	result, err := s.db.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query closest AI prediction: %v", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no AI predictions found for spot %s within the time window", spotId)
	}

	// Debug logging
	fmt.Printf("Found %d raw items for spot %s\n", len(result.Items), spotId)

	// Find the closest prediction to current time
	var closestPrediction map[string]interface{}
	var closestTimeDiff int64 = 999999999999 // Large number for comparison
	validItems := 0
	
	for i, item := range result.Items {
		var prediction map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &prediction)
		if err != nil {
			fmt.Printf("Failed to unmarshal item %d: %v\n", i, err)
			continue
		}
		
		// Extract the data field and properly unmarshal it
		if dataAttr, exists := item["data"]; exists {
			var dataMap map[string]interface{}
			err := dynamodbattribute.Unmarshal(dataAttr, &dataMap)
			if err != nil {
				fmt.Printf("Failed to unmarshal data field for item %d: %v\n", i, err)
				continue
			}
			
			// Calculate time difference from current time based on arrival_time
			if arrivalTime, ok := dataMap["arrival_time"].(string); ok {
				// Parse arrival_time (ISO format like "2024-01-15T14:30:00")
				if parsedTime, err := time.Parse("2006-01-02T15:04:05", arrivalTime); err == nil {
					timeDiff := now.Sub(parsedTime).Abs().Nanoseconds()
					validItems++
					fmt.Printf("Item %d: arrival_time=%s, time_diff=%.1fh\n", i, arrivalTime, float64(timeDiff)/3600000000000)
					if timeDiff < closestTimeDiff {
						closestTimeDiff = timeDiff
						closestPrediction = dataMap
						closestPrediction["spot_id"] = prediction["spot_id"]
						closestPrediction["forecast_timestamp"] = prediction["forecast_timestamp"]
						closestPrediction["generated_at"] = prediction["generated_at"]
					}
				} else {
					fmt.Printf("Failed to parse arrival_time for item %d: %v\n", i, err)
				}
			} else {
				fmt.Printf("No arrival_time string for item %d\n", i)
			}
		} else {
			fmt.Printf("No data field for item %d\n", i)
		}
	}
	
	fmt.Printf("Valid items processed: %d\n", validItems)
	
	if closestPrediction == nil {
		return nil, fmt.Errorf("no valid AI predictions found for spot %s", spotId)
	}
	
	return closestPrediction, nil
}