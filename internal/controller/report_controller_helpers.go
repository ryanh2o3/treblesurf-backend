package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"treblesurf-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// logRequestDetails logs request metadata for debugging.
func logRequestDetails(c *gin.Context, prefix string) {
	log.Printf("=== %s ===", prefix)
	log.Printf("User-Agent: %s", c.Request.UserAgent())
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))
	log.Printf("X-CSRF-Token: %s", c.GetHeader("X-CSRF-Token"))
	log.Printf("Origin: %s", c.GetHeader("Origin"))
	log.Printf("Referer: %s", c.GetHeader("Referer"))
}

// validateReportLocation validates that required location fields are present.
func validateReportLocation(c *gin.Context, country, region, spot string) bool {
	if country == "" || region == "" || spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required fields",
			"message": "Country, region, and spot are required",
			"help":    "Please provide all required location information.",
		})
		return false
	}
	return true
}

// validateReportFields validates all surf report fields using the report service.
func validateReportFields(c *gin.Context, report *model.ReportWithImage) bool {
	if err := validateSurfSize(c, report.SurfSize); err != nil {
		return false
	}
	if err := validateWindAmount(c, report.WindAmount); err != nil {
		return false
	}
	if err := validateWindDirection(c, report.WindDirection); err != nil {
		return false
	}
	if err := validateConsistency(c, report.Consistency); err != nil {
		return false
	}
	if err := validateQuality(c, report.Quality); err != nil {
		return false
	}
	if err := validateMessiness(c, report.Messiness); err != nil {
		return false
	}
	return true
}

// validateS3ReportFields validates all surf report fields for S3 image reports.
func validateS3ReportFields(c *gin.Context, report *model.ReportWithS3Image) bool {
	if err := validateSurfSize(c, report.SurfSize); err != nil {
		return false
	}
	if err := validateWindAmount(c, report.WindAmount); err != nil {
		return false
	}
	if err := validateWindDirection(c, report.WindDirection); err != nil {
		return false
	}
	if err := validateConsistency(c, report.Consistency); err != nil {
		return false
	}
	if err := validateQuality(c, report.Quality); err != nil {
		return false
	}
	if err := validateMessiness(c, report.Messiness); err != nil {
		return false
	}
	return true
}

// validateSurfSize validates the surf size field.
func validateSurfSize(c *gin.Context, surfSize string) error {
	if surfSize != "" && !ReportService.IsValidSurfSize(surfSize) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid surf size",
			"message": fmt.Sprintf("Surf size '%s' is not valid", surfSize),
			"help":    "Valid surf sizes are: flat, knee-waist, chest-shoulder, head-high, overhead, double-overhead",
		})
		return fmt.Errorf("invalid surf size: %s", surfSize)
	}
	return nil
}

// validateWindAmount validates the wind amount field.
func validateWindAmount(c *gin.Context, windAmount string) error {
	if windAmount != "" && !ReportService.IsValidWindAmount(windAmount) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wind amount",
			"message": fmt.Sprintf("Wind amount '%s' is not valid", windAmount),
			"help":    "Valid wind amounts are: light, moderate, strong, very-strong",
		})
		return fmt.Errorf("invalid wind amount: %s", windAmount)
	}
	return nil
}

// validateWindDirection validates the wind direction field.
func validateWindDirection(c *gin.Context, windDirection string) error {
	if windDirection != "" && !ReportService.IsValidWindDirection(windDirection) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid wind direction",
			"message": fmt.Sprintf("Wind direction '%s' is not valid", windDirection),
			"help":    "Valid wind directions are: onshore, offshore, cross-shore, no-wind",
		})
		return fmt.Errorf("invalid wind direction: %s", windDirection)
	}
	return nil
}

// validateConsistency validates the consistency field.
func validateConsistency(c *gin.Context, consistency string) error {
	if consistency != "" && !ReportService.IsValidSurfDifficulty(consistency) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid consistency",
			"message": fmt.Sprintf("Consistency '%s' is not valid", consistency),
			"help":    "Valid consistency values are: setty, consistent, inconsistent, sporadic",
		})
		return fmt.Errorf("invalid consistency: %s", consistency)
	}
	return nil
}

// validateQuality validates the quality field.
func validateQuality(c *gin.Context, quality string) error {
	if quality != "" && !ReportService.IsValidSurfConditions(quality) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid quality",
			"message": fmt.Sprintf("Quality '%s' is not valid", quality),
			"help":    "Valid quality values are: mushy, average, okay, good, excellent",
		})
		return fmt.Errorf("invalid quality: %s", quality)
	}
	return nil
}

