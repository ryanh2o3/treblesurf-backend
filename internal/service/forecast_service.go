package service

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type ForecastService struct {
	db *dynamodb.DynamoDB
}

func NewForecastService(db *dynamodb.DynamoDB) *ForecastService {
	return &ForecastService{db: db}
}

func (s *ForecastService) GetSpotForecast(spotName, regionName, countryName string) ([]map[string]interface{}, error) {
	return s.queryForecastByDateTime(spotName, regionName, countryName, nil)
}

func (s *ForecastService) GetListSpotsForecast(spots []string, regionName, countryName string) ([][]map[string]interface{}, error) {
	var spotIds []string
	for _, spot := range spots {
		spotIds = append(spotIds, fmt.Sprintf("%s#%s#%s", countryName, regionName, spot))
	}

	return s.queryMultipleSpotForecasts(spotIds, aws.Int64(72))
}

func (s *ForecastService) GetRegionForecast(regionName, countryName string) ([]map[string]interface{}, error) {
	forecastDate := time.Now().Format("2006-01-02")

	input := &dynamodb.QueryInput{
		TableName: aws.String("SpotForecastData"),
		KeyConditionExpression: aws.String("ForecastDate = :date AND begins_with(country_region_spot, :location)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":date": {
				S: aws.String(forecastDate),
			},
			":location": {
				S: aws.String(fmt.Sprintf("%s_%s_", countryName, regionName)),
			},
		},
		ScanIndexForward: aws.Bool(false),
	}

	result, err := s.db.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var forecasts []map[string]interface{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
	if err != nil {
		return nil, err
	}

	return forecasts, nil
}

func (s *ForecastService) GetCurrentWeather(spotName, regionName, countryName string) ([]map[string]interface{}, error) {
	return s.queryForecastByDateTime(spotName, regionName, countryName, aws.Int64(1))
}

func (s *ForecastService) queryForecastByDateTime(spotName, regionName, countryName string, limit *int64) ([]map[string]interface{}, error) {
	spotId := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	currentEpoch := time.Now().Unix()
	
	input := &dynamodb.QueryInput{
		TableName: aws.String("SpotForecastData"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotId),
			},
			":current_time": {
				S: aws.String(fmt.Sprintf("%d", currentEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	}
	if limit != nil {
		input.Limit = limit
	}

	result, err := s.db.Query(input)
	if err != nil {
		return nil, err
	}

	var forecasts []map[string]interface{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
	if err != nil {
		return nil, err
	}

	return forecasts, nil
}

func (s *ForecastService) queryMultipleSpotForecasts(spotIds []string, limit *int64) ([][]map[string]interface{}, error) {
	currentEpoch := time.Now().Unix()
	var allForecasts [][]map[string]interface{}

	for _, spotId := range spotIds {
		input := &dynamodb.QueryInput{
			TableName: aws.String("SpotForecastData"),
			KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":spot_id": {
					S: aws.String(spotId),
				},
				":current_time": {
					S: aws.String(fmt.Sprintf("%d", currentEpoch)),
				},
			},
			ScanIndexForward: aws.Bool(true),
		}
		if limit != nil {
			input.Limit = limit
		}

		result, err := s.db.Query(input)
		if err != nil {
			return nil, err
		}

		var forecasts []map[string]interface{}
		err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &forecasts)
		if err != nil {
			return nil, err
		}

		allForecasts = append(allForecasts, forecasts)
	}

	return allForecasts, nil
}
