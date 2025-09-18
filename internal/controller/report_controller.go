package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"treblesurf-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// This controller uses the shared service registry

// SubmitCurrentSurfReport handles surf report submission
func SubmitCurrentSurfReport(c *gin.Context) {
	log.Printf("=== Report Submission Request ===")
	log.Printf("User-Agent: %s", c.Request.UserAgent())
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))
	log.Printf("X-CSRF-Token: %s", c.GetHeader("X-CSRF-Token"))
	log.Printf("Origin: %s", c.GetHeader("Origin"))
	log.Printf("Referer: %s", c.GetHeader("Referer"))

	log.Print("start of submit report")
	var report model.ReportWithImage
	if err := c.BindJSON(&report); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		log.Printf("Request body: %+v", c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"message": "The request data is not in the correct format",
			"help": "Please ensure you're sending valid JSON data with all required fields.",
		})
		return
	}

	log.Printf("Report data received: Country=%s, Region=%s, Spot=%s, SurfSize=%s",
		report.Country, report.Region, report.Spot, report.SurfSize)

	// Validate required fields
	if report.Country == "" || report.Region == "" || report.Spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required fields",
			"message": "Country, region, and spot are required",
			"help": "Please provide all required location information.",
		})
		return
	}

	// Validate surf size if provided
	if report.SurfSize != "" && !ReportService.IsValidSurfSize(report.SurfSize) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid surf size",
			"message": fmt.Sprintf("Surf size '%s' is not valid", report.SurfSize),
			"help": "Valid surf sizes are: flat, knee-waist, chest-shoulder, head-high, overhead, double-overhead",
		})
		return
	}

	// Validate wind amount if provided
	if report.WindAmount != "" && !ReportService.IsValidWindAmount(report.WindAmount) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind amount",
			"message": fmt.Sprintf("Wind amount '%s' is not valid", report.WindAmount),
			"help": "Valid wind amounts are: light, moderate, strong, very-strong",
		})
		return
	}

	// Validate wind direction if provided
	if report.WindDirection != "" && !ReportService.IsValidWindDirection(report.WindDirection) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind direction",
			"message": fmt.Sprintf("Wind direction '%s' is not valid", report.WindDirection),
			"help": "Valid wind directions are: onshore, offshore, cross-shore, no-wind",
		})
		return
	}

	// Validate consistency if provided
	if report.Consistency != "" && !ReportService.IsValidSurfDifficulty(report.Consistency) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid consistency",
			"message": fmt.Sprintf("Consistency '%s' is not valid", report.Consistency),
			"help": "Valid consistency values are: setty, consistent, inconsistent, sporadic",
		})
		return
	}

	// Validate quality if provided
	if report.Quality != "" && !ReportService.IsValidSurfConditions(report.Quality) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid quality",
			"message": fmt.Sprintf("Quality '%s' is not valid", report.Quality),
			"help": "Valid quality values are: mushy, average, okay, good, excellent",
		})
		return
	}

	// Validate messiness if provided
	if report.Messiness != "" && !ReportService.IsValidMessiness(report.Messiness) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid messiness",
			"message": fmt.Sprintf("Messiness '%s' is not valid", report.Messiness),
			"help": "Valid messiness values are: clean, slight-chop, choppy, messy",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to submit a surf report",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("User email from context: %s", email.(string))

	user, err2 := getUserByEmail(email.(string))
	if err2 != nil {
		log.Printf("Failed to fetch user information: %v", err2)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User information error",
			"message": "Unable to retrieve your user profile",
			"help": "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	// Use the report service to submit the report
	err := ReportService.SubmitSurfReport(&report, email.(string), user.GivenName)
	if err != nil {
		log.Printf("Failed to submit report: %v", err)
		
		// Handle specific error types
		var imageValidationErr *model.ImageValidationError
		if errors.As(err, &imageValidationErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Image validation failed",
				"message": imageValidationErr.Error(),
				"help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. The image should be clear and focused on surf conditions.",
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
				"help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) and that the image data is properly encoded.",
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
		return
	}

	log.Printf("Report submitted successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// SubmitSurfReportWithS3Image handles surf report submission with pre-uploaded S3 image
func SubmitSurfReportWithS3Image(c *gin.Context) {
	log.Printf("=== S3 Image Report Submission Request ===")
	log.Printf("User-Agent: %s", c.Request.UserAgent())
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))

	log.Print("start of submit S3 image report")
	var report model.ReportWithS3Image
	if err := c.BindJSON(&report); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		log.Printf("Request body: %+v", c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"message": "The request data is not in the correct format",
			"help": "Please ensure you're sending valid JSON data with all required fields.",
		})
		return
	}

	log.Printf("S3 Image Report data received: Country=%s, Region=%s, Spot=%s, ImageKey=%s",
		report.Country, report.Region, report.Spot, report.ImageKey)

	// Validate required fields
	if report.Country == "" || report.Region == "" || report.Spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required fields",
			"message": "Country, region, and spot are required",
			"help": "Please provide all required location information.",
		})
		return
	}

	// Validate surf size if provided
	if report.SurfSize != "" && !ReportService.IsValidSurfSize(report.SurfSize) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid surf size",
			"message": fmt.Sprintf("Surf size '%s' is not valid", report.SurfSize),
			"help": "Valid surf sizes are: flat, knee-waist, chest-shoulder, head-high, overhead, double-overhead",
		})
		return
	}

	// Validate wind amount if provided
	if report.WindAmount != "" && !ReportService.IsValidWindAmount(report.WindAmount) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind amount",
			"message": fmt.Sprintf("Wind amount '%s' is not valid", report.WindAmount),
			"help": "Valid wind amounts are: light, moderate, strong, very-strong",
		})
		return
	}

	// Validate wind direction if provided
	if report.WindDirection != "" && !ReportService.IsValidWindDirection(report.WindDirection) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind direction",
			"message": fmt.Sprintf("Wind direction '%s' is not valid", report.WindDirection),
			"help": "Valid wind directions are: onshore, offshore, cross-shore, no-wind",
		})
		return
	}

	// Validate consistency if provided
	if report.Consistency != "" && !ReportService.IsValidSurfDifficulty(report.Consistency) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid consistency",
			"message": fmt.Sprintf("Consistency '%s' is not valid", report.Consistency),
			"help": "Valid consistency values are: setty, consistent, inconsistent, sporadic",
		})
		return
	}

	// Validate quality if provided
	if report.Quality != "" && !ReportService.IsValidSurfConditions(report.Quality) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid quality",
			"message": fmt.Sprintf("Quality '%s' is not valid", report.Quality),
			"help": "Valid quality values are: mushy, average, okay, good, excellent",
		})
		return
	}

	// Validate messiness if provided
	if report.Messiness != "" && !ReportService.IsValidMessiness(report.Messiness) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid messiness",
			"message": fmt.Sprintf("Messiness '%s' is not valid", report.Messiness),
			"help": "Valid messiness values are: clean, slight-chop, choppy, messy",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to submit a surf report",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("User email from context: %s", email.(string))

	user, err2 := getUserByEmail(email.(string))
	if err2 != nil {
		log.Printf("Failed to fetch user information: %v", err2)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User information error",
			"message": "Unable to retrieve your user profile",
			"help": "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	// Use the report service to submit the report with S3 image
	err := ReportService.SubmitSurfReportWithS3Image(&report, email.(string), user.GivenName)
	if err != nil {
		log.Printf("Failed to submit S3 image report: %v", err)
		
		// Handle specific error types
		var imageValidationErr *model.ImageValidationError
		if errors.As(err, &imageValidationErr) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Image validation failed",
				"message": imageValidationErr.Error(),
				"help": "Please ensure your image clearly shows the ocean, waves, beach, or coastline. The image should be clear and focused on surf conditions.",
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
				"help": "Please ensure you're uploading a valid image file (JPEG, PNG, etc.) and that the image data is properly encoded.",
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
		return
	}

	log.Printf("S3 Image Report submitted successfully for user: %s", email.(string))
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

// RetrieveTodaysSurfReports retrieves surf reports for a specific spot
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

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to generate a video view URL",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("Generating video view URL for key: %s, user: %s", videoKey, email.(string))

	// Generate the presigned URL
	response, err := ReportService.GenerateVideoViewURL(videoKey, email.(string))
	if err != nil {
		log.Printf("Failed to generate video view URL: %v", err)
		
		// Provide more helpful error messages for common failures
		if strings.Contains(err.Error(), "video not found or not accessible") {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Video not found",
				"message": "The requested video could not be found or accessed",
				"help": "The video may have been deleted or the video key may be incorrect.",
			})
			return
		}
		
		if strings.Contains(err.Error(), "user not found") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not found",
				"message": "Unable to verify your user account",
				"help": "Please log in again and try again.",
			})
			return
		}
		
		if strings.Contains(err.Error(), "access denied") {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
				"message": "You don't have permission to view this video",
				"help": "You can only view videos from your own surf reports.",
			})
			return
		}
		
		if strings.Contains(err.Error(), "failed to generate presigned view URL") {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to generate view URL",
				"message": "Unable to create a secure view link for the video",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate video view URL"})
		return
	}

	log.Printf("Video view URL generated successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, response)
}

