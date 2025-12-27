package controller

import (
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// SwellPredictionController provides HTTP handlers for swell prediction endpoints.
type SwellPredictionController struct {
	swellPredictionService *service.SwellPredictionService
}

// NewSwellPredictionController creates a new swell prediction controller instance.
func NewSwellPredictionController(swellPredictionService *service.SwellPredictionService) *SwellPredictionController {
	return &SwellPredictionController{
		swellPredictionService: swellPredictionService,
	}
}

// GetSpotSwellPrediction retrieves the latest swell prediction for a specific spot
func (c *SwellPredictionController) GetSpotSwellPrediction(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	if spotName == "" || regionName == "" || countryName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "spot, region, and country parameters are required"})
		return
	}

	predictions, err := c.swellPredictionService.GetSpotSwellPrediction(spotName, regionName, countryName)
	if err != nil {
		log.Printf("Error getting spot swell prediction: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if len(predictions) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No swell prediction found for this spot"})
		return
	}
	
	// Sort predictions by forecast_timestamp (earliest first for better UX)
	sort.Slice(predictions, func(i, j int) bool {
		timeI, _ := strconv.ParseInt(predictions[i]["forecast_timestamp"].(string), 10, 64)
		timeJ, _ := strconv.ParseInt(predictions[j]["forecast_timestamp"].(string), 10, 64)
		return timeI < timeJ
	})
	
	ctx.JSON(http.StatusOK, predictions) // Return all predictions
}

// GetListSpotsSwellPrediction retrieves swell predictions for multiple spots
func (c *SwellPredictionController) GetListSpotsSwellPrediction(ctx *gin.Context) {
	spotsStr := ctx.Query("spots")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	
	if spotsStr == "" || regionName == "" || countryName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "spots, region, and country parameters are required"})
		return
	}
	
	spots := strings.Split(spotsStr, ",")
	log.Printf("Getting swell predictions for spots: %v", spots)

	predictions, err := c.swellPredictionService.GetListSpotsSwellPrediction(spots, regionName, countryName)
	if err != nil {
		log.Printf("Error getting list spots swell prediction: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(predictions) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No swell predictions found"})
		return
	}

	ctx.JSON(http.StatusOK, predictions)
}

// GetRegionSwellPrediction retrieves swell predictions for all spots in a region
func (c *SwellPredictionController) GetRegionSwellPrediction(ctx *gin.Context) {
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	if regionName == "" || countryName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "region and country parameters are required"})
		return
	}

	predictions, err := c.swellPredictionService.GetRegionSwellPrediction(regionName, countryName)
	if err != nil {
		log.Printf("Error getting region swell prediction: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(predictions) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No swell predictions found for this region"})
		return
	}

	// Sort the predictions by spot_id
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i]["spot_id"].(string) < predictions[j]["spot_id"].(string)
	})

	ctx.JSON(http.StatusOK, predictions)
}

// GetSpotSwellPredictionRange retrieves swell predictions for a spot within a time range
func (c *SwellPredictionController) GetSpotSwellPredictionRange(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	startTimeStr := ctx.Query("startTime") // Expected format: 2006-01-02T15:00:00Z
	endTimeStr := ctx.Query("endTime")     // Expected format: 2006-01-02T15:00:00Z

	if spotName == "" || regionName == "" || countryName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "spot, region, and country parameters are required"})
		return
	}

	if startTimeStr == "" || endTimeStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "startTime and endTime parameters are required"})
		return
	}

	startTime, err := time.Parse("2006-01-02T15:00:00Z", startTimeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format. Use: 2006-01-02T15:00:00Z"})
		return
	}

	endTime, err := time.Parse("2006-01-02T15:00:00Z", endTimeStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format. Use: 2006-01-02T15:00:00Z"})
		return
	}

	predictions, err := c.swellPredictionService.GetSpotSwellPredictionRange(
		spotName, regionName, countryName, startTime, endTime,
	)
	if err != nil {
		log.Printf("Error getting spot swell prediction range: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(predictions) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No swell predictions found for this time range"})
		return
	}

	ctx.JSON(http.StatusOK, predictions)
}

// GetRecentSwellPredictions retrieves all recent swell predictions (last N hours)
func (c *SwellPredictionController) GetRecentSwellPredictions(ctx *gin.Context) {
	hoursStr := ctx.Query("hours")
	hours := 24 // Default to 24 hours

	if hoursStr != "" {
		if parsedHours, err := strconv.Atoi(hoursStr); err == nil && parsedHours > 0 {
			hours = parsedHours
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hours parameter. Must be a positive integer"})
			return
		}
	}

	predictions, err := c.swellPredictionService.GetRecentSwellPredictions(hours)
	if err != nil {
		log.Printf("Error getting recent swell predictions: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(predictions) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No recent swell predictions found"})
		return
	}

	// Sort the predictions by spot_id
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i]["spot_id"].(string) < predictions[j]["spot_id"].(string)
	})

	ctx.JSON(http.StatusOK, predictions)
}

// GetSwellPredictionStatus returns metadata about swell predictions (count, last update, etc.)
func (c *SwellPredictionController) GetSwellPredictionStatus(ctx *gin.Context) {
	// Get recent predictions to determine status
	predictions, err := c.swellPredictionService.GetRecentSwellPredictions(24)
	if err != nil {
		log.Printf("Error getting swell prediction status: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := gin.H{
		"total_predictions": len(predictions),
		"last_updated":      time.Now().Format(time.RFC3339),
		"status":            "active",
	}

	if len(predictions) == 0 {
		status["status"] = "no_data"
		status["message"] = "No recent swell predictions available"
	} else {
		status["message"] = "Swell predictions are up to date"
	}

	ctx.JSON(http.StatusOK, status)
}

// GetClosestAIPredictionForSpot retrieves the closest AI prediction for a specific spot
// 
// This endpoint finds the AI prediction with an arrival_time closest to the current time.
// It efficiently uses forecast_timestamp (Unix timestamp) for comparison instead of parsing arrival_time.
// 
// Timestamp Fields:
// - forecast_timestamp: Unix timestamp (string) representing when the swell is predicted to arrive at the surf spot
// - generated_at: Unix timestamp (string) representing when the AI prediction was generated/created
// - arrival_time: ISO format datetime string (e.g., "2024-01-15T14:30:00") representing the predicted arrival time
//
// The endpoint searches within a 60-hour window (12 hours back, 48 hours forward) and falls back to
// a 7-day search if no predictions are found in the primary window.
func (c *SwellPredictionController) GetClosestAIPredictionForSpot(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	if spotName == "" || regionName == "" || countryName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "spot, region, and country parameters are required"})
		return
	}

	prediction, err := c.swellPredictionService.GetClosestAIPredictionForSpot(spotName, regionName, countryName)
	if err != nil {
		log.Printf("Error getting closest AI prediction: %v", err)
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	
	ctx.JSON(http.StatusOK, prediction)
}
