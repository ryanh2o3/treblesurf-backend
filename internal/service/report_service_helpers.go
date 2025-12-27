package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/rwcarlsen/goexif/exif"
)

// getUserAndValidate retrieves and validates a user by email.
func (s *ReportService) getUserAndValidate(userEmail string) (*model.User, error) {
	user, err := s.userService.GetUserByEmail(userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.UUID == "" {
		return nil, fmt.Errorf("user does not have a UUID")
	}
	return user, nil
}

// parseReportDate parses the report date string, falling back to current time if invalid.
func parseReportDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	parsedDate, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err != nil {
		log.Printf("Failed to parse date '%s': %v, using current time", dateStr, err)
		return time.Now()
	}

	return parsedDate
}

// extractDateFromImageData extracts EXIF date from image data if available.
func extractDateFromImageData(imageData []byte, currentTime time.Time) time.Time {
	exifData, err := exif.Decode(bytes.NewReader(imageData))
	if err != nil {
		return currentTime
	}

	dateTime, err := exifData.DateTime()
	if err != nil {
		return currentTime
	}

	log.Printf("Extracted EXIF time from image: %v", dateTime)
	return dateTime
}

// decodeBase64ImageData decodes base64 image data, handling data URIs.
func decodeBase64ImageData(base64String string) ([]byte, error) {
	// Handle data URIs by removing the prefix
	if strings.HasPrefix(base64String, "data:") {
		commaIndex := strings.Index(base64String, ",")
		if commaIndex != -1 {
			base64String = base64String[commaIndex+1:]
		}
	}

	return base64.StdEncoding.DecodeString(base64String)
}

// addReportFieldsToItem adds report fields to a DynamoDB item.
func addReportFieldsToItem(item map[string]*dynamodb.AttributeValue, surfSize, windAmount, windDirection, consistency, quality, messiness string) {
	if surfSize != "" {
		item["SurfSize"] = &dynamodb.AttributeValue{S: aws.String(surfSize)}
	}
	if windAmount != "" {
		item["WindAmount"] = &dynamodb.AttributeValue{S: aws.String(windAmount)}
	}
	if windDirection != "" {
		item["WindDirection"] = &dynamodb.AttributeValue{S: aws.String(windDirection)}
	}
	if consistency != "" {
		item["Consistency"] = &dynamodb.AttributeValue{S: aws.String(consistency)}
	}
	if quality != "" {
		item["Quality"] = &dynamodb.AttributeValue{S: aws.String(quality)}
	}
	if messiness != "" {
		item["Messiness"] = &dynamodb.AttributeValue{S: aws.String(messiness)}
	}
}

// buildWebSocketMessage builds a WebSocket message for broadcasting new reports.
func buildWebSocketMessage(country, region, spot, userName, userUUID, imageKey, videoKey, mediaType string, iosValidated bool, reportFields map[string]string, currentTime time.Time) map[string]interface{} {
	data := map[string]interface{}{
		"country":      country,
		"region":       region,
		"spot":         spot,
		"reporter":     userName,
		"reportedBy":   userUUID,
		"imageKey":     imageKey,
		"videoKey":     videoKey,
		"mediaType":    mediaType,
		"iosValidated": iosValidated,
		"reportTime":   currentTime.Format(time.RFC3339),
	}

	// Add report fields
	for key, value := range reportFields {
		data[key] = value
	}

	return map[string]interface{}{
		"action": "new_report",
		"data":   data,
	}
}

// broadcastReportMessage broadcasts a report message to spot subscribers.
func (s *ReportService) broadcastReportMessage(country, region, spot string, message map[string]interface{}) {
	subscribers, err := s.getSpotSubscribers(country, region, spot)
	if err != nil {
		log.Printf("Failed to get subscribers: %v", err)
		return
	}

	go func() {
		s.broadcastToUsers(subscribers, message)
	}()
}

// createBaseReportItem creates the base DynamoDB item structure for a surf report.
func (s *ReportService) createBaseReportItem(countryRegionSpot, dateReported, userEmail, userName, userUUID string, currentTime time.Time, mediaType string, iosValidated bool) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"country_region_spot": {S: aws.String(countryRegionSpot)},
		"dateReported":        {S: aws.String(dateReported)},
		"UserEmail":           {S: aws.String(userEmail)},
		"Reporter":            {S: aws.String(userName)},
		"Time":                {S: aws.String(currentTime.String())},
		"reportedBy":          {S: aws.String(userUUID)},
		"MediaType":           {S: aws.String(mediaType)},
		"IOSValidated":        {BOOL: aws.Bool(iosValidated)},
	}
}