// validateMessiness validates the messiness field.
func validateMessiness(c *gin.Context, messiness string) error {
	if messiness != "" && !ReportService.IsValidMessiness(messiness) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid messiness",
			"message": fmt.Sprintf("Messiness '%s' is not valid", messiness),
			"help":    "Valid messiness values are: clean, slight-chop, choppy, messy",
		})
		return fmt.Errorf("invalid messiness: %s", messiness)
	}
	return nil
}

// getAuthenticatedUserEmail retrieves and validates the authenticated user's email from context.
func getAuthenticatedUserEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication required",
			"message": "You must be logged in to submit a surf report",
			"help":    "Please log in and try again.",
		})
		return "", false
	}
	return email.(string), true
}

// handleReportBindingError handles JSON binding errors for report requests.
func handleReportBindingError(c *gin.Context, err error) {
	log.Printf("Failed to bind JSON: %v", err)
	log.Printf("Request body: %+v", c.Request.Body)
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "Invalid request format",
		"message": "The request data is not in the correct format",
		"help":    "Please ensure you're sending valid JSON data with all required fields.",
	})
}

// validateIOSReportFields validates all surf report fields for iOS validated reports.
func validateIOSReportFields(c *gin.Context, report *model.ReportWithIOSValidation) bool {
	if err := validateSurfSize(c, report.SurfSize); err != nil {
		return false
	}
	if err := validateWindAmount(c, report.WindAmount); err != nil {
		return false
	}
	if err := validateWindDirection(c, report.WindDirection); err != nil {
		return false
	}
	if err := validateConsistency(c, report.Consistency); err != nil {
		return false
	}
	if err := validateQuality(c, report.Quality); err != nil {
		return false
	}
	if err := validateMessiness(c, report.Messiness); err != nil {
		return false
	}
	return true
}

// validateIOSValidation checks that iOS validation flag is set.
func validateIOSValidation(c *gin.Context, iosValidated bool) bool {
	if !iosValidated {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "iOS validation required",
			"message": "This endpoint requires iOS validation to be set to true",
			"help":    "Please use the iOS app to validate your surf report before submission.",
		})
		return false
	}
	return true
}

// validateMediaProvided checks that at least one media type is provided.
func validateMediaProvided(c *gin.Context, imageKey, videoKey string) bool {
	if imageKey == "" && videoKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No media provided",
			"message": "At least one image or video must be provided",
			"help":    "Please provide either an imageKey or videoKey (or both).",
		})
		return false
	}
	return true
}

// handleIOSReportError handles errors from iOS report submission.
func handleIOSReportError(c *gin.Context, err error) {
	log.Printf("Failed to submit iOS validated report: %v", err)

	// Handle specific error types
	switch {
	case errors.Is(err, model.ErrVideoUploadFailed):
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Video upload failed",
			"message": "Failed to upload the video to storage",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
	case errors.Is(err, model.ErrVideoRetrievalFailed):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Video not found",
			"message": "The uploaded video could not be found or accessed",
			"help":    "Please try uploading your video again. If the problem persists, contact support.",
		})
	case errors.Is(err, model.ErrInvalidVideoFormat):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid video format",
			"message": "The video format is not supported",
			"help":    "Please upload a video in MP4, MOV, or AVI format.",
		})
	case errors.Is(err, model.ErrVideoTooLarge):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Video too large",
			"message": "The video file is too large",
			"help":    "Please upload a video smaller than 100MB.",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
	}
}

// validateMediaType validates that media type is either "image" or "video".
func validateMediaType(c *gin.Context, mediaType string) bool {
	if mediaType != "image" && mediaType != "video" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid media type",
			"message": "Media type must be either 'image' or 'video'",
			"help":    "Please specify 'image' for images or 'video' for videos.",
		})
		return false
	}
	return true
}

// parseBuoyQueryParams parses and validates buoy query parameters.
func parseBuoyQueryParams(c *gin.Context) (waveHeight, waveDirection, period float64, buoyName string, ok bool) {
	waveHeightStr := c.Query("waveHeight")
	waveDirectionStr := c.Query("waveDirection")
	periodStr := c.Query("period")
	buoyName = c.Query("buoyName")

	if waveHeightStr == "" || waveDirectionStr == "" || periodStr == "" || buoyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "waveHeight, waveDirection, period, and buoyName parameters are required",
			"help":    "Please provide buoy data parameters (waveHeight, waveDirection, period, buoyName) in your request.",
		})
		return 0, 0, 0, "", false
	}

	var err error
	waveHeight, err = strconv.ParseFloat(waveHeightStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid waveHeight",
			"message": "waveHeight must be a valid number",
		})
		return 0, 0, 0, "", false
	}

	waveDirection, err = strconv.ParseFloat(waveDirectionStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid waveDirection",
			"message": "waveDirection must be a valid number",
		})
		return 0, 0, 0, "", false
	}

	period, err = strconv.ParseFloat(periodStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid period",
			"message": "period must be a valid number",
		})
		return 0, 0, 0, "", false
	}

	return waveHeight, waveDirection, period, buoyName, true
}

