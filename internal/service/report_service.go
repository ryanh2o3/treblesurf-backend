package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/storage"
	"treblesurf-backend/internal/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/rwcarlsen/goexif/exif"
)

type ReportService struct {
	dbStorage          storage.DynamoDBStorage
	s3Storage          storage.S3Storage
	rekognitionClient  *rekognition.Rekognition
	bucketName         string
}

func NewReportService(dbStorage storage.DynamoDBStorage, s3Storage storage.S3Storage, rekognitionClient *rekognition.Rekognition, bucketName string) *ReportService {
	return &ReportService{
		dbStorage:         dbStorage,
		s3Storage:         s3Storage,
		rekognitionClient: rekognitionClient,
		bucketName:        bucketName,
	}
}

// SubmitSurfReport submits a new surf report
func (s *ReportService) SubmitSurfReport(report *model.ReportWithImage, userEmail string, userName string) error {
	currentTime := time.Now()
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	dateReported := fmt.Sprintf("%s_%s", currentTime, userEmail)

	// Create the DynamoDB item
	item := map[string]*dynamodb.AttributeValue{
		"country_region_spot": {S: aws.String(countryRegionSpot)},
		"dateReported":        {S: aws.String(dateReported)},
		"SurfSize":            {S: aws.String(report.SurfSize)},
		"WindAmount":          {S: aws.String(report.WindAmount)},
		"WindDirection":       {S: aws.String(report.WindDirection)},
		"Consistency":         {S: aws.String(report.Consistency)},
		"Quality":             {S: aws.String(report.Quality)},
		"Messiness":           {S: aws.String(report.Messiness)},
		"UserEmail":           {S: aws.String(userEmail)},
		"Reporter":            {S: aws.String(userName)},
		"Time":                {S: aws.String(currentTime.String())},
	}

	var s3KeyReport = ""

	// Process image if provided
	if report.ImageData != "" {
		// Extract base64 data
		base64String := report.ImageData
		// Handle data URIs by removing the prefix
		if strings.HasPrefix(base64String, "data:") {
			// Find the comma that separates the header from the data
			commaIndex := strings.Index(base64String, ",")
			if commaIndex != -1 {
				base64String = base64String[commaIndex+1:]
			}
		}
		imageData, err := base64.StdEncoding.DecodeString(base64String)
		if err != nil {
			return fmt.Errorf("invalid image data: %v", err)
		}

		// Extract time from EXIF data if available
		exifData, err := exif.Decode(bytes.NewReader(imageData))
		if err == nil {
			if dateTime, err := exifData.DateTime(); err == nil {
				currentTime = dateTime
			}
		}

		// Validate image using Rekognition
		valid, err := s.validateImageWithRekognition(imageData)
		if err != nil {
			return fmt.Errorf("failed to validate image: %v", err)
		}

		if valid {
			// Upload to S3
			imageKey := fmt.Sprintf(
				"surf-reports/%s/%s_%s.jpg",
				countryRegionSpot,
				currentTime.UTC().Format("2006-01-02T15:04:05Z"),
				userEmail,
			)
			s3Key, err := s.uploadImageToS3(imageData, imageKey)
			if err != nil {
				return fmt.Errorf("failed to upload image: %v", err)
			}

			// Store just the S3 key in DynamoDB
			item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(s3Key)}
			s3KeyReport = s3Key
		} else {
			return fmt.Errorf("image validation failed")
		}
	}

	// Insert into DynamoDB
	input := &dynamodb.PutItemInput{
		TableName: aws.String("SurfReports"),
		Item:      item,
	}

	_, err := s.dbStorage.PutItem(input)
	if err != nil {
		return fmt.Errorf("failed to store report: %v", err)
	}

	log.Printf("Report submitted successfully with image key: %s", s3KeyReport)
	return nil
}

// GetTodaysSurfReports retrieves surf reports for a specific spot
func (s *ReportService) GetTodaysSurfReports(countryName, regionName, spotName string) ([]map[string]interface{}, error) {
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)

	input := &dynamodb.QueryInput{
		TableName: aws.String("SurfReports"),
		KeyConditionExpression: aws.String("country_region_spot = :crs"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crs": {S: aws.String(countryRegionSpot)},
		},
		ScanIndexForward: aws.Bool(false), // Sort in descending order to get the latest reports
		Limit:            aws.Int64(1),     // Limit to the last report
	}

	result, err := s.dbStorage.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %v", err)
	}

	var reports []map[string]interface{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &reports)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}

	return reports, nil
}

// GetReportImage retrieves a report image from S3
func (s *ReportService) GetReportImage(imageKey string) ([]byte, string, error) {
	// Read the image data using the interface method
	imageData, err := s.s3Storage.GetObject(s.bucketName, imageKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %v", err)
	}

	// For now, assume JPEG content type
	// TODO: Implement proper content type detection
	contentType := "image/jpeg"

	return imageData, contentType, nil
}

// validateImageWithRekognition validates an image using AWS Rekognition
func (s *ReportService) validateImageWithRekognition(imageData []byte) (bool, error) {
	if os.Getenv("GO_ENV") == "development" {
		// In development, always return true to allow all images
		return true, nil
	}

	input := &rekognition.DetectLabelsInput{
		Image: &rekognition.Image{
			Bytes: imageData,
		},
		MinConfidence: aws.Float64(90.0),
	}

	result, err := s.rekognitionClient.DetectLabels(input)
	if err != nil {
		return false, fmt.Errorf("rekognition error: %v", err)
	}

	validLabels := []string{"Sea", "Water", "Sea Waves", "Beach", "Coast"}
	for _, label := range result.Labels {
		for _, validLabel := range validLabels {
			if strings.EqualFold(*label.Name, validLabel) {
				return true, nil
			}
		}
	}

	return false, nil
}

// uploadImageToS3 uploads an image to S3
func (s *ReportService) uploadImageToS3(imageData []byte, key string) (string, error) {
	err := s.s3Storage.PutObject(s.bucketName, key, imageData, "image/jpeg")
	if err != nil {
		return "", fmt.Errorf("failed to upload image to S3: %v", err)
	}

	return key, nil
}

// Validation methods now use the validation package
func (s *ReportService) IsValidSwellSize(swellSize string) bool {
	return validation.IsValidSwellSize(swellSize)
}

func (s *ReportService) IsValidWindAmount(windAmount string) bool {
	return validation.IsValidWindAmount(windAmount)
}

func (s *ReportService) IsValidWindDirection(windDirection string) bool {
	return validation.IsValidWindDirection(windDirection)
}

func (s *ReportService) IsValidSurfConditions(surfConditions string) bool {
	return validation.IsValidSurfConditions(surfConditions)
}

func (s *ReportService) IsValidSurfDifficulty(surfDifficulty string) bool {
	return validation.IsValidSurfDifficulty(surfDifficulty)
}
