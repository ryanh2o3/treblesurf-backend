package controller

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ReportController handles surf report routes.
type ReportController struct {
	reports *service.ReportService
	users   *service.UserService
}

type reportSubmissionConfig struct {
	validate     func(*gin.Context) bool
	submit       func(string, string) error
	country      string
	region       string
	spot         string
	successLabel string
	errorPrefix  string
	logFields    []slog.Attr
}

func buildReportConfigFromJSON[T any](
	c *gin.Context,
	report *T,
	build func(*T) *reportSubmissionConfig,
) (*reportSubmissionConfig, bool) {
	if err := c.BindJSON(report); err != nil {
		handleReportBindingError(c, err)
		return nil, false
	}
	return build(report), true
}

func NewReportController(reports *service.ReportService, users *service.UserService) *ReportController {
	return &ReportController{reports: reports, users: users}
}

func handleImageError(c *gin.Context, err error, logPrefix string) {
	requestLogger(c).Warn(logPrefix, slog.Any("error", err))

	// Handle ImageValidationError type
	var imageValidationErr *model.ImageValidationError
	if errors.As(err, &imageValidationErr) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Image validation failed",
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
			"error":   "Image not surf-related",
			"message": "The image does not appear to show surf conditions",
			"help":    "Please upload a photo that clearly shows the ocean, waves, beach, or coastline.",
		})
	case errors.Is(err, model.ErrInvalidImageData):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid image data",
			"message": "The image data provided is not in a valid format",
			"help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) " +
				"and that the image data is properly encoded.",
		})
	case errors.Is(err, model.ErrImageUploadFailed):
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Image upload failed",
			"message": "Failed to upload the image to storage",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
	case errors.Is(err, model.ErrImageRetrievalFailed):
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Image not found",
			"message": "The uploaded image could not be found or accessed",
			"help":    "Please try uploading your image again. If the problem persists, contact support.",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
	}
}

func (rc *ReportController) submitReport(
	c *gin.Context,
	country, region, spot string,
	validate func(*gin.Context) bool,
	submit func(email, userName string) error,
	successLabel string,
	errorPrefix string,
) {
	email, user, ok := handleReportSubmissionCommon(
		c, country, region, spot,
		validate,
		rc.users,
	)
	if !ok {
		return
	}

	if err := submit(email, user.GivenName); err != nil {
		handleImageError(c, err, errorPrefix)
		return
	}

	handleReportSubmissionSuccess(c, email, successLabel)
}

func (rc *ReportController) handleReportSubmission(
	c *gin.Context,
	logPrefix string,
	startMessage string,
	builder func(*gin.Context) (*reportSubmissionConfig, bool),
) {
	logRequestDetails(c, logPrefix)
	requestLogger(c).Info(startMessage)

	cfg, ok := builder(c)
	if !ok || cfg == nil {
		return
	}

	if len(cfg.logFields) > 0 {
		requestLogger(c).Info("report data received", attrsToArgs(cfg.logFields)...)
	}

	rc.submitReport(
		c,
		cfg.country,
		cfg.region,
		cfg.spot,
		cfg.validate,
		cfg.submit,
		cfg.successLabel,
		cfg.errorPrefix,
	)
}

func buildReportConfig(
	country string,
	region string,
	spot string,
	logFields []slog.Attr,
	validate func(*gin.Context) bool,
	submit func(string, string) error,
	successLabel string,
	errorPrefix string,
) *reportSubmissionConfig {
	return &reportSubmissionConfig{
		country:      country,
		region:       region,
		spot:         spot,
		logFields:    logFields,
		validate:     validate,
		submit:       submit,
		successLabel: successLabel,
		errorPrefix:  errorPrefix,
	}
}

func buildReportConfigWithExtraField(
	country string,
	region string,
	spot string,
	extraKey string,
	extraValue string,
	validate func(*gin.Context) bool,
	submit func(string, string) error,
	successLabel string,
	errorPrefix string,
) *reportSubmissionConfig {
	logFields := []slog.Attr{
		slog.String("country", country),
		slog.String("region", region),
		slog.String("spot", spot),
		slog.String(extraKey, extraValue),
	}
	return buildReportConfig(
		country,
		region,
		spot,
		logFields,
		validate,
		submit,
		successLabel,
		errorPrefix,
	)
}

