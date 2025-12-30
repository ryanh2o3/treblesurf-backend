package service

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/storage"
	"treblesurf-backend/internal/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

type ReportService struct {
	dbStorage         storage.DynamoDBStorage
	s3Storage         storage.S3Storage
	rekognitionClient *rekognition.Rekognition
	userService       *UserService
	bucketName        string
}

func NewReportService(
	dbStorage storage.DynamoDBStorage,
	s3Storage storage.S3Storage,
	rekognitionClient *rekognition.Rekognition,
	bucketName string,
	userService *UserService,
) *ReportService {
	return &ReportService{
		dbStorage:         dbStorage,
		s3Storage:         s3Storage,
		rekognitionClient: rekognitionClient,
		bucketName:        bucketName,
		userService:       userService,
	}
}

func (s *ReportService) SubmitSurfReport(report *model.ReportWithImage, userEmail string, userName string) error {
	user, err := s.getUserAndValidate(userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)

	// Process image data if provided
	s3KeyReport, err := s.processBase64Image(report.ImageData, report.Date, countryRegionSpot, user.UUID, &currentTime)
	if err != nil {
		return err
	}

	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)
	item := s.createBaseReportItem(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, "image", false,
	)
	addReportFieldsToItem(
		item, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	if s3KeyReport != "" {
		item["ImageKey"] = &dynamodb.AttributeValue{S: aws.String(s3KeyReport)}
	}

	if err := s.storeReport(item); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, "", "image", false, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

func (s *ReportService) SubmitSurfReportWithS3Image(
	report *model.ReportWithS3Image,
	userEmail string,
	userName string,
) error {
	user, err := s.getUserAndValidate(userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)

	item := s.createBaseReportItem(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, "image", false,
	)
	addReportFieldsToItem(
		item, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	s3KeyReport, err := s.processS3ImageForReport(report.ImageKey, item)
	if err != nil {
		return err
	}

	if err := s.storeReportWithCleanup(item, s3KeyReport); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, "", "image", false, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

// generateUploadURLParams contains common parameters for generating upload URLs
type generateUploadURLParams struct {
	user       *model.User
	keyPrefix  string
	fileExt    string
	expiration time.Duration
}

// prepareUploadURLParams validates user and prepares common parameters for URL generation
func (s *ReportService) prepareUploadURLParams(
	country, region, spot, userEmail string,
	fileExt string,
) (*generateUploadURLParams, error) {
	// Get the user's UUID
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

	// Generate a predictable S3 key based on location and user UUID
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", country, region, spot)
	keyPrefix := fmt.Sprintf("surf-reports/%s", countryRegionSpot)

	return &generateUploadURLParams{
		user:       user,
		keyPrefix:  keyPrefix,
		fileExt:    fileExt,
		expiration: 15 * time.Minute,
	}, nil
}

func (s *ReportService) GenerateImageUploadURL(
	country, region, spot, userEmail string,
) (*model.PresignedUploadResponse, error) {
	params, err := s.prepareUploadURLParams(country, region, spot, userEmail, "jpg")
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	imageKey := fmt.Sprintf(
		"%s/%s_%s.%s",
		params.keyPrefix,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		params.user.UUID,
		params.fileExt,
	)

	// Generate presigned URL valid for 15 minutes
	presignedURL, err := s.s3Storage.GeneratePresignedUploadURL(s.bucketName, imageKey, params.expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := currentTime.Add(params.expiration)

	return &model.PresignedUploadResponse{
		UploadURL: presignedURL,
		ImageKey:  imageKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *ReportService) GenerateVideoUploadURL(
	country, region, spot, userEmail string,
) (*model.VideoUploadResponse, error) {
	params, err := s.prepareUploadURLParams(country, region, spot, userEmail, "mp4")
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	videoKey := fmt.Sprintf(
		"%s/%s_%s.%s",
		params.keyPrefix,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		params.user.UUID,
		params.fileExt,
	)

	// Generate presigned URL valid for 15 minutes
	presignedURL, err := s.s3Storage.GeneratePresignedUploadURL(s.bucketName, videoKey, params.expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := currentTime.Add(params.expiration)

	return &model.VideoUploadResponse{
		UploadURL: presignedURL,
		VideoKey:  videoKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *ReportService) GetTodaysSurfReports(
	countryName, regionName, spotName string,
) ([]map[string]interface{}, error) {
	return s.GetSpotSurfReports(countryName, regionName, spotName, 1, nil)
}

// GetSpotSurfReports retrieves surf reports for a specific spot with pagination support.
// limit: maximum number of reports to return (0 for all).
// lastEvaluatedKey: for pagination, provide the last key from previous query.
func (s *ReportService) GetSpotSurfReports(
	countryName, regionName, spotName string,
	limit int,
	lastEvaluatedKey map[string]*dynamodb.AttributeValue,
) ([]map[string]interface{}, error) {
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)

	input := s.buildSpotReportsQueryInput(countryRegionSpot, limit, lastEvaluatedKey)
	result, err := s.dbStorage.Query(input)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %w", err)
	}

	reports, err := s.unmarshalSpotReports(result.Items)
	if err != nil {
		return nil, err
	}

	s.normalizeSpotReports(reports)
	return reports, nil
}

func (s *ReportService) GetReportImage(imageKey string) ([]byte, string, error) {
	imageData, err := s.s3Storage.GetObject(s.bucketName, imageKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %v", err)
	}

	// For now, assume JPEG content type
	// TODO: Implement proper content type detection
	contentType := "image/jpeg"

	return imageData, contentType, nil
}

func (s *ReportService) GetReportVideo(videoKey string) ([]byte, string, error) {
	videoData, err := s.s3Storage.GetObject(s.bucketName, videoKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read video data: %v", err)
	}

	// For now, assume MP4 content type
	// TODO: Implement proper content type detection
	contentType := "video/mp4"

	return videoData, contentType, nil
}

func (s *ReportService) GenerateVideoViewURL(videoKey string, userEmail string) (*model.VideoViewURLResponse, error) {
	if videoKey == "" {
		return nil, fmt.Errorf("video key is required")
	}

	user, err := s.userService.GetUserByEmail(userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.UUID == "" {
		return nil, fmt.Errorf("user does not have a UUID")
	}

	_, err = s.s3Storage.GetObject(s.bucketName, videoKey)
	if err != nil {
		return nil, fmt.Errorf("video not found or not accessible: %v", err)
	}

	// Video keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.mp4
	if !s.canUserAccessVideo(videoKey, user.UUID) {
		return nil, fmt.Errorf("access denied: you don't have permission to view this video")
	}

	expires := 1 * time.Hour
	viewURL, err := s.s3Storage.GeneratePresignedViewURL(s.bucketName, videoKey, expires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned view URL: %v", err)
	}

	expiresAt := time.Now().Add(expires)

	return &model.VideoViewURLResponse{
		ViewURL:   viewURL,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// SubmitSurfReportWithIOSValidation submits a surf report that has been validated
// using iOS Vision framework.
func (s *ReportService) SubmitSurfReportWithIOSValidation(
	report *model.ReportWithIOSValidation,
	userEmail string,
	userName string,
) error {
	user, err := s.getUserAndValidate(userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)
	mediaType := determineMediaType(report.ImageKey != "", report.VideoKey != "")

	item := s.createBaseReportItem(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, mediaType, report.IOSValidated,
	)
	addReportFieldsToItem(
		item, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	s3KeyReport, videoKeyReport := s.processIOSMediaKeys(report.ImageKey, report.VideoKey, item)

	if err := s.storeReport(item); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, videoKeyReport, mediaType, report.IOSValidated, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

func (s *ReportService) validateImageWithRekognition(imageData []byte) (bool, error) {
	if os.Getenv("GO_ENV") == constants.EnvDevelopment {
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
		return false, model.NewImageValidationError(err, "image analysis failed")
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
		return false, model.ErrImageNotSurfRelated
	}

	return false, model.ErrImageAnalysisFailed
}

func (s *ReportService) uploadImageToS3(imageData []byte, key string) (string, error) {
	err := s.s3Storage.PutObject(s.bucketName, key, imageData, "image/jpeg")
	if err != nil {
		return "", model.NewImageValidationError(err, "failed to upload image to S3")
	}

	return key, nil
}

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

func (s *ReportService) IsValidMessiness(messiness string) bool {
	return validation.IsValidMessiness(messiness)
}

func (s *ReportService) getSpotSubscribers(_ string, _ string, _ string) ([]string, error) {
	// TODO: Implement spot subscribers retrieval
	// For now, return empty list
	return []string{}, nil
}

func (s *ReportService) broadcastToUsers(subscribers []string, message interface{}) {
	// TODO: Implement user broadcasting
	// For now, just log the message
	log.Printf("Broadcasting message to %d subscribers: %v", len(subscribers), message)
}

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

func (s *ReportService) DeleteMediaFromS3(mediaKey string) error {
	if mediaKey == "" {
		return fmt.Errorf("media key is required")
	}

	log.Printf("🗑️ [CLEANUP] Deleting media: %s", mediaKey)
	err := s.s3Storage.DeleteObject(s.bucketName, mediaKey)
	if err != nil {
		log.Printf("❌ [CLEANUP] Failed to delete media %s: %v", mediaKey, err)
		return fmt.Errorf("failed to delete media from S3: %v", err)
	}

	log.Printf("✅ [CLEANUP] Successfully deleted media: %s", mediaKey)
	return nil
}

func (s *ReportService) canUserAccessVideo(videoKey, userUUID string) bool {
	// Video keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.mp4
	// We need to extract the UUID from the video key to verify ownership

	// Split the video key by "/" to get the parts
	parts := strings.Split(videoKey, "/")
	if len(parts) < 3 {
		log.Printf("Invalid video key format: %s", videoKey)
		return false
	}

	// Get the filename part (last part)
	filename := parts[len(parts)-1]

	// Remove the .mp4 extension
	if !strings.HasSuffix(filename, ".mp4") {
		log.Printf("Video key does not end with .mp4: %s", videoKey)
		return false
	}

	filenameWithoutExt := strings.TrimSuffix(filename, ".mp4")

	// Split by "_" to get timestamp and UUID
	fileParts := strings.Split(filenameWithoutExt, "_")
	if len(fileParts) < 2 {
		log.Printf("Invalid video key filename format: %s", filename)
		return false
	}

	// The UUID should be the last part after splitting by "_"
	videoUUID := fileParts[len(fileParts)-1]

	// Check if the UUID matches the user's UUID
	if videoUUID != userUUID {
		log.Printf("Video UUID %s does not match user UUID %s", videoUUID, userUUID)
		return false
	}

	return true
}

// GetSurfReportsWithSimilarBuoyData retrieves surf reports that had similar buoy conditions.
// It takes buoy data parameters (waveHeight, waveDirection, period), a specific buoy name,
// and optionally spot parameters. Returns a list of surf reports with similarity scores.
//
//nolint:gocyclo,funlen // Complex business logic with multiple conditional branches
func (s *ReportService) GetSurfReportsWithSimilarBuoyData(
	waveHeight float64,
	waveDirection float64,
	period float64,
	buoyName string,
	countryName string,
	regionName string,
	spotName string,
	daysBack int,
	maxResults int,
) ([]map[string]interface{}, error) {
	// Default values
	if daysBack == 0 {
		daysBack = 365 // Default to 1 year
	}
	if maxResults == 0 {
		maxResults = 20 // Default to 20 results
	}

	// Build spot filter
	var countryRegionSpot string
	if countryName != "" && regionName != "" && spotName != "" {
		countryRegionSpot = fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)
	}

	// Calculate cutoff time
	cutoffTime := time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)
	cutoffStr := cutoffTime.UTC().Format("2006-01-02T15:04:05Z")

	// Query surf reports
	var result *dynamodb.QueryOutput
	var scanResult *dynamodb.ScanOutput
	var err error

	if countryRegionSpot != "" {
		// Query for specific spot
		queryInput := &dynamodb.QueryInput{
			TableName:              aws.String("SurfReports"),
			KeyConditionExpression: aws.String("country_region_spot = :crs"),
			FilterExpression:       aws.String("#Time > :cutoff"),
			ExpressionAttributeNames: map[string]*string{
				"#Time": aws.String("Time"), // Escape reserved keyword
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":crs": {
					S: aws.String(countryRegionSpot),
				},
				":cutoff": {
					S: aws.String(cutoffStr),
				},
			},
			ScanIndexForward: aws.Bool(false), // Most recent first
		}

		result, err = s.dbStorage.Query(queryInput)
		if err != nil {
			return nil, fmt.Errorf("failed to query surf reports: %v", err)
		}
	} else {
		// Scan all reports (filtered by time)
		scanInput := &dynamodb.ScanInput{
			TableName:        aws.String("SurfReports"),
			FilterExpression: aws.String("#Time > :cutoff"),
			ExpressionAttributeNames: map[string]*string{
				"#Time": aws.String("Time"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":cutoff": {
					S: aws.String(cutoffStr),
				},
			},
			Limit: aws.Int64(int64(maxResults * 10)), // Get more reports to filter
		}

		scanResult, err = s.dbStorage.Scan(scanInput)
		if err != nil {
			return nil, fmt.Errorf("failed to scan surf reports: %v", err)
		}
	}

	// Unmarshal reports
	var reports []map[string]interface{}
	var items []map[string]*dynamodb.AttributeValue
	switch {
	case result != nil:
		items = result.Items
	case scanResult != nil:
		items = scanResult.Items
	default:
		return []map[string]interface{}{}, nil // No results
	}

	err = dynamodbattribute.UnmarshalListOfMaps(items, &reports)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}

	// For each report, get buoy data at that time and calculate similarity
	type reportWithSimilarity struct {
		report     map[string]interface{}
		similarity float64
	}

	var reportsWithSimilarity []reportWithSimilarity

	// Validate buoy name
	if buoyName == "" {
		return nil, fmt.Errorf("buoyName is required")
	}

	// Get the specified buoy location
	buoyLocations := s.getBuoyLocations()
	buoyLocation, ok := buoyLocations[buoyName]
	if !ok {
		return nil, fmt.Errorf("buoy %s not found", buoyName)
	}

	buoyLat, ok := buoyLocation["Latitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid latitude for buoy %s", buoyName)
	}

	buoyLon, ok := buoyLocation["Longitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid longitude for buoy %s", buoyName)
	}

	// Use only the specified buoy for data lookup
	buoyPriority := []string{buoyName}

	for _, report := range reports {
		// Parse report time
		timeStr, ok := report["Time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, err := parseReportTime(timeStr)
		if err != nil {
			log.Printf("Failed to parse report time %s: %v", timeStr, err)
			continue
		}

		// Calculate travel time if spot location is available
		var travelTimeHours float64
		var targetBuoyTime = reportTime

		// Try to get spot location from report
		// First try from report fields, then from query parameters
		var spotLat, spotLon float64

		// Extract spot location from report country_region_spot
		if countryRegionSpot, ok := report["country_region_spot"].(string); ok {
			// Format is "Country_Region_Spot"
			parts := strings.Split(countryRegionSpot, "_")
			if len(parts) == 3 {
				spotLoc, err := s.getSpotLocation(parts[0], parts[1], parts[2])
				if err == nil && spotLoc != nil {
					if lat, ok := spotLoc["Latitude"].(float64); ok {
						spotLat = lat
					}
					if lon, ok := spotLoc["Longitude"].(float64); ok {
						spotLon = lon
					}
				}
			}
		}

		// Fallback to query parameters if not found in report
		if spotLat == 0 && spotLon == 0 && countryName != "" && regionName != "" && spotName != "" {
			spotLoc, err := s.getSpotLocation(countryName, regionName, spotName)
			if err == nil && spotLoc != nil {
				if lat, ok := spotLoc["Latitude"].(float64); ok {
					spotLat = lat
				}
				if lon, ok := spotLoc["Longitude"].(float64); ok {
					spotLon = lon
				}
			}
		}

		// Calculate travel time based on swell direction and spot location
		if spotLat != 0 && spotLon != 0 {
			// Calculate bearing from buoy to spot
			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, spotLat, spotLon)

			// Check if spot is downwave from buoy (within reasonable angle of swell direction)
			// Swell direction is where waves are coming FROM, so we need to check if the spot
			// is in the direction the swell is traveling TO
			swellTravelDirection := math.Mod(waveDirection+180, 360) // Direction waves are traveling TO

			// Calculate angle difference between swell travel direction and bearing to spot
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff // Handle wraparound
			}

			// If angle difference is less than 45 degrees, spot is downwave - use travel time
			// If angle difference is greater, spot is not in swell path - minimal/no travel time
			if angleDiff < 45.0 {
				// Spot is downwave - calculate normal travel time
				distance := s.calculateDistance(buoyLat, buoyLon, spotLat, spotLon)

				// Calculate travel time using phase velocity
				// Phase velocity = 1.56 * sqrt(period) in m/s for deep water waves
				phaseVelocity := 1.56 * math.Sqrt(period) // m/s

				// Travel time in hours = distance (meters) / (phase_velocity * 3600)
				travelTimeHours = distance / (phaseVelocity * 3600)

				// Cap travel time between 1-8 hours (matching prediction service)
				if travelTimeHours > 8.0 {
					travelTimeHours = 8.0
				}
				if travelTimeHours < 1.0 {
					travelTimeHours = 1.0
				}

				// Look for buoy data at (report_time - travel_time)
				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			} else {
				// Spot is not directly downwave - use minimal travel time (or same time)
				// For perpendicular/non-direct swells, look at buoy data closer to report time
				targetBuoyTime = reportTime
				travelTimeHours = 0.0
			}
		}

		// Get buoy data at target time (accounting for travel time if calculated)
		buoyData := s.getBuoyDataAtTime(targetBuoyTime, buoyPriority)
		if buoyData == nil {
			continue // Skip if no buoy data found
		}

		// Calculate similarity
		similarity := s.calculateBuoyConditionSimilarity(
			waveHeight, waveDirection, period,
			buoyData,
		)

		// Only include reports with similarity > 0.7
		if similarity > 0.7 {
			// Remove sensitive fields
			delete(report, "UserEmail")

			// Add similarity score and buoy data info
			report["similarity"] = similarity
			report["buoy_wave_height"] = buoyData["WaveHeight"]
			report["buoy_wave_direction"] = buoyData["MeanWaveDirection"]
			report["buoy_period"] = buoyData["MaxPeriod"]
			if travelTimeHours > 0 {
				report["travel_time_hours"] = travelTimeHours
			}

			reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
				report:     report,
				similarity: similarity,
			})
		}
	}

	// Sort by similarity (highest first)
	for i := 0; i < len(reportsWithSimilarity); i++ {
		for j := i + 1; j < len(reportsWithSimilarity); j++ {
			if reportsWithSimilarity[i].similarity < reportsWithSimilarity[j].similarity {
				reportsWithSimilarity[i], reportsWithSimilarity[j] = reportsWithSimilarity[j], reportsWithSimilarity[i]
			}
		}
	}

	// Limit results
	if len(reportsWithSimilarity) > maxResults {
		reportsWithSimilarity = reportsWithSimilarity[:maxResults]
	}

	// Convert back to map slice
	var finalReports []map[string]interface{}
	for _, rws := range reportsWithSimilarity {
		finalReports = append(finalReports, rws.report)
	}

	return finalReports, nil
}

func (s *ReportService) getBuoyDataAtTime(targetTime time.Time, buoyPriority []string) map[string]interface{} {
	// Look for data within 6 hours of target time
	startTime := targetTime.Add(-6 * time.Hour)
	endTime := targetTime.Add(6 * time.Hour)
	startStr := startTime.UTC().Format("2006-01-02T15:04:05Z")
	endStr := endTime.UTC().Format("2006-01-02T15:04:05Z")

	// Try multiple buoys in order of priority
	for _, buoyName := range buoyPriority {
		regionBuoy := fmt.Sprintf("Ireland_%s", buoyName)

		input := &dynamodb.QueryInput{
			TableName:              aws.String("BuoyData"),
			KeyConditionExpression: aws.String("region_buoy = :rb AND dataDateTime BETWEEN :start AND :end"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":rb": {
					S: aws.String(regionBuoy),
				},
				":start": {
					S: aws.String(startStr),
				},
				":end": {
					S: aws.String(endStr),
				},
			},
			ScanIndexForward: aws.Bool(true),
			Limit:            aws.Int64(1),
		}

		result, err := s.dbStorage.Query(input)
		if err != nil {
			log.Printf("Error querying buoy data for %s at %s: %v", buoyName, targetTime.Format(time.RFC3339), err)
			continue
		}

		if len(result.Items) > 0 {
			var buoyData map[string]interface{}
			err := dynamodbattribute.UnmarshalMap(result.Items[0], &buoyData)
			if err == nil {
				return buoyData
			}
		}
	}

	return nil
}

func (s *ReportService) calculateBuoyConditionSimilarity(
	predHeight float64,
	predDirection float64,
	predPeriod float64,
	buoyData map[string]interface{},
) float64 {
	// Extract buoy measurements
	buoyHeight := 0.0
	buoyDirection := 0.0
	buoyPeriod := 0.0

	if h, ok := buoyData["WaveHeight"].(float64); ok {
		buoyHeight = h
	} else if hStr, ok := buoyData["WaveHeight"].(string); ok {
		if h, err := strconv.ParseFloat(hStr, 64); err == nil {
			buoyHeight = h
		}
	}

	if d, ok := buoyData["MeanWaveDirection"].(float64); ok {
		buoyDirection = d
	} else if dStr, ok := buoyData["MeanWaveDirection"].(string); ok {
		if d, err := strconv.ParseFloat(dStr, 64); err == nil {
			buoyDirection = d
		}
	}

	if p, ok := buoyData["MaxPeriod"].(float64); ok {
		buoyPeriod = p
	} else if pStr, ok := buoyData["MaxPeriod"].(string); ok {
		if p, err := strconv.ParseFloat(pStr, 64); err == nil {
			buoyPeriod = p
		}
	}

	// Calculate height similarity (within 50% is considered similar)
	maxHeight := predHeight
	if buoyHeight > maxHeight {
		maxHeight = buoyHeight
	}
	if maxHeight < 0.1 {
		maxHeight = 0.1 // Avoid division by zero
	}
	heightDiff := absFloat(predHeight-buoyHeight) / maxHeight
	heightSimilarity := maxFloat(0.0, 1.0-heightDiff/0.5)

	// Calculate direction similarity (within 30 degrees is considered similar)
	directionDiff := absFloat(predDirection - buoyDirection)
	if directionDiff > 180 {
		directionDiff = 360 - directionDiff // Handle wraparound
	}
	directionSimilarity := maxFloat(0.0, 1.0-directionDiff/30.0)

	// Calculate period similarity (within 2 seconds is considered similar)
	periodDiff := absFloat(predPeriod - buoyPeriod)
	periodSimilarity := maxFloat(0.0, 1.0-periodDiff/2.0)

	// Combined similarity (weighted average)
	// Height and direction are more important than period
	return 0.5*heightSimilarity + 0.4*directionSimilarity + 0.1*periodSimilarity
}

// Helper functions
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func parseReportTime(timeStr string) (time.Time, error) {
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try Go time format
	if t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", timeStr); err == nil {
		return t, nil
	}

	// Try simplified format
	if t, err := time.Parse("2006-01-02 15:04:05 -0700", timeStr); err == nil {
		return t, nil
	}

	// Try ISO format
	if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}

func (s *ReportService) getSpotLocation(countryName, regionName, spotName string) (map[string]interface{}, error) {
	locationKey := fmt.Sprintf("%s/%s/%s", countryName, regionName, spotName)

	input := &dynamodb.QueryInput{
		TableName:              aws.String("LocationData"),
		KeyConditionExpression: aws.String("country_region_spot = :location"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":location": {
				S: aws.String(locationKey),
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

	var location map[string]interface{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &location)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal location: %v", err)
	}

	return location, nil
}

func (s *ReportService) getBuoyLocations() map[string]map[string]interface{} {
	input := &dynamodb.ScanInput{
		TableName: aws.String("BuoyLocations"),
	}

	result, err := s.dbStorage.Scan(input)
	if err != nil {
		log.Printf("Error scanning buoy locations: %v", err)
		return make(map[string]map[string]interface{})
	}

	buoyLocations := make(map[string]map[string]interface{})
	for _, item := range result.Items {
		var buoy map[string]interface{}
		err := dynamodbattribute.UnmarshalMap(item, &buoy)
		if err != nil {
			continue
		}

		// Extract buoy name from region_buoy (format: "Ireland_M4")
		if regionBuoy, ok := buoy["region_buoy"].(string); ok {
			parts := strings.Split(regionBuoy, "_")
			if len(parts) > 1 {
				buoyName := parts[len(parts)-1] // Get "M4" from "Ireland_M4"
				buoyLocations[buoyName] = buoy
			}
		} else if name, ok := buoy["Name"].(string); ok {
			// Fallback to Name field if region_buoy not available
			buoyLocations[name] = buoy
		}
	}

	return buoyLocations
}

func (s *ReportService) getNearestBuoys(spotLat, spotLon float64, numBuoys int) []map[string]interface{} {
	if numBuoys <= 0 {
		numBuoys = 2 // Default to 2 nearest buoys
	}

	allBuoys := s.getBuoyLocations()
	type buoyWithDistance struct {
		buoy     map[string]interface{}
		name     string
		distance float64
	}

	var buoysWithDistance []buoyWithDistance

	for name, buoy := range allBuoys {
		buoyLat, ok1 := buoy["Latitude"].(float64)
		buoyLon, ok2 := buoy["Longitude"].(float64)

		if !ok1 || !ok2 {
			continue // Skip if coordinates are missing or invalid
		}

		distance := s.calculateDistance(spotLat, spotLon, buoyLat, buoyLon)

		// Create a copy of the buoy map with the name added
		buoyCopy := make(map[string]interface{})
		for k, v := range buoy {
			buoyCopy[k] = v
		}
		buoyCopy["Name"] = name

		buoysWithDistance = append(buoysWithDistance, buoyWithDistance{
			buoy:     buoyCopy,
			name:     name,
			distance: distance,
		})
	}

	// Sort by distance (closest first)
	for i := 0; i < len(buoysWithDistance); i++ {
		for j := i + 1; j < len(buoysWithDistance); j++ {
			if buoysWithDistance[i].distance > buoysWithDistance[j].distance {
				buoysWithDistance[i], buoysWithDistance[j] = buoysWithDistance[j], buoysWithDistance[i]
			}
		}
	}

	result := []map[string]interface{}{}
	for i := 0; i < numBuoys && i < len(buoysWithDistance); i++ {
		result = append(result, buoysWithDistance[i].buoy)
	}

	return result
}

// Returns distance in meters
func (s *ReportService) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (s *ReportService) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(deltaLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLon)

	bearing := math.Atan2(y, x)
	bearingDegrees := bearing * 180 / math.Pi

	// Convert to 0-360 range
	bearingDegrees = (bearingDegrees + 360)
	return math.Mod(bearingDegrees, 360)
}

func (s *ReportService) getCurrentBuoyData(buoyName string) map[string]interface{} {
	// Start from current time rounded down to the nearest hour
	now := time.Now()
	currentTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// Search backwards up to 12 hours to find the most recent data
	for i := 0; i < 12; i++ {
		searchTime := currentTime.Add(time.Duration(-i) * time.Hour)
		dateStr := searchTime.UTC().Format("2006-01-02T15:00:00Z")

		regionBuoy := fmt.Sprintf("Ireland_%s", buoyName)
		input := &dynamodb.QueryInput{
			TableName:              aws.String("BuoyData"),
			KeyConditionExpression: aws.String("region_buoy = :rb AND dataDateTime = :dt"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":rb": {
					S: aws.String(regionBuoy),
				},
				":dt": {
					S: aws.String(dateStr),
				},
			},
			ScanIndexForward: aws.Bool(false),
			Limit:            aws.Int64(1),
		}

		result, err := s.dbStorage.Query(input)
		if err != nil {
			log.Printf("Error querying current buoy data for %s at %s: %v", buoyName, dateStr, err)
			continue
		}

		if len(result.Items) > 0 {
			var buoyData map[string]interface{}
			err := dynamodbattribute.UnmarshalMap(result.Items[0], &buoyData)
			if err == nil {
				return buoyData
			}
		}
	}

	return nil
}

func (s *ReportService) getCurrentWindConditions(
	countryName, regionName, spotName string,
) (windSpeed float64, windDirection float64, err error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	result, err := s.queryCurrentForecast(spotID)
	if err != nil {
		return 0, 0, err
	}

	if len(result.Items) == 0 {
		result, err = s.queryHistoricalForecast(spotID)
		if err != nil || len(result.Items) == 0 {
			return 0, 0, fmt.Errorf("no forecast data found for spot")
		}
	}

	forecast, err := s.unmarshalForecast(result.Items[0])
	if err != nil {
		return 0, 0, err
	}

	return s.extractWindData(forecast)
}

func (s *ReportService) getForecastDataAtTime(
	countryName, regionName, spotName string, targetTime time.Time,
) map[string]interface{} {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	targetEpoch := targetTime.Unix()

	// Search within ±3 hours window
	startEpoch := targetEpoch - 3*3600
	endEpoch := targetEpoch + 3*3600

	input := &dynamodb.QueryInput{
		TableName:              aws.String("SpotForecastData"),
		KeyConditionExpression: aws.String("spot_id = :spot_id AND forecast_timestamp BETWEEN :start_time AND :end_time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spot_id": {
				S: aws.String(spotID),
			},
			":start_time": {
				S: aws.String(fmt.Sprintf("%d", startEpoch)),
			},
			":end_time": {
				S: aws.String(fmt.Sprintf("%d", endEpoch)),
			},
		},
		ScanIndexForward: aws.Bool(true),
		Limit:            aws.Int64(1),
	}

	result, err := s.dbStorage.Query(input)
	if err != nil {
		log.Printf("Error querying forecast data for %s at %s: %v", spotID, targetTime.Format(time.RFC3339), err)
		return nil
	}

	if len(result.Items) == 0 {
		return nil
	}

	var forecast map[string]interface{}
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &forecast)
	if err != nil {
		log.Printf("Error unmarshaling forecast data: %v", err)
		return nil
	}

	return forecast
}

func (s *ReportService) calculateWindSimilarity(
	currentSpeed, currentDirection float64,
	historicalSpeed, historicalDirection float64,
) float64 {
	// Calculate wind speed similarity (within 20% or 5 m/s, whichever is larger)
	maxSpeed := currentSpeed
	if historicalSpeed > maxSpeed {
		maxSpeed = historicalSpeed
	}
	if maxSpeed < 1.0 {
		maxSpeed = 1.0 // Avoid division by zero
	}

	speedDiff := absFloat(currentSpeed - historicalSpeed)
	speedThreshold := maxFloat(maxSpeed*0.2, 5.0) // 20% or 5 m/s
	speedSimilarity := maxFloat(0.0, 1.0-speedDiff/speedThreshold)

	// Calculate wind direction similarity (within 30 degrees)
	directionDiff := absFloat(currentDirection - historicalDirection)
	if directionDiff > 180 {
		directionDiff = 360 - directionDiff // Handle wraparound
	}
	directionSimilarity := maxFloat(0.0, 1.0-directionDiff/30.0)

	// Combined similarity (equal weight for speed and direction)
	return 0.5*speedSimilarity + 0.5*directionSimilarity
}

// GetSurfReportsWithMatchingConditions retrieves surf reports for a spot where:
// 1. Buoy data at the report time (accounting for travel time) matches current buoy data from nearest buoys
// 2. Wind conditions from forecast data at the report time are similar to current wind conditions
//
//nolint:gocyclo,funlen // Complex business logic with multiple conditional branches
func (s *ReportService) GetSurfReportsWithMatchingConditions(
	countryName string,
	regionName string,
	spotName string,
	daysBack int,
	maxResults int,
) ([]map[string]interface{}, error) {
	// Default values
	if daysBack == 0 {
		daysBack = 365 // Default to 1 year
	}
	if maxResults == 0 {
		maxResults = 20 // Default to 20 results
	}

	// Step 1: Get spot location
	spotLoc, err := s.getSpotLocation(countryName, regionName, spotName)
	if err != nil {
		return nil, fmt.Errorf("failed to get spot location: %v", err)
	}

	spotLat, ok := spotLoc["Latitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid latitude for spot")
	}

	spotLon, ok := spotLoc["Longitude"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid longitude for spot")
	}

	// Step 2: Find 2 nearest buoys
	nearestBuoys := s.getNearestBuoys(spotLat, spotLon, 2)
	if len(nearestBuoys) == 0 {
		return nil, fmt.Errorf("no buoys found")
	}

	// Step 3: Get current buoy data for all nearest buoys
	type buoyData struct {
		location      map[string]interface{}
		currentData   map[string]interface{}
		name          string
		waveHeight    float64
		waveDirection float64
		period        float64
	}

	buoyDataList := []buoyData{}
	for _, buoy := range nearestBuoys {
		buoyName, ok := buoy["Name"].(string)
		if !ok {
			continue
		}

		currentBuoyData := s.getCurrentBuoyData(buoyName)
		if currentBuoyData == nil {
			log.Printf("Warning: No current buoy data found for buoy %s", buoyName)
			continue
		}

		waveHeight := 0.0
		waveDirection := 0.0
		period := 0.0

		if h, ok := currentBuoyData["WaveHeight"].(float64); ok {
			waveHeight = h
		} else if hStr, ok := currentBuoyData["WaveHeight"].(string); ok {
			if h, parseErr := strconv.ParseFloat(hStr, 64); parseErr == nil {
				waveHeight = h
			}
		}

		if d, ok := currentBuoyData["MeanWaveDirection"].(float64); ok {
			waveDirection = d
		} else if dStr, ok := currentBuoyData["MeanWaveDirection"].(string); ok {
			if d, parseErr := strconv.ParseFloat(dStr, 64); parseErr == nil {
				waveDirection = d
			}
		}

		if p, ok := currentBuoyData["MaxPeriod"].(float64); ok {
			period = p
		} else if pStr, ok := currentBuoyData["MaxPeriod"].(string); ok {
			if p, parseErr := strconv.ParseFloat(pStr, 64); parseErr == nil {
				period = p
			}
		}

		// Will be accessed from location map later
		if _, ok1 := buoy["Latitude"].(float64); !ok1 {
			continue
		}
		if _, ok2 := buoy["Longitude"].(float64); !ok2 {
			continue
		}

		buoyDataList = append(buoyDataList, buoyData{
			name:          buoyName,
			location:      buoy,
			currentData:   currentBuoyData,
			waveHeight:    waveHeight,
			waveDirection: waveDirection,
			period:        period,
		})
	}

	if len(buoyDataList) == 0 {
		return nil, fmt.Errorf("no current buoy data found for any nearest buoy")
	}

	// Step 4: Get current wind conditions
	currentWindSpeed, currentWindDirection, err := s.getCurrentWindConditions(countryName, regionName, spotName)
	if err != nil {
		log.Printf("Warning: Could not get current wind conditions: %v", err)
		// Continue without wind matching if we can't get current wind data
		currentWindSpeed = 0
		currentWindDirection = 0
	}

	// Step 5: Query historical surf reports
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", countryName, regionName, spotName)
	cutoffTime := time.Now().Add(-time.Duration(daysBack) * 24 * time.Hour)
	cutoffStr := cutoffTime.UTC().Format("2006-01-02T15:04:05Z")

	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("SurfReports"),
		KeyConditionExpression: aws.String("country_region_spot = :crs"),
		FilterExpression:       aws.String("#Time > :cutoff"),
		ExpressionAttributeNames: map[string]*string{
			"#Time": aws.String("Time"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":crs": {
				S: aws.String(countryRegionSpot),
			},
			":cutoff": {
				S: aws.String(cutoffStr),
			},
		},
		ScanIndexForward: aws.Bool(false), // Most recent first
	}

	result, err := s.dbStorage.Query(queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query surf reports: %v", err)
	}

	var reports []map[string]interface{}
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &reports)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal reports: %v", err)
	}

	// Step 6: Process each report and calculate similarity
	type reportWithSimilarity struct {
		report             map[string]interface{}
		matchedBuoy        string
		buoySimilarity     float64
		windSimilarity     float64
		combinedSimilarity float64
		travelTimeHours    float64
	}

	var reportsWithSimilarity []reportWithSimilarity

	for _, report := range reports {
		// Parse report time
		timeStr, ok := report["Time"].(string)
		if !ok || timeStr == "" {
			continue
		}

		reportTime, err := parseReportTime(timeStr)
		if err != nil {
			log.Printf("Failed to parse report time %s: %v", timeStr, err)
			continue
		}

		// Try each buoy and find the best match
		bestMatch := struct {
			historicalData map[string]interface{}
			buoyName       string
			similarity     float64
			travelTime     float64
		}{
			similarity: 0.0,
		}

		for _, buoyInfo := range buoyDataList {
			buoyLat, ok1 := buoyInfo.location["Latitude"].(float64)
			buoyLon, ok2 := buoyInfo.location["Longitude"].(float64)
			if !ok1 || !ok2 {
				continue
			}

			// Calculate travel time independently for this buoy
			var travelTimeHours float64
			targetBuoyTime := reportTime

			// Calculate bearing from buoy to spot
			bearingToSpot := s.calculateBearing(buoyLat, buoyLon, spotLat, spotLon)

			// Check if spot is downwave from buoy
			swellTravelDirection := math.Mod(buoyInfo.waveDirection+180, 360) // Direction waves are traveling TO
			angleDiff := math.Abs(swellTravelDirection - bearingToSpot)
			if angleDiff > 180 {
				angleDiff = 360 - angleDiff // Handle wraparound
			}

			if angleDiff < 45.0 {
				// Spot is downwave - calculate travel time
				distance := s.calculateDistance(buoyLat, buoyLon, spotLat, spotLon)
				phaseVelocity := 1.56 * math.Sqrt(buoyInfo.period) // m/s
				travelTimeHours = distance / (phaseVelocity * 3600)

				// Cap travel time between 1-8 hours
				if travelTimeHours > 8.0 {
					travelTimeHours = 8.0
				}
				if travelTimeHours < 1.0 {
					travelTimeHours = 1.0
				}

				targetBuoyTime = reportTime.Add(-time.Duration(travelTimeHours) * time.Hour)
			} else {
				// Spot is not directly downwave (targetBuoyTime already set to reportTime)
				travelTimeHours = 0.0
			}

			// Get historical buoy data at target time
			historicalBuoyData := s.getBuoyDataAtTime(targetBuoyTime, []string{buoyInfo.name})
			if historicalBuoyData == nil {
				continue // Skip if no buoy data found
			}

			// Calculate buoy similarity
			buoySimilarity := s.calculateBuoyConditionSimilarity(
				buoyInfo.waveHeight, buoyInfo.waveDirection, buoyInfo.period,
				historicalBuoyData,
			)

			// Track the best matching buoy
			if buoySimilarity > bestMatch.similarity {
				bestMatch.buoyName = buoyInfo.name
				bestMatch.similarity = buoySimilarity
				bestMatch.travelTime = travelTimeHours
				bestMatch.historicalData = historicalBuoyData
			}
		}

		// Only continue if buoy similarity is high enough
		if bestMatch.similarity < 0.7 {
			continue
		}

		// Get historical forecast data at report time
		historicalForecast := s.getForecastDataAtTime(countryName, regionName, spotName, reportTime)
		if historicalForecast == nil {
			// If no forecast data, skip wind matching but still include if buoy matches
			continue
		}

		// Extract historical wind data
		data, ok := historicalForecast["data"].(map[string]interface{})
		if !ok {
			continue
		}

		historicalWindSpeed := 0.0
		historicalWindDirection := 0.0

		if ws, ok := data["windSpeed"].(float64); ok {
			historicalWindSpeed = ws
		} else if wsStr, ok := data["windSpeed"].(string); ok {
			if ws, err := strconv.ParseFloat(wsStr, 64); err == nil {
				historicalWindSpeed = ws
			}
		}

		if wd, ok := data["windDirection"].(float64); ok {
			historicalWindDirection = wd
		} else if wdStr, ok := data["windDirection"].(string); ok {
			if wd, err := strconv.ParseFloat(wdStr, 64); err == nil {
				historicalWindDirection = wd
			}
		}

		// Calculate wind similarity
		var windSimilarity float64
		if currentWindSpeed > 0 || currentWindDirection > 0 {
			windSimilarity = s.calculateWindSimilarity(
				currentWindSpeed, currentWindDirection,
				historicalWindSpeed, historicalWindDirection,
			)
		} else {
			// If we don't have current wind data, skip wind matching
			windSimilarity = 1.0 // Neutral score
		}

		// Only include if wind similarity is reasonable (>= 0.5)
		if windSimilarity < 0.5 {
			continue
		}

		// Calculate combined similarity (70% buoy, 30% wind)
		combinedSimilarity := 0.7*bestMatch.similarity + 0.3*windSimilarity

		// Remove sensitive fields
		delete(report, "UserEmail")

		// Add similarity scores and metadata
		report["buoy_similarity"] = bestMatch.similarity
		report["wind_similarity"] = windSimilarity
		report["combined_similarity"] = combinedSimilarity
		report["matched_buoy"] = bestMatch.buoyName
		report["historical_buoy_wave_height"] = bestMatch.historicalData["WaveHeight"]
		report["historical_buoy_wave_direction"] = bestMatch.historicalData["MeanWaveDirection"]
		report["historical_buoy_period"] = bestMatch.historicalData["MaxPeriod"]
		report["historical_wind_speed"] = historicalWindSpeed
		report["historical_wind_direction"] = historicalWindDirection
		if bestMatch.travelTime > 0 {
			report["travel_time_hours"] = bestMatch.travelTime
		}

		reportsWithSimilarity = append(reportsWithSimilarity, reportWithSimilarity{
			report:             report,
			buoySimilarity:     bestMatch.similarity,
			windSimilarity:     windSimilarity,
			combinedSimilarity: combinedSimilarity,
			matchedBuoy:        bestMatch.buoyName,
			travelTimeHours:    bestMatch.travelTime,
		})
	}

	// Sort by combined similarity (highest first)
	for i := 0; i < len(reportsWithSimilarity); i++ {
		for j := i + 1; j < len(reportsWithSimilarity); j++ {
			if reportsWithSimilarity[i].combinedSimilarity < reportsWithSimilarity[j].combinedSimilarity {
				reportsWithSimilarity[i], reportsWithSimilarity[j] = reportsWithSimilarity[j], reportsWithSimilarity[i]
			}
		}
	}

	// Limit results
	if len(reportsWithSimilarity) > maxResults {
		reportsWithSimilarity = reportsWithSimilarity[:maxResults]
	}

	// Convert back to map slice
	// Initialize as empty slice (not nil) to ensure JSON serialization returns [] instead of null
	finalReports := make([]map[string]interface{}, 0)
	for _, rws := range reportsWithSimilarity {
		finalReports = append(finalReports, rws.report)
	}

	return finalReports, nil
}
