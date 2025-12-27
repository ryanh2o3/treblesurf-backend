package controller

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"treblesurf-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// This controller uses the shared service registry

// handleImageError handles image-related errors and sends appropriate HTTP responses
func handleImageError(c *gin.Context, err error, logPrefix string) {
	log.Printf("%s: %v", logPrefix, err)

	// Handle ImageValidationError type
	var imageValidationErr *model.ImageValidationError
	if errors.As(err, &imageValidationErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Image validation failed",
			"message": imageValidationErr.Error(),
			"help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. " +
				"The image should be clear and focused on surf conditions.",
		})
		return
	}

	// Handle specific error types
	switch {
	case errors.Is(err, model.ErrImageNotSurfRelated):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Image not surf-related",
			"message": "The image does not appear to show surf conditions",
			"help": "Please upload a photo that clearly shows the ocean, waves, beach, or coastline.",
		})
	case errors.Is(err, model.ErrInvalidImageData):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid image data",
			"message": "The image data provided is not in a valid format",
			"help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) " +
				"and that the image data is properly encoded.",
		})
	case errors.Is(err, model.ErrImageUploadFailed):
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Image upload failed",
			"message": "Failed to upload the image to storage",
			"help": "Please try again in a moment. If the problem persists, contact support.",
		})
	case errors.Is(err, model.ErrImageRetrievalFailed):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Image not found",
			"message": "The uploaded image could not be found or accessed",
			"help": "Please try uploading your image again. If the problem persists, contact support.",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
	}
}