func (rc *ReportController) buildImageReportConfig(c *gin.Context) (*reportSubmissionConfig, bool) {
	var report model.ReportWithImage
	return buildReportConfigFromJSON(c, &report, func(r *model.ReportWithImage) *reportSubmissionConfig {
		return buildReportConfigWithExtraField(
			r.Country,
			r.Region,
			r.Spot,
			"surf_size",
			r.SurfSize,
			func(ctx *gin.Context) bool { return validateReportFields(rc.reports, ctx, r) },
			func(email, userName string) error {
				return rc.reports.SubmitSurfReport(c.Request.Context(), r, email, userName)
			},
			"Report",
			"Failed to submit report",
		)
	})
}

func (rc *ReportController) buildS3ReportConfig(c *gin.Context) (*reportSubmissionConfig, bool) {
	var report model.ReportWithS3Image
	return buildReportConfigFromJSON(c, &report, func(r *model.ReportWithS3Image) *reportSubmissionConfig {
		return buildReportConfigWithExtraField(
			r.Country,
			r.Region,
			r.Spot,
			"image_key",
			r.ImageKey,
			func(ctx *gin.Context) bool { return validateS3ReportFields(rc.reports, ctx, r) },
			func(email, userName string) error {
				return rc.reports.SubmitSurfReportWithS3Image(c.Request.Context(), r, email, userName)
			},
			"S3 Image Report",
			"Failed to submit S3 image report",
		)
	})
}

func attrsToArgs(attrs []slog.Attr) []any {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return args
}

func (rc *ReportController) serveReportMedia(
	c *gin.Context,
	key string,
	mediaType string,
	fetch func(context.Context, string) ([]byte, string, error),
) {
	title := strings.ToUpper(mediaType[:1]) + mediaType[1:]
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   fmt.Sprintf("Missing %s key", mediaType),
			"message": fmt.Sprintf("%s key parameter is required", title),
			"help":    fmt.Sprintf("Please provide the %s key in your request.", mediaType),
		})
		return
	}

	data, contentType, err := fetch(c.Request.Context(), key)
	if err != nil {
		requestLogger(c).Warn("error getting media", slog.String("type", mediaType), slog.Any("error", err))

		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   fmt.Sprintf("%s not found", title),
				"message": fmt.Sprintf("The requested %s could not be found or accessed", mediaType),
				"help":    fmt.Sprintf("The %s may have been deleted or the %s key may be incorrect.", mediaType, mediaType),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve %s", mediaType),
		})
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(data)
	c.JSON(http.StatusOK, gin.H{
		fmt.Sprintf("%sData", mediaType): base64Data,
		"contentType":                    contentType,
	})
}

func (rc *ReportController) validateIOSReportSubmission(c *gin.Context, report *model.ReportWithIOSValidation) bool {
	if report == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report payload"})
		return false
	}
	if !validateReportLocation(c, report.Country, report.Region, report.Spot) {
		return false
	}
	if !validateIOSValidation(c, report.IOSValidated) {
		return false
	}
	if !validateMediaProvided(c, report.ImageKey, report.VideoKey) {
		return false
	}
	if !validateIOSReportFields(rc.reports, c, report) {
		return false
	}
	return true
}

func (rc *ReportController) fetchAuthenticatedUser(c *gin.Context) (*model.User, string, bool) {
	email, ok := getAuthenticatedUserEmail(c)
	if !ok {
		return nil, "", false
	}

	requestLogger(c).Info("user email from context", slog.String("email", email))
	user, ok := loadUserOr404(c, rc.users, email)
	if !ok {
		return nil, "", false
	}

	return user, email, true
}

func (rc *ReportController) SubmitCurrentSurfReport(c *gin.Context) {
	rc.handleReportSubmission(
		c,
		"Report Submission Request",
		"start of submit report",
		rc.buildImageReportConfig,
	)
}

func (rc *ReportController) SubmitSurfReportWithS3Image(c *gin.Context) {
	rc.handleReportSubmission(
		c,
		"S3 Image Report Submission Request",
		"start of submit S3 image report",
		rc.buildS3ReportConfig,
	)
}

func (rc *ReportController) GenerateImageUploadURL(c *gin.Context) {
	requestLogger(c).Info("generate image upload URL request")

	country, region, spot, email, ok := validateUploadURLParams(c)
	if !ok {
		return
	}

	requestLogger(c).Info("generating upload URL",
		slog.String("country", country),
		slog.String("region", region),
		slog.String("spot", spot),
		slog.String("user", email),
	)

	response, err := rc.reports.GenerateImageUploadURL(c.Request.Context(), country, region, spot, email)
	if err != nil {
		handleUploadURLError(c, "image", err)
		return
	}

	requestLogger(c).Info("upload URL generated successfully", slog.String("user", email))
	c.JSON(http.StatusOK, response)
}

