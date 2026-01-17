package service

import (
	"bytes"
	"context"
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

func addReportFieldsToItem(
	item map[string]*dynamodb.AttributeValue,
	surfSize, windAmount, windDirection, consistency, quality, messiness string,
) {
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

func buildWebSocketMessage(
	country, region, spot, userName, userUUID, imageKey, videoKey, mediaType string,
	iosValidated bool,
	reportFields map[string]string,
	currentTime time.Time,
) map[string]interface{} {
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

func (s *ReportService) createBaseReportItem(
	countryRegionSpot, dateReported, userEmail, userName, userUUID string,
	currentTime time.Time,
	mediaType string,
	iosValidated bool,
) map[string]*dynamodb.AttributeValue {
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

func (s *ReportService) storeReport(item map[string]*dynamodb.AttributeValue) error {
	var report model.SurfReport
	if err := dynamodbattribute.UnmarshalMap(item, &report); err != nil {
		return fmt.Errorf("failed to unmarshal report: %w", err)
	}

	if err := s.reportRepo.Create(context.Background(), &report); err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	log.Print("done putting")
	return nil
}

func (s *ReportService) processBase64Image(
	imageDataStr, dateStr, countryRegionSpot, userUUID string,
	currentTime *time.Time,
) (string, error) {
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

func (s *ReportService) processS3ImageForReport(
	imageKey string,
	item map[string]*dynamodb.AttributeValue,
) (string, error) {
	if imageKey == "" {
		return "", nil
	}

	imageData, err := s.mediaRepo.Download(context.Background(), imageKey)
	if err != nil {
		log.Printf("Failed to retrieve pre-uploaded image %s: %v", imageKey, err)
		if delErr := s.mediaRepo.Delete(context.Background(), imageKey); delErr != nil {
			log.Printf("Failed to cleanup image %s: %v", imageKey, delErr)
		}
		return "", model.ErrImageRetrievalFailed
	}

	valid, err := s.validateImageWithRekognition(imageData)
	if err != nil {
		log.Printf("Failed to validate pre-uploaded image %s: %v", imageKey, err)
		if delErr := s.mediaRepo.Delete(context.Background(), imageKey); delErr != nil {
			log.Printf("Failed to cleanup image %s: %v", imageKey, delErr)
		}
		return "", err
	}
	if !valid {
		log.Printf("Pre-uploaded image %s failed validation, deleting", imageKey)
		if delErr := s.mediaRepo.Delete(context.Background(), imageKey); delErr != nil {
			log.Printf("Failed to cleanup image %s: %v", imageKey, delErr)
		}
		return "", model.ErrImageNotSurfRelated
	}

	item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(imageKey)}
	return imageKey, nil
}

func (s *ReportService) storeReportWithCleanup(item map[string]*dynamodb.AttributeValue, s3Key string) error {
	err := s.storeReport(item)
	if err != nil && s3Key != "" {
		log.Printf("Database insertion failed, cleaning up image %s", s3Key)
		if delErr := s.mediaRepo.Delete(context.Background(), s3Key); delErr != nil {
			log.Printf("Failed to cleanup image %s: %v", s3Key, delErr)
		}
	}
	return err
}

func (s *ReportService) processIOSMediaKeys(
	imageKey, videoKey string,
	item map[string]*dynamodb.AttributeValue,
) (s3KeyReport, videoKeyReport string) {
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
func (s *ReportService) convertReportsToMaps(
	reports []*model.SurfReport,
) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		item, err := dynamodbattribute.MarshalMap(report)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal report: %w", err)
		}
		var reportMap map[string]interface{}
		if err := dynamodbattribute.UnmarshalMap(item, &reportMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report: %w", err)
		}
		out = append(out, reportMap)
	}
	return out, nil
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
func (s *ReportService) queryCurrentForecast(spotID string) (map[string]interface{}, error) {
	currentTime := time.Now().Add(-1 * time.Hour)
	forecasts, err := s.forecastDataRepo.QuerySince(context.Background(), spotID, currentTime, 1)
	if err != nil {
		return nil, err
	}
	if len(forecasts) == 0 {
		return nil, nil
	}
	return forecasts[0], nil
}

// queryHistoricalForecast queries for forecast data looking backwards up to 24 hours.
//
//nolint:unparam // Error return maintained for API consistency
func (s *ReportService) queryHistoricalForecast(spotID string) (map[string]interface{}, error) {
	currentTime := time.Now()

	for i := 1; i <= 24; i++ {
		pastTime := currentTime.Add(-time.Duration(i) * time.Hour)
		forecasts, err := s.forecastDataRepo.QuerySince(context.Background(), spotID, pastTime, 1)
		if err == nil && len(forecasts) > 0 {
			return forecasts[0], nil
		}
	}

	return nil, nil
}

// extractWindData extracts wind speed and direction from forecast data.
func (s *ReportService) extractWindData(forecast map[string]interface{}) (windSpeed, windDirection float64, err error) {
	data, ok := forecast["data"].(map[string]interface{})
	if !ok {
		return 0, 0, fmt.Errorf("invalid forecast data structure")
	}

	windSpeed = extractFloatFromData(data, "windSpeed")
	windDirection = extractFloatFromData(data, "windDirection")

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
