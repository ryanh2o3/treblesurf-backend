// Package controller provides HTTP handlers for the API endpoints.
package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// BuoyController handles buoy data routes.
type BuoyController struct {
	buoys *service.BuoyService
}

type buoyLocationResponse struct {
	Name      string  `json:"name"`
	RegionBuoy string `json:"region_buoy,omitempty"`
	Region    string  `json:"region"`
	Country   string  `json:"country"`
	Spot      string  `json:"spot"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// NewBuoyController creates a new BuoyController with the given service.
func NewBuoyController(buoys *service.BuoyService) *BuoyController {
	return &BuoyController{buoys: buoys}
}

// BuoyLocationInfo returns location information for all buoys.
func (bc *BuoyController) BuoyLocationInfo(c *gin.Context) {
	locations, err := bc.buoys.GetLocations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]buoyLocationResponse, 0, len(locations))
	for name, location := range locations {
		if response := buoyLocationToResponse(name, location); response != nil {
			results = append(results, *response)
		}
	}

	c.JSON(http.StatusOK, results)
}

// IndividualBuoyLocationInfo returns location information for a specific buoy.
func (bc *BuoyController) IndividualBuoyLocationInfo(c *gin.Context) {
	regionName := c.Query("region")
	buoyName := strings.ReplaceAll(c.Query("buoyName"), " ", "")

	location, err := bc.buoys.GetLocationByName(c.Request.Context(), buoyName)
	if err != nil {
		// Try with region prefix
		if regionName != "" {
			location, err = bc.buoys.GetLocationByName(c.Request.Context(), regionName+"_"+buoyName)
		}
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no buoy location found"})
			return
		}
	}

	c.JSON(http.StatusOK, buoyLocationToResponse(buoyName, location))
}

// GetLiveBuoyData returns the most recent buoy data for all default buoys.
func (bc *BuoyController) GetLiveBuoyData(c *gin.Context) {
	buoyNames := bc.buoys.DefaultBuoys()
	data, err := bc.buoys.GetMultipleBuoysData(c.Request.Context(), buoyNames)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	regionBuoyMap := bc.getRegionBuoyMap(c.Request.Context())
	results := make([]clientBuoyResponse, 0, len(data))
	for _, d := range data {
		if response := buoyDataToClientResponse(d, regionBuoyMap); response != nil {
			results = append(results, *response)
		}
	}

	c.JSON(http.StatusOK, results)
}

// GetBuoyDataRange returns buoy data for a specific buoy within a time range.
func (bc *BuoyController) GetBuoyDataRange(c *gin.Context) {
	buoyName := c.Query("buoyName")
	startTimeStr := c.Query("startTime")
	endTimeStr := c.Query("endTime")

	startTime, err := parseTime(startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid start time format, expected 2006-01-02T15:04:05Z or 2006-01-02",
		})
		return
	}

	endTime, err := parseTime(endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid end time format, expected 2006-01-02T15:04:05Z or 2006-01-02",
		})
		return
	}

	// If only date provided, set end to end of day
	if len(endTimeStr) == 10 {
		endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	}

	data, err := bc.buoys.GetDataRange(c.Request.Context(), buoyName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	regionBuoyMap := bc.getRegionBuoyMap(c.Request.Context())
	results := buoyDataSliceToClientResponses(data, regionBuoyMap)
	c.JSON(http.StatusOK, results)
}

// GetSingleBuoyData returns the most recent data for a specific buoy.
func (bc *BuoyController) GetSingleBuoyData(c *gin.Context) {
	buoyName := c.Query("buoyName")

	var data *model.BuoyData
	var err error

	// Retry up to 12 times for eventual consistency
	for i := 0; i < 12; i++ {
		data, err = bc.buoys.GetLiveData(c.Request.Context(), buoyName)
		if err == nil && data != nil {
			break
		}
	}

	if data == nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	regionBuoyMap := bc.getRegionBuoyMap(c.Request.Context())
	c.JSON(http.StatusOK, buoyDataToClientResponse(data, regionBuoyMap))
}

// GetLast24HoursBuoyData returns buoy data from the last 24 hours for a specific buoy.
func (bc *BuoyController) GetLast24HoursBuoyData(c *gin.Context) {
	buoyName := c.Query("buoyName")

	data, err := bc.buoys.GetLast24HoursData(c.Request.Context(), buoyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(data) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no data found for the last 24 hours"})
		return
	}

	regionBuoyMap := bc.getRegionBuoyMap(c.Request.Context())
	results := buoyDataSliceToClientResponses(data, regionBuoyMap)
	c.JSON(http.StatusOK, results)
}

// GetMultipleBuoyData returns the most recent data for multiple specified buoys.
func (bc *BuoyController) GetMultipleBuoyData(c *gin.Context) {
	buoysStr := c.Query("buoys")
	buoyNames := strings.Split(buoysStr, ",")

	data, err := bc.buoys.GetMultipleBuoysData(c.Request.Context(), buoyNames)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	regionBuoyMap := bc.getRegionBuoyMap(c.Request.Context())
	results := buoyDataSliceToClientResponses(data, regionBuoyMap)
	c.JSON(http.StatusOK, results)
}

// GetRegionBuoys returns buoy location information for a specific region.
func (bc *BuoyController) GetRegionBuoys(c *gin.Context) {
	regionName := c.Query("region")

	buoys, err := bc.buoys.GetRegionBuoys(c.Request.Context(), regionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to query"})
		return
	}

	if len(buoys) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no buoys found for this region"})
		return
	}

	c.JSON(http.StatusOK, buoys)
}

// Helper functions

func parseTime(s string) (time.Time, error) {
	// Try full timestamp first
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err == nil {
		return t, nil
	}
	// Try date only
	return time.Parse("2006-01-02", s)
}

func buoyLocationToResponse(name string, location *model.BuoyLocation) *buoyLocationResponse {
	if location == nil {
		return nil
	}
	return &buoyLocationResponse{
		Name:      name,
		Latitude:  location.Latitude,
		Longitude: location.Longitude,
		Region:    location.Region,
		Country:   location.Country,
		Spot:      location.Spot,
		RegionBuoy: location.RegionBuoy,
	}
}

func (bc *BuoyController) getRegionBuoyMap(ctx context.Context) map[string]string {
	locations, err := bc.buoys.GetLocations(ctx)
	if err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(locations))
	for name, location := range locations {
		if location == nil {
			continue
		}
		regionBuoy := location.RegionBuoy
		if regionBuoy == "" && location.Region != "" && name != "" {
			regionBuoy = fmt.Sprintf("%s_%s", location.Region, name)
		}
		if regionBuoy == "" {
			regionBuoy = name
		}
		out[name] = regionBuoy
	}
	return out
}

func buoyDataToClientResponse(data *model.BuoyData, regionBuoyMap map[string]string) *clientBuoyResponse {
	if data == nil {
		return nil
	}

	regionBuoy := regionBuoyMap[data.BuoyName]
	if regionBuoy == "" {
		regionBuoy = data.BuoyName
	}

	dataDateTime := ""
	if !data.Timestamp.IsZero() {
		dataDateTime = data.Timestamp.UTC().Format(time.RFC3339)
	}

	return &clientBuoyResponse{
		WaveHeight:          floatPtr(data.WaveHeight),
		WavePeriod:          floatPtr(data.WavePeriod),
		MaxPeriod:           floatPtr(data.MaxPeriod),
		MeanWaveDirection:   floatPtr(data.WaveDirection),
		WindSpeed:           floatPtr(data.WindSpeed),
		WindDirection:       floatPtr(data.WindDirection),
		SeaTemperature:      floatPtr(data.Temperature),
		AtmosphericPressure: floatPtr(data.Pressure),
		DataDateTime:        dataDateTime,
		Name:                data.BuoyName,
		RegionBuoy:          regionBuoy,
	}
}

func buoyDataSliceToClientResponses(data []*model.BuoyData, regionBuoyMap map[string]string) []clientBuoyResponse {
	if data == nil {
		return []clientBuoyResponse{}
	}
	results := make([]clientBuoyResponse, 0, len(data))
	for _, d := range data {
		if response := buoyDataToClientResponse(d, regionBuoyMap); response != nil {
			results = append(results, *response)
		}
	}
	return results
}
