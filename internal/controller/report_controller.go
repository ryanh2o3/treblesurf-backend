package controller

import (
	"encoding/base64"
	"log"
	"net/http"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Report data received: Country=%s, Region=%s, Spot=%s, SurfSize=%s",
		report.Country, report.Region, report.Spot, report.SurfSize)

	email, exists := c.Get("email")
	if !exists {
		log.Printf("No email found in context - authentication issue")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	log.Printf("User email from context: %s", email.(string))

	user, err2 := getUserByEmail(email.(string))
	if err2 != nil {
		log.Printf("Failed to fetch user information: %v", err2)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	// Use the report service to submit the report
	err := ReportService.SubmitSurfReport(&report, email.(string), user.GivenName)
	if err != nil {
		log.Printf("Failed to submit report: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	log.Printf("Report submitted successfully for user: %s", email.(string))
	c.JSON(http.StatusOK, gin.H{"message": "Report submitted successfully"})
}

// RetrieveTodaysSurfReports retrieves surf reports for a specific spot
func RetrieveTodaysSurfReports(c *gin.Context) {
	countryName := c.Query("country")
	regionName := c.Query("region")
	spotName := c.Query("spot")

	reports, err := ReportService.GetTodaysSurfReports(countryName, regionName, spotName)
	if err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image key is required"})
		return
	}

	// Get the image from the report service
	imageData, contentType, err := ReportService.GetReportImage(imageKey)
	if err != nil {
		log.Printf("Error getting image: %v", err)
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

// This controller uses the shared service registry

// Helper functions
func getUserByEmail(email string) (*model.User, error) {
	return UserService.GetUserByEmail(email)
}