// parseOptionalIntParam parses an optional integer query parameter with a default value.
func parseOptionalIntParam(c *gin.Context, paramName string, defaultValue int) int {
	valueStr := c.DefaultQuery(paramName, fmt.Sprintf("%d", defaultValue))
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// handleVideoViewURLError handles errors from video view URL generation.
func handleVideoViewURLError(c *gin.Context, err error) {
	log.Printf("Failed to generate video view URL: %v", err)
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "video not found or not accessible"):
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Video not found",
			"message": "The requested video could not be found or accessed",
			"help":    "The video may have been deleted or the video key may be incorrect.",
		})
	case strings.Contains(errStr, "user not found"):
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "User not found",
			"message": "Unable to verify your user account",
			"help":    "Please log in again and try again.",
		})
	case strings.Contains(errStr, "access denied"):
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "You don't have permission to view this video",
			"help":    "You can only view videos from your own surf reports.",
		})
	case strings.Contains(errStr, "failed to generate presigned view URL"):
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate view URL",
			"message": "Unable to create a secure view link for the video",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video view URL"})
	}
}

// validateMediaDeletionRequest validates media deletion request parameters.
func validateMediaDeletionRequest(c *gin.Context, mediaKey, mediaType string) bool {
	if mediaKey == "" || mediaType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "Both 'key' and 'type' parameters are required",
			"help":    "Please provide the media key and type (image or video) in your request.",
		})
		return false
	}

	if !validateMediaType(c, mediaType) {
		return false
	}

	if !isValidMediaKey(mediaKey) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid media key format",
			"message": "The media key format is not valid",
			"help":    "Please provide a valid media key.",
		})
		return false
	}

	return true
}

// verifyMediaAccess verifies that the user has permission to access the media.
func verifyMediaAccess(c *gin.Context, email, mediaKey, mediaType string) (*model.User, bool) {
	user, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Failed to fetch user information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "User information error",
			"message": "Unable to retrieve your user profile",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return nil, false
	}

	if !canUserAccessMedia(mediaKey, user.UUID, mediaType) {
		log.Printf("User %s attempted to delete media they don't own: %s", email, mediaKey)
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "Access denied",
			"message": "You don't have permission to delete this media",
			"help":    "You can only delete media that you uploaded.",
		})
		return nil, false
	}

	return user, true
}

// handleMediaDeletionError handles errors from media deletion.
func handleMediaDeletionError(c *gin.Context, err error) {
	log.Printf("Failed to delete media: %v", err)
	errStr := err.Error()

	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "NoSuchKey") {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Media not found",
			"message": "The requested media could not be found",
			"help":    "The media may have already been deleted or the key may be incorrect.",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error":   "Failed to delete media",
		"message": "Unable to delete the media from storage",
		"help":    "Please try again in a moment. If the problem persists, contact support.",
	})
}

// handleReportSubmissionCommon handles common logic for report submission handlers.
func handleReportSubmissionCommon(
	c *gin.Context,
	country, region, spot string,
	validateFunc func(*gin.Context) bool,
) (string, *model.User, bool) {
	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return "", nil, false
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
		return "", nil, false
	}

	if !validateReportLocation(c, country, region, spot) {
		return "", nil, false
	}

	if !validateFunc(c) {
		return "", nil, false
	}

	return email, user, true
}

// handleReportSubmissionSuccess handles the success response for report submissions.
func handleReportSubmissionSuccess(c *gin.Context, email, reportType string) {
	log.Printf("%s submitted successfully for user: %s", reportType, email)
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// validateUploadURLParams validates upload URL generation parameters.
func validateUploadURLParams(c *gin.Context) (country, region, spot, email string, ok bool) {
	country = c.Query("country")
	region = c.Query("region")
	spot = c.Query("spot")

	if country == "" || region == "" || spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help":    "Please provide all required location parameters in your request.",
		})
		return "", "", "", "", false
	}

	emailVal, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Authentication required",
			"message": "You must be logged in to generate an upload URL",
			"help":    "Please log in and try again.",
		})
		return "", "", "", "", false
	}

	email = emailVal.(string)
	return country, region, spot, email, true
}

// handleUploadURLError handles errors from upload URL generation.
func handleUploadURLError(c *gin.Context, mediaType string, err error) {
	log.Printf("Failed to generate %s upload URL: %v", mediaType, err)

	if strings.Contains(err.Error(), "failed to generate presigned URL") {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate upload URL",
			"message": fmt.Sprintf("Unable to create a secure upload link for your %s", mediaType),
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": fmt.Sprintf("Failed to generate %s upload URL", mediaType),
	})
}

