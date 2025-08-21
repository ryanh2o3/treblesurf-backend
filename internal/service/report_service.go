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

	if report.Date != "" {
		parsedDate, err := time.Parse("2006-01-02 15:04:05", report.Date)
		if err != nil {
			log.Printf("Failed to parse date '%s': %v, using current time", report.Date, err)
		} else {
			currentTime = parsedDate
		}
	}
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

		if report.Date == ""  {
			exifData, err := exif.Decode(bytes.NewReader(imageData))
			if err == nil {
				if dateTime, err := exifData.DateTime(); err == nil {
					currentTime = dateTime
					log.Printf("Extracted EXIF time from image: %v", currentTime)
				}
			}
		}
	}

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
		// Extract base64 data again for validation and upload
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

		// Validate image using Rekognition
		valid, err := s.validateImageWithRekognition(imageData)
		if err != nil {
			if strings.Contains(err.Error(), "image does not appear to be surf-related") {
				return fmt.Errorf("image validation failed: %v. Please upload a photo that clearly shows the ocean, waves, beach, or coastline", err.Error())
			}
			return fmt.Errorf("image validation failed: %v", err)
		}

		if !valid {
			return fmt.Errorf("image validation failed: image does not appear to be surf-related. Please upload a photo showing the ocean, waves, beach, or coastline")
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

	log.Print("done putting")

	// Build message for WebSocket broadcasting
	message := map[string]interface{}{
		"action": "new_report",
		"data": map[string]interface{}{
			"country":       report.Country,
			"region":        report.Region,
			"spot":          report.Spot,
			"quality":       report.Quality,
			"surfSize":      report.SurfSize,
			"windAmount":    report.WindAmount,
			"windDirection": report.WindDirection,
			"messiness":     report.Messiness,
			"consistency":   report.Consistency,
			"reporter":      userName,
			"imageKey":      s3KeyReport,
			"reportTime":    currentTime.Format(time.RFC3339),
		},
	}

	// Get spot subscribers and broadcast (this is what was missing!)
	subscribers, err := s.getSpotSubscribers(report.Country, report.Region, report.Spot)
	if err != nil {
		log.Printf("Failed to get subscribers: %v", err)
	} else {
		// Broadcast to subscribers asynchronously
		go func() {
			err := s.broadcastToUsers(subscribers, message)
			if err != nil {
				log.Printf("Failed to broadcast message: %v", err)
			}
		}()
	}

	return nil
}

// SubmitSurfReportWithS3Image submits a new surf report with a pre-uploaded S3 image
func (s *ReportService) SubmitSurfReportWithS3Image(report *model.ReportWithS3Image, userEmail string, userName string) error {
	currentTime := time.Now()
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)

	if report.Date != "" {
		parsedDate, err := time.Parse("2006-01-02 15:04:05", report.Date)
		if err != nil {
			log.Printf("Failed to parse date '%s': %v, using current time", report.Date, err)
		} else {
			currentTime = parsedDate
		}
	}

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

	// Process pre-uploaded image if provided
	if report.ImageKey != "" {
		// Retrieve the image from S3 for validation
		imageData, err := s.s3Storage.GetObject(s.bucketName, report.ImageKey)
		if err != nil {
			// If image doesn't exist, clean up and return error
			log.Printf("Failed to retrieve pre-uploaded image %s: %v", report.ImageKey, err)
			// Try to delete the image key if it exists
			_ = s.s3Storage.DeleteObject(s.bucketName, report.ImageKey)
			return fmt.Errorf("failed to retrieve pre-uploaded image: %v", err)
		}

		// Validate image using Rekognition
		valid, err := s.validateImageWithRekognition(imageData)
		if err != nil {
			// Clean up invalid image
			log.Printf("Failed to validate pre-uploaded image %s: %v", report.ImageKey, err)
			_ = s.s3Storage.DeleteObject(s.bucketName, report.ImageKey)
			
			if strings.Contains(err.Error(), "image does not appear to be surf-related") {
				return fmt.Errorf("S3 image validation failed: %v. Please upload a photo that clearly shows the ocean, waves, beach, or coastline", err.Error())
			}
			return fmt.Errorf("S3 image validation failed: %v", err)
		}

		if !valid {
			// Clean up invalid image
			log.Printf("Pre-uploaded image %s failed validation, deleting", report.ImageKey)
			_ = s.s3Storage.DeleteObject(s.bucketName, report.ImageKey)
			return fmt.Errorf("S3 image validation failed: image does not appear to be surf-related. Please upload a photo showing the ocean, waves, beach, or coastline")
		}
		
		// Store the S3 key in DynamoDB when validation succeeds
		item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(report.ImageKey)}
		s3KeyReport = report.ImageKey
	}

	// Insert into DynamoDB
	input := &dynamodb.PutItemInput{
		TableName: aws.String("SurfReports"),
		Item:      item,
	}

	_, err := s.dbStorage.PutItem(input)
	if err != nil {
		// If database insertion fails, clean up the image
		if s3KeyReport != "" {
			log.Printf("Database insertion failed, cleaning up image %s", s3KeyReport)
			_ = s.s3Storage.DeleteObject(s.bucketName, s3KeyReport)
		}
		return fmt.Errorf("failed to store report: %v", err)
	}

	log.Print("done putting")

	// Build message for WebSocket broadcasting
	message := map[string]interface{}{
		"action": "new_report",
		"data": map[string]interface{}{
			"country":       report.Country,
			"region":        report.Region,
			"spot":          report.Spot,
			"quality":       report.Quality,
			"surfSize":      report.SurfSize,
			"windAmount":    report.WindAmount,
			"windDirection": report.WindDirection,
			"messiness":     report.Messiness,
			"consistency":   report.Consistency,
			"reporter":      userName,
			"imageKey":      s3KeyReport,
			"reportTime":    currentTime.Format(time.RFC3339),
		},
	}

	// Get spot subscribers and broadcast
	subscribers, err := s.getSpotSubscribers(report.Country, report.Region, report.Spot)
	if err != nil {
		log.Printf("Failed to get subscribers: %v", err)
	} else {
		// Broadcast to subscribers asynchronously
		go func() {
			err := s.broadcastToUsers(subscribers, message)
			if err != nil {
				log.Printf("Failed to broadcast message: %v", err)
			}
		}()
	}

	return nil
}

