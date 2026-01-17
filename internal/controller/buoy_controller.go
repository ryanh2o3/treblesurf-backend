package controller

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// BuoyController handles buoy data routes.
type BuoyController struct {
	buoys repository.BuoyRepository
}

func NewBuoyController(buoys repository.BuoyRepository) *BuoyController {
	return &BuoyController{buoys: buoys}
}

// BuoyLocationInfo returns location information for all buoys.
func (bc *BuoyController) BuoyLocationInfo(c *gin.Context) {
	locations, err := bc.buoys.GetLocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]map[string]interface{}, 0, len(locations))
	for name, location := range locations {
		if location == nil {
			continue
		}
		results = append(results, buoyLocationToMap(name, location))
	}

	c.JSON(http.StatusOK, results)
}

// IndividualBuoyLocationInfo returns location information for a specific buoy.
func (bc *BuoyController) IndividualBuoyLocationInfo(c *gin.Context) {
	regionName := c.Query("region")
	buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")
	locations, err := bc.buoys.GetLocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	location, ok := locations[buoyName]
	if !ok && regionName != "" {
		location = locations[fmt.Sprintf("%s_%s", regionName, buoyName)]
		ok = location != nil
	}
	if !ok || location == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No buoy location found"})
		return
	}

	c.JSON(http.StatusOK, buoyLocationToMap(buoyName, location))
}

// GetLiveBuoyData returns the most recent buoy data for all buoys.
func (bc *BuoyController) GetLiveBuoyData(c *gin.Context) {
	buoys := []string{"M4", "Blackstones", "West Hebrides", "M2", "M3", "M5", "M6"}
	var buoyData []map[string]interface{}

	for _, buoy := range buoys {
		data := bc.getBuoyData(c.Request.Context(), buoy)
		buoyData = append(buoyData, data)
	}

	c.JSON(http.StatusOK, buoyData)
}

// GetBuoyDataRange returns buoy data for a specific buoy within a specified time range.
func (bc *BuoyController) GetBuoyDataRange(c *gin.Context) {
	buoyName := c.Query("buoyName")
	startTimeStr := c.Query("startTime") // expected format: 2006-01-02T15:00:00Z or 2006-01-02
	endTimeStr := c.Query("endTime")     // expected format: 2006-01-02T15:00:00Z or 2006-01-02

	// Try parsing with full timestamp first, then try date-only format
	startTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr)
	if err != nil {
		// Try parsing as date only
		startTime, err = time.Parse("2006-01-02", startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start time format. Expected 2006-01-02T15:04:05Z or 2006-01-02",
			})
			return
		}
	}

	endTime, err := time.Parse("2006-01-02T15:04:05Z", endTimeStr)
	if err != nil {
		// Try parsing as date only
		endTime, err = time.Parse("2006-01-02", endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format. Expected 2006-01-02T15:04:05Z or 2006-01-02"})
			return
		}
		// If parsing date only, set to end of day
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	}

	data, err := bc.getBuoyDataRange(c.Request.Context(), buoyName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return empty array instead of null when no data found
	if data == nil {
		data = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, data)
}

// GetSingleBuoyData returns the most recent data for a specific buoy.
func (bc *BuoyController) GetSingleBuoyData(c *gin.Context) {
	var data map[string]interface{}

	for i := 0; i < 12; i++ {
		data = bc.getBuoyData(c.Request.Context(), c.Query("buoyName"))
		if data != nil {
			break
		}
	}

	c.JSON(http.StatusOK, data)
}

// GetLast24HoursBuoyData returns buoy data from the last 24 hours for a specific buoy.
func (bc *BuoyController) GetLast24HoursBuoyData(c *gin.Context) {
	// buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")
	// Calculate time range
	endTime := time.Now().UTC()
	startTime := endTime.AddDate(0, 0, -1) // 7 days ago

	// Get the data range
	data, err := bc.getBuoyDataRange(c.Request.Context(), c.Query("buoyName"), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(data) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data found for the last week"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetMultipleBuoyData returns the most recent data for multiple specified buoys.
func (bc *BuoyController) GetMultipleBuoyData(c *gin.Context) {
	buoysStr := c.Query("buoys")
	buoys := strings.Split(buoysStr, ",")
	var values []map[string]interface{}

	for _, buoy := range buoys {
		var data map[string]interface{}
		data = bc.getBuoyData(c.Request.Context(), buoy)
		if data != nil {
			values = append(values, data)
		}
	}

	c.JSON(http.StatusOK, values)
}

func (bc *BuoyController) getBuoyDataRange(ctx context.Context, buoyName string, startTime, endTime time.Time) ([]map[string]interface{}, error) {
	data, err := bc.buoys.GetDataRange(ctx, buoyName, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("error querying buoy data: %v", err)
	}

	items := make([]map[string]interface{}, 0, len(data))
	for _, entry := range data {
		if entry == nil {
			continue
		}
		items = append(items, buoyDataToMap(entry))
	}

	return items, nil
}

func (bc *BuoyController) getBuoyData(ctx context.Context, buoyName string) map[string]interface{} {
	data, err := bc.buoys.GetLiveData(ctx, buoyName)
	if err != nil {
		slog.Warn("error querying buoy data", slog.Any("error", err))
		return nil
	}
	return buoyDataToMap(data)
}

// GetRegionBuoys returns buoy location information for a specific region.
func (bc *BuoyController) GetRegionBuoys(c *gin.Context) {
	regionName := c.Query("region")
	var buoys []model.Buoy
	locations, err := bc.buoys.GetLocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to query"})
		return
	}

	for name, location := range locations {
		if location == nil {
			continue
		}
		if regionName != "" && !strings.EqualFold(location.Region, regionName) {
			continue
		}
		buoys = append(buoys, model.Buoy{
			RegionBuoy: fmt.Sprintf("%s_%s", location.Region, name),
			Name:       name,
			Latitude:   location.Latitude,
			Longitude:  location.Longitude,
		})
	}

	if len(buoys) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no buoys found for this region"})
		return
	}
	c.JSON(http.StatusOK, buoys)
}

func buoyLocationToMap(name string, location *model.BuoyLocation) map[string]interface{} {
	if location == nil {
		return nil
	}
	return map[string]interface{}{
		"name":      name,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"region":    location.Region,
		"country":   location.Country,
		"spot":      location.Spot,
	}
}

func buoyDataToMap(data *model.BuoyData) map[string]interface{} {
	if data == nil {
		return nil
	}
	return map[string]interface{}{
		"wave_height":     data.WaveHeight,
		"max_period":      data.MaxPeriod,
		"wave_period":     data.WavePeriod,
		"wave_direction":  data.WaveDirection,
		"wind_speed":      data.WindSpeed,
		"wind_direction":  data.WindDirection,
		"temperature":     data.Temperature,
		"pressure":        data.Pressure,
		"dataDateTime":      data.Timestamp.UTC().Format(time.RFC3339),
	}
}
