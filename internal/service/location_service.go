package service

import (
	"encoding/base64"
	"fmt"
	"strings"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type LocationService struct {
	dbStorage  storage.DynamoDBStorage
	s3Storage  storage.S3Storage
	bucketName string
}

func NewLocationService(dbStorage storage.DynamoDBStorage, s3Storage storage.S3Storage, bucketName string) *LocationService {
	return &LocationService{
		dbStorage:  dbStorage,
		s3Storage:  s3Storage,
		bucketName: bucketName,
	}
}

// GetRegions returns a list of regions for the specified country.
func (s *LocationService) GetRegions(countryName string) ([]string, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String("LocationData"),
	}

	result, err := s.dbStorage.Scan(input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan locations: %v", err)
	}

	var locations []map[string]interface{}
	err = storage.UnmarshalListOfMaps(result.Items, &locations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal locations: %v", err)
	}

	var regions []string
	for _, location := range locations {
		parts := strings.Split(location["country_region_spot"].(string), "/")
		if parts[0] == countryName && len(parts) > 1 {
			region := parts[1]
			if !contains(regions, region) {
				regions = append(regions, region)
			}
		}
	}

	return regions, nil
}

// GetSpots returns a list of spots for the specified country and region.
func (s *LocationService) GetSpots(countryName, regionName string) ([]model.LocationInfo, error) {
	input := &dynamodb.ScanInput{
		TableName: aws.String("LocationData"),
		FilterExpression: aws.String("begins_with(country_region_spot, :location)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {
				S: aws.String(fmt.Sprintf("%s/%s/", countryName, regionName)),
			},
		},
	}

	result, err := s.dbStorage.Scan(input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan spots: %v", err)
	}

	var locations []model.LocationInfo
	err = storage.UnmarshalListOfMaps(result.Items, &locations)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spots: %v", err)
	}

	// Fetch and encode images for each location
	for i := range locations {
		// Extract spot name from CountryRegionSpot field
		// Format: country/region/spot
		parts := strings.Split(locations[i].CountryRegionSpot, "/")
		if len(parts) >= 3 {
			spotName := parts[2]
			
			// Construct the S3 key for the image
			// Format: spot-images/country_region_spot.jpg
			// Remove all spaces from input strings since S3 image names have no spaces
			imageKey := fmt.Sprintf("spot-images/%s_%s_%s.jpg", 
				strings.ReplaceAll(countryName, " ", ""), 
				strings.ReplaceAll(regionName, " ", ""), 
				strings.ReplaceAll(spotName, " ", ""))
			
			// Try to fetch the image from S3
			imageData, err := s.s3Storage.GetObject(s.bucketName, imageKey)
			if err != nil {
				// Log error but don't fail the request
				// locations[i].ImageString will remain empty
				fmt.Printf("Failed to fetch image for %s: %v\n", imageKey, err)
			} else {
				// Encode the image data as base64
				locations[i].ImageString = base64.StdEncoding.EncodeToString(imageData)
			}
		}
	}

	return locations, nil
}

// GetLocationInfo retrieves detailed information for a specific location.
func (s *LocationService) GetLocationInfo(countryName, regionName, spotName string) (*model.LocationInfo, error) {
	input := &dynamodb.QueryInput{
		TableName: aws.String("LocationData"),
		KeyConditionExpression: aws.String("country_region_spot = :location"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {
				S: aws.String(fmt.Sprintf("%s/%s/%s", countryName, regionName, spotName)),
			},
		},
	}

	result, err := s.dbStorage.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query location: %v", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no location found")
	}

	var location model.LocationInfo
	err = storage.UnmarshalMap(result.Items[0], &location)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %v", err)
	}

	// Get image from S3 if available
	// Construct the S3 key for the image
	// Format: spot-images/country_region_spot.jpg
	// Remove all spaces from input strings since S3 image names have no spaces
	imageKey := fmt.Sprintf("spot-images/%s_%s_%s.jpg", 
		strings.ReplaceAll(countryName, " ", ""), 
		strings.ReplaceAll(regionName, " ", ""), 
		strings.ReplaceAll(spotName, " ", ""))
	
	imageData, err := s.s3Storage.GetObject(s.bucketName, imageKey)
	if err != nil {
		// Log error but don't fail the request
		// location.ImageString will remain empty
		fmt.Printf("Failed to fetch image for %s: %v\n", imageKey, err)
	} else {
		// Encode the image data as base64
		location.ImageString = base64.StdEncoding.EncodeToString(imageData)
	}

	return &location, nil
}

// GetCoordinates retrieves the latitude and longitude coordinates for a specific location.
func (s *LocationService) GetCoordinates(countryName, regionName, spotName string) ([]float64, error) {
	input := &dynamodb.QueryInput{
		TableName: aws.String("LocationData"),
		KeyConditionExpression: aws.String("country_region_spot = :location"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {
				S: aws.String(fmt.Sprintf("%s/%s/%s", countryName, regionName, spotName)),
			},
		},
	}

	result, err := s.dbStorage.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query coordinates: %v", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("no coordinates found")
	}

	var location map[string]interface{}
	err = storage.UnmarshalMap(result.Items[0], &location)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal coordinates: %v", err)
	}

	coordinates := []float64{
		location["Latitude"].(float64),
		location["Longitude"].(float64),
	}

	return coordinates, nil
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
