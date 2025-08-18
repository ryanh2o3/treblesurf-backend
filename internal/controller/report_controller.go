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
	log.Print("start of submit report")
	var report model.ReportWithImage
	if err := c.BindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user, err2 := getUserByEmail(email.(string))
	if err2 != nil {
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