// SubmitSurfReportWithIOSValidation handles surf report submission with iOS validation
func SubmitSurfReportWithIOSValidation(c *gin.Context) {
	log.Printf("=== iOS Validated Report Submission Request ===")
	log.Printf("User-Agent: %s", c.Request.UserAgent())
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))

	log.Print("start of submit iOS validated report")
	var report model.ReportWithIOSValidation
	if err := c.BindJSON(&report); err != nil {
		log.Printf("Failed to bind JSON: %v", err)
		log.Printf("Request body: %+v", c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"message": "The request data is not in the correct format",
			"help": "Please ensure you're sending valid JSON data with all required fields.",
		})
		return
	}

	log.Printf("iOS Validated Report data received: Country=%s, Region=%s, Spot=%s, ImageKey=%s, VideoKey=%s, IOSValidated=%t",
		report.Country, report.Region, report.Spot, report.ImageKey, report.VideoKey, report.IOSValidated)

	// Validate required fields
	if report.Country == "" || report.Region == "" || report.Spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required fields",
			"message": "Country, region, and spot are required",
			"help": "Please provide all required location information.",
		})
		return
	}

	// Validate that iOS validation flag is set
	if !report.IOSValidated {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "iOS validation required",
			"message": "This endpoint requires iOS validation to be set to true",
			"help": "Please use the iOS app to validate your surf report before submission.",
		})
		return
	}

	// Validate that at least one media type is provided
	if report.ImageKey == "" && report.VideoKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No media provided",
			"message": "At least one image or video must be provided",
			"help": "Please provide either an imageKey or videoKey (or both).",
		})
		return
	}

	// Validate surf size if provided
	if report.SurfSize != "" && !ReportService.IsValidSurfSize(report.SurfSize) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid surf size",
			"message": fmt.Sprintf("Surf size '%s' is not valid", report.SurfSize),
			"help": "Valid surf sizes are: flat, knee-waist, chest-shoulder, head-high, overhead, double-overhead",
		})
		return
	}

	// Validate wind amount if provided
	if report.WindAmount != "" && !ReportService.IsValidWindAmount(report.WindAmount) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind amount",
			"message": fmt.Sprintf("Wind amount '%s' is not valid", report.WindAmount),
			"help": "Valid wind amounts are: light, moderate, strong, very-strong",
		})
		return
	}

	// Validate wind direction if provided
	if report.WindDirection != "" && !ReportService.IsValidWindDirection(report.WindDirection) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid wind direction",
			"message": fmt.Sprintf("Wind direction '%s' is not valid", report.WindDirection),
			"help": "Valid wind directions are: onshore, offshore, cross-shore, no-wind",
		})
		return
	}

	// Validate consistency if provided
	if report.Consistency != "" && !ReportService.IsValidSurfDifficulty(report.Consistency) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid consistency",
			"message": fmt.Sprintf("Consistency '%s' is not valid", report.Consistency),
			"help": "Valid consistency values are: setty, consistent, inconsistent, sporadic",
		})
		return
	}

	// Validate quality if provided
	if report.Quality != "" && !ReportService.IsValidSurfConditions(report.Quality) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid quality",
			"message": fmt.Sprintf("Quality '%s' is not valid", report.Quality),
			"help": "Valid quality values are: mushy, average, okay, good, excellent",
		})
		return
	}

	// Validate messiness if provided
	if report.Messiness != "" && !ReportService.IsValidMessiness(report.Messiness) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid messiness",
			"message": fmt.Sprintf("Messiness '%s' is not valid", report.Messiness),
			"help": "Valid messiness values are: clean, slight-chop, choppy, messy",
		})
		return
	}

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
			"message": "You must be logged in to submit a surf report",
			"help": "Please log in and try again.",
		})
		return
	}

	log.Printf("User email from context: %s", email.(string))

	user, err2 := getUserByEmail(email.(string))
	if err2 != nil {
		log.Printf("Failed to fetch user information: %v", err2)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User information error",
			"message": "Unable to retrieve your user profile",
			"help": "Please try again in a moment. If the problem persists, contact support.",
		})
		return
	}

	// Use the report service to submit the iOS-validated report
	err := ReportService.SubmitSurfReportWithIOSValidation(&report, email.(string), user.GivenName)
	if err != nil {
		log.Printf("Failed to submit iOS validated report: %v", err)
		
		// Handle specific error types
		switch {
		case errors.Is(err, model.ErrVideoUploadFailed):
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Video upload failed",
				"message": "Failed to upload the video to storage",
				"help": "Please try again in a moment. If the problem persists, contact support.",
			})
		case errors.Is(err, model.ErrVideoRetrievalFailed):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Video not found",
				"message": "The uploaded video could not be found or accessed",
				"help": "Please try uploading your video again. If the problem persists, contact support.",
			})
		case errors.Is(err, model.ErrInvalidVideoFormat):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid video format",
				"message": "The video format is not supported",
				"help": "Please upload a video in MP4, MOV, or AVI format.",
			})
		case errors.Is(err, model.ErrVideoTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Video too large",
				"message": "The video file is too large",
				"help": "Please upload a video smaller than 100MB.",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		}
		return
	}

	log.Printf("iOS Validated Report submitted successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// This controller uses the shared service registry

// Helper functions
func getUserByEmail(email string) (*model.User, error) {
	return UserService.GetUserByEmail(email)
}

// isIOSClient detects if the request is from an iOS client
func isIOSClient(userAgent string) bool {
	userAgentLower := strings.ToLower(userAgent)
	return strings.Contains(userAgentLower, "ios") ||
		strings.Contains(userAgentLower, "iphone") ||
		strings.Contains(userAgentLower, "ipad")
}