// SubmitCurrentSurfReport handles surf report submission
func SubmitCurrentSurfReport(c *gin.Context) {
	logRequestDetails(c, "Report Submission Request")
	log.Print("start of submit report")

	var report model.ReportWithImage
	if err := c.BindJSON(&report); err != nil {
		handleReportBindingError(c, err)
		return
	}

	log.Printf("Report data received: Country=%s, Region=%s, Spot=%s, SurfSize=%s",
		report.Country, report.Region, report.Spot, report.SurfSize)

	if !validateReportLocation(c, report.Country, report.Region, report.Spot) {
		return
	}

	if !validateReportFields(c, &report) {
		return
	}

	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return
	}

	log.Printf("User email from context: %s", email)
	user, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Failed to fetch user information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User information error",
			"message": "Unable to retrieve your user profile",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	if err := ReportService.SubmitSurfReport(&report, email, user.GivenName); err != nil {
		handleImageError(c, err, "Failed to submit report")
		return
	}

	log.Printf("Report submitted successfully for user: %s", email)
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// SubmitSurfReportWithS3Image handles surf report submission with pre-uploaded S3 image
func SubmitSurfReportWithS3Image(c *gin.Context) {
	logRequestDetails(c, "S3 Image Report Submission Request")
	log.Print("start of submit S3 image report")

	var report model.ReportWithS3Image
	if err := c.BindJSON(&report); err != nil {
		handleReportBindingError(c, err)
		return
	}

	log.Printf("S3 Image Report data received: Country=%s, Region=%s, Spot=%s, ImageKey=%s",
		report.Country, report.Region, report.Spot, report.ImageKey)

	if !validateReportLocation(c, report.Country, report.Region, report.Spot) {
		return
	}

	if !validateS3ReportFields(c, &report) {
		return
	}

	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return
	}

	log.Printf("User email from context: %s", email)
	user, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Failed to fetch user information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User information error",
			"message": "Unable to retrieve your user profile",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	if err := ReportService.SubmitSurfReportWithS3Image(&report, email, user.GivenName); err != nil {
		handleImageError(c, err, "Failed to submit S3 image report")
		return
	}

	log.Printf("S3 Image Report submitted successfully for user: %s", email)
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// GenerateImageUploadURL generates a presigned URL for uploading an image to S3
func GenerateImageUploadURL(c *gin.Context) {
	log.Printf("=== Generate Image Upload URL Request ===")

	// Get query parameters
	country := c.Query("country")
	region := c.Query("region")
	spot := c.Query("spot")

	if country == "" || region == "" || spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help": "Please provide all required location parameters in your request.",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to generate an upload URL",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("Generating upload URL for: Country=%s, Region=%s, Spot=%s, User=%s",
		country, region, spot, email.(string))

	// Generate the presigned URL
	response, err := ReportService.GenerateImageUploadURL(country, region, spot, email.(string))
	if err != nil {
		log.Printf("Failed to generate upload URL: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to generate presigned URL") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate upload URL",
				"message": "Unable to create a secure upload link for your image",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate upload URL"})
		return
	}

	log.Printf("Upload URL generated successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, response)
}

// RetrieveTodaysSurfReports retrieves the most recent surf report for a specific spot (legacy endpoint)
func RetrieveTodaysSurfReports(c *gin.Context) {
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help": "Please provide all required location parameters in your request.",
		})
		return
	}

	reports, err := ReportService.GetTodaysSurfReports(countryName, regionName, spotName)
	if err != nil {
		log.Printf("Failed to retrieve surf reports: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to query reports") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve reports",
				"message": "Unable to fetch surf reports from the database",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// GetAllSpotSurfReports retrieves all surf reports for a specific spot with pagination support
func GetAllSpotSurfReports(c *gin.Context) {
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	limitStr := c.DefaultQuery("limit", "50") // Default to 50 reports per page

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help": "Please provide all required location parameters in your request.",
		})
		return
	}

	// Parse limit parameter
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit parameter",
			"message": "Limit must be a positive integer or 0 for all reports",
		})
		return
	}

	// For simplicity, we'll skip pagination token handling for now
	// In a production system, you'd want to handle pagination tokens
	reports, err := ReportService.GetSpotSurfReports(countryName, regionName, spotName, limit, nil)
	if err != nil {
		log.Printf("Failed to retrieve surf reports: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to query reports") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve reports",
				"message": "Unable to fetch surf reports from the database",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// GetSurfReportsWithSimilarBuoyData retrieves surf reports that had similar buoy conditions
func GetSurfReportsWithSimilarBuoyData(c *gin.Context) {
	waveHeight, waveDirection, period, buoyName, ok := parseBuoyQueryParams(c)
	if !ok {
		return
	}

	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	daysBack := parseOptionalIntParam(c, "daysBack", 365)
	maxResults := parseOptionalIntParam(c, "maxResults", 20)

	reports, err := ReportService.GetSurfReportsWithSimilarBuoyData(
		waveHeight,
		waveDirection,
		period,
		buoyName,
		countryName,
		regionName,
		spotName,
		daysBack,
		maxResults,
	)
	if err != nil {
		log.Printf("Failed to retrieve surf reports with similar buoy data: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve reports",
			"message": "Unable to fetch surf reports with similar buoy conditions",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// GetSurfReportsWithMatchingConditions retrieves surf reports for a spot where:
// 1. Buoy data at the report time (accounting for travel time) matches current buoy data from nearest buoys
// 2. Wind conditions from forecast data at the report time are similar to current wind conditions
func GetSurfReportsWithMatchingConditions(c *gin.Context) {
	// Parse query parameters
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	daysBackStr := c.DefaultQuery("daysBack", "365")
	maxResultsStr := c.DefaultQuery("maxResults", "20")

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
			"message": "country, region, and spot parameters are required",
			"help": "Please provide all required location parameters (country, region, spot) in your request.",
		})
		return
	}

	// Parse optional parameters
	daysBack, err := strconv.Atoi(daysBackStr)
	if err != nil {
		daysBack = 365 // Default to 1 year
	}

	maxResults, err := strconv.Atoi(maxResultsStr)
	if err != nil {
		maxResults = 20 // Default to 20 results
	}

	// Get reports with matching conditions
	reports, err := ReportService.GetSurfReportsWithMatchingConditions(
		countryName,
		regionName,
		spotName,
		daysBack,
		maxResults,
	)
	if err != nil {
		log.Printf("Failed to retrieve surf reports with matching conditions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve reports",
			"message": "Unable to fetch surf reports with matching conditions",
			"help": "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	c.JSON(http.StatusOK, reports)
}

// GetReportImage retrieves a report image from S3
func GetReportImage(c *gin.Context) {
	// Get the image key from the query parameter
	imageKey := c.Query("key")
	if imageKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing image key",
			"message": "Image key parameter is required",
			"help": "Please provide the image key in your request.",
		})
		return
	}

	// Get the image from the report service
	imageData, contentType, err := ReportService.GetReportImage(imageKey)
	if err != nil {
		log.Printf("Error getting image: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to read image data") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Image not found",
				"message": "The requested image could not be found or accessed",
				"help": "The image may have been deleted or the image key may be incorrect.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve image"})
		return
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// Return the base64-encoded image
	c.JSON(http.StatusOK, gin.H{
		"imageData":   base64Data,
		"contentType": contentType,
	})
}