func (rc *ReportController) RetrieveTodaysSurfReports(c *gin.Context) {
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help":    "Please provide all required location parameters in your request.",
		})
		return
	}

	reports, err := rc.reports.GetTodaysSurfReports(c.Request.Context(), countryName, regionName, spotName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No reports found"})
			return
		}
		requestLogger(c).Warn("failed to retrieve surf reports", slog.Any("error", err))

		if errors.Is(err, service.ErrSurfReportsQuery) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve reports",
				"message": "Unable to fetch surf reports from the database",
				"help":    "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve reports"})
		return
	}

	c.JSON(http.StatusOK, mapSpotReportsToClient(reports))
}

func (rc *ReportController) GetAllSpotSurfReports(c *gin.Context) {
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	limitStr := c.DefaultQuery("limit", "50") // Default to 50 reports per page

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "Country, region, and spot parameters are required",
			"help":    "Please provide all required location parameters in your request.",
		})
		return
	}

	// Parse limit parameter
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid limit parameter",
			"message": "Limit must be a positive integer or 0 for all reports",
		})
		return
	}

	// For simplicity, we'll skip pagination token handling for now
	// In a production system, you'd want to handle pagination tokens
	reports, err := rc.reports.GetSpotSurfReports(c.Request.Context(), countryName, regionName, spotName, limit)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No reports found"})
			return
		}
		requestLogger(c).Warn("failed to retrieve surf reports", slog.Any("error", err))

		if errors.Is(err, service.ErrSurfReportsQuery) {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to retrieve reports",
				"message": "Unable to fetch surf reports from the database",
				"help":    "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve reports"})
		return
	}

	c.JSON(http.StatusOK, mapSpotReportsToClient(reports))
}