// storeReport stores a report item in DynamoDB.
func (s *ReportService) storeReport(item map[string]*dynamodb.AttributeValue) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String("SurfReports"),
		Item:      item,
	}

	_, err := s.dbStorage.PutItem(input)
	if err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	log.Print("done putting")
	return nil
}

// processBase64Image processes base64 image data: extracts date from EXIF if needed, validates, and uploads to S3.
func (s *ReportService) processBase64Image(imageDataStr, dateStr, countryRegionSpot, userUUID string, currentTime *time.Time) (string, error) {
	if imageDataStr == "" {
		return "", nil
	}

	decoded, err := decodeBase64ImageData(imageDataStr)
	if err != nil {
		return "", model.ErrInvalidImageData
	}

	// Extract EXIF date if no date provided
	if dateStr == "" {
		*currentTime = extractDateFromImageData(decoded, *currentTime)
	}

	// Validate image using Rekognition
	valid, err := s.validateImageWithRekognition(decoded)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", model.ErrImageNotSurfRelated
	}

	// Upload to S3
	imageKey := fmt.Sprintf(
		"surf-reports/%s/%s_%s.jpg",
		countryRegionSpot,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		userUUID,
	)
	s3Key, err := s.uploadImageToS3(decoded, imageKey)
	if err != nil {
		return "", model.NewImageValidationError(err, "failed to upload image")
	}

	return s3Key, nil
}

// determineMediaType determines the media type based on image and video keys.
func determineMediaType(hasImage, hasVideo bool) string {
	switch {
	case hasImage && hasVideo:
		return "both"
	case hasImage:
		return "image"
	case hasVideo:
		return "video"
	default:
		return "none"
	}
}

// processS3ImageForReport processes an S3 image key: retrieves, validates with Rekognition, and adds to item.
func (s *ReportService) processS3ImageForReport(imageKey string, item map[string]*dynamodb.AttributeValue) (string, error) {
	if imageKey == "" {
		return "", nil
	}

	imageData, err := s.s3Storage.GetObject(s.bucketName, imageKey)
	if err != nil {
		log.Printf("Failed to retrieve pre-uploaded image %s: %v", imageKey, err)
		_ = s.s3Storage.DeleteObject(s.bucketName, imageKey)
		return "", model.ErrImageRetrievalFailed
	}

	valid, err := s.validateImageWithRekognition(imageData)
	if err != nil {
		log.Printf("Failed to validate pre-uploaded image %s: %v", imageKey, err)
		_ = s.s3Storage.DeleteObject(s.bucketName, imageKey)
		return "", err
	}
	if !valid {
		log.Printf("Pre-uploaded image %s failed validation, deleting", imageKey)
		_ = s.s3Storage.DeleteObject(s.bucketName, imageKey)
		return "", model.ErrImageNotSurfRelated
	}

	item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(imageKey)}
	return imageKey, nil
}

// storeReportWithCleanup stores a report and cleans up S3 image if storage fails.
func (s *ReportService) storeReportWithCleanup(item map[string]*dynamodb.AttributeValue, s3Key string) error {
	err := s.storeReport(item)
	if err != nil && s3Key != "" {
		log.Printf("Database insertion failed, cleaning up image %s", s3Key)
		_ = s.s3Storage.DeleteObject(s.bucketName, s3Key)
	}
	return err
}

// processIOSMediaKeys processes iOS validated image and video keys (trusts client-side validation).
func (s *ReportService) processIOSMediaKeys(imageKey, videoKey string, item map[string]*dynamodb.AttributeValue) (string, string) {
	var s3KeyReport, videoKeyReport string

	if imageKey != "" {
		log.Printf("iOS validated report with image: %s", imageKey)
		item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(imageKey)}
		s3KeyReport = imageKey
	}

	if videoKey != "" {
		log.Printf("iOS validated report with video: %s", videoKey)
		item["VideoKey"] = &dynamodb.AttributeValue{S: aws.String(videoKey)}
		videoKeyReport = videoKey
	}

	return s3KeyReport, videoKeyReport
}