// GenerateImageUploadURL generates a presigned URL for uploading an image to S3
func (s *ReportService) GenerateImageUploadURL(country, region, spot, userEmail string) (*model.PresignedUploadResponse, error) {
	// Generate a predictable S3 key based on location and user
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", country, region, spot)
	currentTime := time.Now()
	
	imageKey := fmt.Sprintf(
		"surf-reports/%s/%s_%s.jpg",
		countryRegionSpot,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		userEmail,
	)

	// Generate presigned URL valid for 15 minutes
	presignedURL, err := s.s3Storage.GeneratePresignedUploadURL(s.bucketName, imageKey, 15*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	expiresAt := currentTime.Add(15 * time.Minute)

	return &model.PresignedUploadResponse{
		UploadURL: presignedURL,
		ImageKey:  imageKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
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
		return false, fmt.Errorf("image analysis failed: %v", err)
	}

	validLabels := []string{"Sea", "Water", "Sea Waves", "Beach", "Coast"}
	var detectedLabels []string
	
	for _, label := range result.Labels {
		detectedLabels = append(detectedLabels, *label.Name)
		for _, validLabel := range validLabels {
			if strings.EqualFold(*label.Name, validLabel) {
				return true, nil
			}
		}
	}

	// Return a helpful error message with detected labels
	if len(detectedLabels) > 0 {
		return false, fmt.Errorf("image does not appear to be surf-related. Detected: %s. Please upload a photo showing the ocean, waves, beach, or coastline", strings.Join(detectedLabels[:min(5, len(detectedLabels))], ", "))
	}
	
	return false, fmt.Errorf("image could not be analyzed. Please ensure the image is clear and shows the surf conditions")
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// uploadImageToS3 uploads an image to S3
func (s *ReportService) uploadImageToS3(imageData []byte, key string) (string, error) {
	err := s.s3Storage.PutObject(s.bucketName, key, imageData, "image/jpeg")
	if err != nil {
		return "", fmt.Errorf("failed to upload image to S3: %v", err)
	}

	return key, nil
}

// ValidateImageKeyExists checks if an image key exists in S3 and is accessible
func (s *ReportService) ValidateImageKeyExists(imageKey string) (bool, error) {
	if imageKey == "" {
		return false, fmt.Errorf("image key is empty")
	}

	// Try to get the object metadata to check if it exists
	_, err := s.s3Storage.GetObject(s.bucketName, imageKey)
	if err != nil {
		return false, fmt.Errorf("image key %s does not exist or is not accessible: %v", imageKey, err)
	}

	return true, nil
}

func (s *ReportService) IsValidSurfSize(swellSize string) bool {
	return validation.IsValidSurfSize(swellSize)
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

// getSpotSubscribers retrieves subscribers for a specific spot
func (s *ReportService) getSpotSubscribers(country, region, spot string) ([]string, error) {
	// TODO: Implement spot subscribers retrieval
	// For now, return empty list
	return []string{}, nil
}

// broadcastToUsers broadcasts a message to multiple users via WebSocket
func (s *ReportService) broadcastToUsers(subscribers []string, message interface{}) error {
	// TODO: Implement user broadcasting
	// For now, just log the message
	log.Printf("Broadcasting message to %d subscribers: %v", len(subscribers), message)
	return nil
}

// CleanupOrphanedImage removes an image from S3 if it's not associated with any report
func (s *ReportService) CleanupOrphanedImage(imageKey string) error {
	if imageKey == "" {
		return nil
	}

	log.Printf("Cleaning up orphaned image: %s", imageKey)
	err := s.s3Storage.DeleteObject(s.bucketName, imageKey)
	if err != nil {
		log.Printf("Failed to cleanup orphaned image %s: %v", imageKey, err)
		return fmt.Errorf("failed to cleanup orphaned image: %v", err)
	}

	log.Printf("Successfully cleaned up orphaned image: %s", imageKey)
	return nil
}