func (rc *ReportController) GetSurfReportsWithSimilarBuoyData(c *gin.Context) {
	waveHeight, waveDirection, period, buoyName, ok := parseBuoyQueryParams(c)
	if !ok {
		return
	}

	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	daysBack := parseOptionalIntParam(c, "daysBack", 365)
	maxResults := parseOptionalIntParam(c, "maxResults", 20)

	reports, err := rc.reports.GetSurfReportsWithSimilarBuoyData(
		c.Request.Context(),
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
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No reports found"})
			return
		}
		requestLogger(c).Warn("failed to retrieve surf reports with similar buoy data", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve reports",
			"message": "Unable to fetch surf reports with similar buoy conditions",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	c.JSON(http.StatusOK, mapSpotReportsToClient(reports))
}

// GetSurfReportsWithMatchingConditions retrieves surf reports for a spot where:
// 1. Buoy data at the report time (accounting for travel time) matches current buoy data from nearest buoys
// 2. Wind conditions from forecast data at the report time are similar to current wind conditions
func (rc *ReportController) GetSurfReportsWithMatchingConditions(c *gin.Context) {
	// Parse query parameters
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")
	daysBackStr := c.DefaultQuery("daysBack", "365")
	maxResultsStr := c.DefaultQuery("maxResults", "20")

	// Validate required parameters
	if countryName == "" || regionName == "" || spotName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing required parameters",
			"message": "country, region, and spot parameters are required",
			"help":    "Please provide all required location parameters (country, region, spot) in your request.",
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
	reports, err := rc.reports.GetSurfReportsWithMatchingConditions(
		c.Request.Context(),
		countryName,
		regionName,
		spotName,
		daysBack,
		maxResults,
	)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No reports found"})
			return
		}
		requestLogger(c).Warn("failed to retrieve surf reports with matching conditions", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve reports",
			"message": "Unable to fetch surf reports with matching conditions",
			"help":    "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	c.JSON(http.StatusOK, mapSpotReportsToClient(reports))
}

func (rc *ReportController) GetReportImage(c *gin.Context) {
	rc.serveReportMedia(c, c.Query("key"), "image", rc.reports.GetReportImage)
}

func (rc *ReportController) GenerateVideoUploadURL(c *gin.Context) {
	requestLogger(c).Info("generate video upload URL request")

	country, region, spot, email, ok := validateUploadURLParams(c)
	if !ok {
		return
	}

	requestLogger(c).Info("generating video upload URL",
		slog.String("country", country),
		slog.String("region", region),
		slog.String("spot", spot),
		slog.String("user", email),
	)

	response, err := rc.reports.GenerateVideoUploadURL(c.Request.Context(), country, region, spot, email)
	if err != nil {
		handleUploadURLError(c, constants.MediaTypeVideo, err)
		return
	}

	requestLogger(c).Info("video upload URL generated successfully", slog.String("user", email))
	c.JSON(http.StatusOK, response)
}

func (rc *ReportController) GetReportVideo(c *gin.Context) {
	rc.serveReportMedia(c, c.Query("key"), "video", rc.reports.GetReportVideo)
}

func (rc *ReportController) GenerateVideoViewURL(c *gin.Context) {
	requestLogger(c).Info("generate video view URL request")

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

	requestLogger(c).Info("generating video view URL", slog.String("video_key", videoKey), slog.String("user", email))
	response, err := rc.reports.GenerateVideoViewURL(c.Request.Context(), videoKey, email)
	if err != nil {
		handleVideoViewURLError(c, err)
		return
	}

	requestLogger(c).Info("video view URL generated successfully", slog.String("user", email))
	c.JSON(http.StatusOK, response)
}

func (rc *ReportController) SubmitSurfReportWithIOSValidation(c *gin.Context) {
	logRequestDetails(c, "iOS Validated Report Submission Request")
	requestLogger(c).Info("start of submit iOS validated report")

	var report model.ReportWithIOSValidation
	if err := c.BindJSON(&report); err != nil {
		handleReportBindingError(c, err)
		return
	}

	requestLogger(c).Info("iOS validated report data received",
		slog.String("country", report.Country),
		slog.String("region", report.Region),
		slog.String("spot", report.Spot),
		slog.String("image_key", report.ImageKey),
		slog.String("video_key", report.VideoKey),
		slog.Bool("ios_validated", report.IOSValidated),
	)

	if !rc.validateIOSReportSubmission(c, &report) {
		return
	}

	user, email, ok := rc.fetchAuthenticatedUser(c)
	if !ok {
		return
	}

	if err := rc.reports.SubmitSurfReportWithIOSValidation(c.Request.Context(), &report, email, user.GivenName); err != nil {
		handleIOSReportError(c, err)
		return
	}

	requestLogger(c).Info("iOS validated report submitted successfully", slog.String("user", email))
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

func (rc *ReportController) DeleteUploadedMedia(c *gin.Context) {
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

	requestLogger(c).Info("deleting media", slog.String("type", mediaType), slog.String("key", mediaKey), slog.String("user", email))
	user, ok := verifyMediaAccess(rc.users, c, email, mediaKey, mediaType)
	if !ok {
		return
	}

	_ = user // User verified but not used further

	if err := rc.reports.DeleteMediaFromS3(c.Request.Context(), mediaKey); err != nil {
		handleMediaDeletionError(c, err)
		return
	}

	requestLogger(c).Info("successfully deleted media",
		slog.String("type", mediaType),
		slog.String("key", mediaKey),
		slog.String("user", email),
	)
	c.JSON(http.StatusOK, gin.H{
		"message":   "Media deleted successfully",
		"mediaKey":  mediaKey,
		"mediaType": mediaType,
	})
}

func isValidMediaKey(mediaKey string) bool {
	// Media keys should follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.ext
	// Prevents path traversal attacks
	if mediaKey == "" {
		return false
	}

	if !strings.HasPrefix(mediaKey, "surf-reports/") {
		return false
	}

	if strings.Contains(mediaKey, "..") || strings.Contains(mediaKey, "//") {
		return false
	}

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

func canUserAccessMedia(logger *slog.Logger, mediaKey, userUUID, mediaType string) bool {
	// Media keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.ext
	// We need to extract the UUID from the media key to verify ownership

	// Split the media key by "/" to get the parts
	parts := strings.Split(mediaKey, "/")
	if len(parts) < 3 {
		logger.Warn("invalid media key format", slog.String("key", mediaKey))
		return false
	}

	// Get the filename part (last part)
	filename := parts[len(parts)-1]

	// Remove the file extension
	var filenameWithoutExt string
	if mediaType == constants.MediaTypeVideo {
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
		logger.Warn("media key does not have a valid extension", slog.String("key", mediaKey))
		return false
	}

	// Split by "_" to get timestamp and UUID
	fileParts := strings.Split(filenameWithoutExt, "_")
	if len(fileParts) < 2 {
		logger.Warn("invalid media key filename format", slog.String("filename", filename))
		return false
	}

	// The UUID should be the last part after splitting by "_"
	mediaUUID := fileParts[len(fileParts)-1]

	// Check if the UUID matches the user's UUID
	if mediaUUID != userUUID {
		logger.Warn("media UUID does not match user UUID", slog.String("media_uuid", mediaUUID), slog.String("user_uuid", userUUID))
		return false
	}

	return true
}