// buildSpotReportsQueryInput builds a DynamoDB query input for retrieving spot reports.
func (s *ReportService) buildSpotReportsQueryInput(countryRegionSpot string, limit int, lastEvaluatedKey map[string]*dynamodb.AttributeValue) *dynamodb.QueryInput {
	input := &dynamodb.QueryInput{
		TableName:              aws.String("SurfReports"),
		KeyConditionExpression: aws.String("country_region_spot = :crs"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crs": {S: aws.String(countryRegionSpot)},
		},
		ScanIndexForward: aws.Bool(false), // Sort in descending order
	}

	if limit > 0 {
		input.Limit = aws.Int64(int64(limit))
	}
	if lastEvaluatedKey != nil {
		input.ExclusiveStartKey = lastEvaluatedKey
	}

	return input
}

// unmarshalSpotReports unmarshals DynamoDB items into report maps.
func (s *ReportService) unmarshalSpotReports(items []map[string]*dynamodb.AttributeValue) ([]map[string]interface{}, error) {
	var reports []map[string]interface{}
	err := dynamodbattribute.UnmarshalListOfMaps(items, &reports)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %w", err)
	}
	return reports, nil
}

// normalizeSpotReports normalizes report maps by removing sensitive fields and adding defaults.
func (s *ReportService) normalizeSpotReports(reports []map[string]interface{}) {
	for _, report := range reports {
		delete(report, "UserEmail") // Remove sensitive field

		// Ensure new fields have defaults if missing
		setDefaultIfMissing(report, "VideoKey", "")
		setDefaultIfMissing(report, "MediaType", "image")
		setDefaultIfMissing(report, "IOSValidated", false)

		// Ensure all required fields have defaults
		setDefaultIfMissing(report, "Consistency", "")
		setDefaultIfMissing(report, "Messiness", "")
		setDefaultIfMissing(report, "Quality", "")
		setDefaultIfMissing(report, "SurfSize", "")
		setDefaultIfMissing(report, "WindAmount", "")
		setDefaultIfMissing(report, "WindDirection", "")
		setDefaultIfMissing(report, "Reporter", "Anonymous")
	}
}

// setDefaultIfMissing sets a default value for a key if it doesn't exist in the map.
func setDefaultIfMissing(m map[string]interface{}, key string, defaultValue interface{}) {
	if _, exists := m[key]; !exists {
		m[key] = defaultValue
	}
}

// queryCurrentForecast queries for the most recent forecast data.
func (s *ReportService) queryCurrentForecast(spotID string) (*dynamodb.QueryOutput, error) {
	currentEpoch := time.Now().Unix()
	input := &dynamodb.QueryInput{
		TableName:              aws.String("SpotForecastData"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":current_time": {
				S: aws.String(fmt.Sprintf("%d", currentEpoch-3600)), // 1 hour ago
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(1),
	}

	return s.dbStorage.Query(input)
}

// queryHistoricalForecast queries for forecast data looking backwards up to 24 hours.
func (s *ReportService) queryHistoricalForecast(spotID string) (*dynamodb.QueryOutput, error) {
	currentEpoch := time.Now().Unix()

	for i := 1; i <= 24; i++ {
		pastEpoch := currentEpoch - int64(i*3600)
		input := &dynamodb.QueryInput{
			TableName:              aws.String("SpotForecastData"),
			KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp > :current_time"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":spot_id": {
					S: aws.String(spotID),
				},
				":current_time": {
					S: aws.String(fmt.Sprintf("%d", pastEpoch)),
				},
			},
			ScanIndexForward: aws.Bool(true),
			Limit:            aws.Int64(1),
		}

		result, err := s.dbStorage.Query(input)
		if err == nil && len(result.Items) > 0 {
			return result, nil
		}
	}

	return &dynamodb.QueryOutput{}, nil
}

// unmarshalForecast unmarshals a DynamoDB item into a forecast map.
func (s *ReportService) unmarshalForecast(item map[string]*dynamodb.AttributeValue) (map[string]interface{}, error) {
	var forecast map[string]interface{}
	err := dynamodbattribute.UnmarshalMap(item, &forecast)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal forecast: %w", err)
	}
	return forecast, nil
}

// extractWindData extracts wind speed and direction from forecast data.
func (s *ReportService) extractWindData(forecast map[string]interface{}) (float64, float64, error) {
	data, ok := forecast["data"].(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("invalid forecast data structure")
	}

	windSpeed := extractFloatFromData(data, "windSpeed")
	windDirection := extractFloatFromData(data, "windDirection")

	return windSpeed, windDirection, nil
}

// extractFloatFromData extracts a float value from data map, handling both float64 and string types.
func extractFloatFromData(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	if valStr, ok := data[key].(string); ok {
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			return val
		}
	}
	return 0
}