// GenerateVideoUploadURL generates a presigned URL for uploading a video to S3
func GenerateVideoUploadURL(c *gin.Context) {
	log.Printf("=== Generate Video Upload URL Request ===")

	// Get query parameters
	country := c.Query("country")
	region := c.Query("region")
	spot := c.Query("spot")

	if country == "" || region == "" || spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help": "Please provide all required location parameters in your request.",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to generate an upload URL",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("Generating video upload URL for: Country=%s, Region=%s, Spot=%s, User=%s",
		country, region, spot, email.(string))

	// Generate the presigned URL
	response, err := ReportService.GenerateVideoUploadURL(country, region, spot, email.(string))
	if err != nil {
		log.Printf("Failed to generate video upload URL: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to generate presigned URL") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate upload URL",
				"message": "Unable to create a secure upload link for your video",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video upload URL"})
		return
	}

	log.Printf("Video upload URL generated successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, response)
}

// GetReportVideo retrieves a report video from S3
func GetReportVideo(c *gin.Context) {
	// Get the video key from the query parameter
	videoKey := c.Query("key")
	if videoKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing video key",
			"message": "Video key parameter is required",
			"help": "Please provide the video key in your request.",
		})
		return
	}

	// Get the video from the report service
	videoData, contentType, err := ReportService.GetReportVideo(videoKey)
	if err != nil {
		log.Printf("Error getting video: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "failed to read video data") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Video not found",
				"message": "The requested video could not be found or accessed",
				"help": "The video may have been deleted or the video key may be incorrect.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video"})
		return
	}

	// Convert to base64
	base64Data := base64.StdEncoding.EncodeToString(videoData)

	// Return the base64-encoded video
	c.JSON(http.StatusOK, gin.H{
		"videoData":   base64Data,
		"contentType": contentType,
	})
}

// GenerateVideoViewURL generates a presigned URL for viewing a video
func GenerateVideoViewURL(c *gin.Context) {
	log.Printf("=== Generate Video View URL Request ===")

	videoKey := c.Query("key")
	if videoKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing video key",
			"message": "Video key parameter is required",
			"help":    "Please provide the video key in your request.",
		})
		return
	}

	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return
	}

	log.Printf("Generating video view URL for key: %s, user: %s", videoKey, email)
	response, err := ReportService.GenerateVideoViewURL(videoKey, email)
	if err != nil {
		handleVideoViewURLError(c, err)
		return
	}

	log.Printf("Video view URL generated successfully for user: %s", email)
	c.JSON(http.StatusOK, response)
}

// SubmitSurfReportWithIOSValidation handles surf report submission with iOS validation
func SubmitSurfReportWithIOSValidation(c *gin.Context) {
	logRequestDetails(c, "iOS Validated Report Submission Request")
	log.Print("start of submit iOS validated report")

	var report model.ReportWithIOSValidation
	if err := c.BindJSON(&report); err != nil {
		handleReportBindingError(c, err)
		return
	}

	log.Printf(
		"iOS Validated Report data received: Country=%s, Region=%s, Spot=%s, ImageKey=%s, VideoKey=%s, IOSValidated=%t",
		report.Country, report.Region, report.Spot, report.ImageKey, report.VideoKey, report.IOSValidated)

	if !validateReportLocation(c, report.Country, report.Region, report.Spot) {
		return
	}

	if !validateIOSValidation(c, report.IOSValidated) {
		return
	}

	if !validateMediaProvided(c, report.ImageKey, report.VideoKey) {
		return
	}

	if !validateIOSReportFields(c, &report) {
		return
	}

	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return
	}

	log.Printf("User email from context: %s", email)
	user, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Failed to fetch user information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User information error",
			"message": "Unable to retrieve your user profile",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	if err := ReportService.SubmitSurfReportWithIOSValidation(&report, email, user.GivenName); err != nil {
		handleIOSReportError(c, err)
		return
	}

	log.Printf("iOS Validated Report submitted successfully for user: %s", email)
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// DeleteUploadedMedia handles deletion of uploaded media from S3
func DeleteUploadedMedia(c *gin.Context) {
	logRequestDetails(c, "Delete Uploaded Media Request")

	mediaKey := c.Query("key")
	mediaType := c.Query("type")

	if !validateMediaDeletionRequest(c, mediaKey, mediaType) {
		return
	}

	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return
	}

	log.Printf("Deleting %s media: %s for user: %s", mediaType, mediaKey, email)
	user, ok := verifyMediaAccess(c, email, mediaKey, mediaType)
	if !ok {
		return
	}

	_ = user // User verified but not used further

	if err := ReportService.DeleteMediaFromS3(mediaKey); err != nil {
		handleMediaDeletionError(c, err)
		return
	}

	log.Printf("Successfully deleted %s media: %s for user: %s", mediaType, mediaKey, email)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Media deleted successfully",
		"mediaKey":  mediaKey,
		"mediaType": mediaType,
	})
}

// This controller uses the shared service registry

// Helper functions
func getUserByEmail(email string) (*model.User, error) {
	return UserService.GetUserByEmail(email)
}

// isValidMediaKey validates that the media key is in the expected format
func isValidMediaKey(mediaKey string) bool {
	// Media keys should follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.ext
	// This prevents path traversal attacks
	if mediaKey == "" {
		return false
	}
	
	// Check if it starts with the expected prefix
	if !strings.HasPrefix(mediaKey, "surf-reports/") {
		return false
	}
	
	// Check for path traversal attempts
	if strings.Contains(mediaKey, "..") || strings.Contains(mediaKey, "//") {
		return false
	}
	
	// Check for valid file extensions
	validExtensions := []string{".jpg", ".jpeg", ".png", ".mp4", ".mov", ".avi"}
	hasValidExtension := false
	for _, ext := range validExtensions {
		if strings.HasSuffix(strings.ToLower(mediaKey), ext) {
			hasValidExtension = true
			break
		}
	}
	
	return hasValidExtension
}

// canUserAccessMedia checks if a user has permission to access/delete a specific media file
func canUserAccessMedia(mediaKey, userUUID, mediaType string) bool {
	// Media keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.ext
	// We need to extract the UUID from the media key to verify ownership
	
	// Split the media key by "/" to get the parts
	parts := strings.Split(mediaKey, "/")
	if len(parts) < 3 {
		log.Printf("Invalid media key format: %s", mediaKey)
		return false
	}
	
	// Get the filename part (last part)
	filename := parts[len(parts)-1]
	
	// Remove the file extension
	var filenameWithoutExt string
	if mediaType == "video" {
		// Remove video extensions
		for _, ext := range []string{".mp4", ".mov", ".avi"} {
			if strings.HasSuffix(filename, ext) {
				filenameWithoutExt = strings.TrimSuffix(filename, ext)
				break
			}
		}
	} else {
		// Remove image extensions
		for _, ext := range []string{".jpg", ".jpeg", ".png"} {
			if strings.HasSuffix(filename, ext) {
				filenameWithoutExt = strings.TrimSuffix(filename, ext)
				break
			}
		}
	}
	
	if filenameWithoutExt == "" {
		log.Printf("Media key does not have a valid extension: %s", mediaKey)
		return false
	}
	
	// Split by "_" to get timestamp and UUID
	fileParts := strings.Split(filenameWithoutExt, "_")
	if len(fileParts) < 2 {
		log.Printf("Invalid media key filename format: %s", filename)
		return false
	}
	
	// The UUID should be the last part after splitting by "_"
	mediaUUID := fileParts[len(fileParts)-1]
	
	// Check if the UUID matches the user's UUID
	if mediaUUID != userUUID {
		log.Printf("Media UUID %s does not match user UUID %s", mediaUUID, userUUID)
		return false
	}
	
	return true
}

