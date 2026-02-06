package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// LocationController handles location routes.
type LocationController struct {
	locations *service.LocationService
}

func NewLocationController(locations *service.LocationService) *LocationController {
	return &LocationController{locations: locations}
}

func (lc *LocationController) GetRegions(c *gin.Context) {
	countryName := c.Query("country")
	if countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country parameter is required"})
		return
	}

	regions, err := lc.locations.GetRegions(c.Request.Context(), countryName)
	if err != nil {
		requestLogger(c).Warn("failed to load regions", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load regions"})
		return
	}

	c.JSON(http.StatusOK, regions)
}

func (lc *LocationController) GetSpots(c *gin.Context) {
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both country and region parameters are required"})
		return
	}

	spots, err := lc.locations.GetSpots(c.Request.Context(), countryName, regionName)
	if err != nil {
		requestLogger(c).Warn("failed to load spots", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load spots"})
		return
	}

	c.JSON(http.StatusOK, spots)
}

func (lc *LocationController) GetLocationInfo(c *gin.Context) {
	spotName := c.Query("spot")
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if spotName == "" || regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country, region, and spot parameters are required"})
		return
	}

	location, err := lc.locations.GetLocationInfo(c.Request.Context(), countryName, regionName, spotName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
			return
		}
		requestLogger(c).Warn("failed to load location info", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load location info"})
		return
	}

	c.JSON(http.StatusOK, location)
}

func (lc *LocationController) GetCoordinates(c *gin.Context) {
	spotName := c.Query("spot")
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if spotName == "" || regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country, region, and spot parameters are required"})
		return
	}

	coordinates, err := lc.locations.GetCoordinates(c.Request.Context(), countryName, regionName, spotName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
			return
		}
		requestLogger(c).Warn("failed to load coordinates", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load coordinates"})
		return
	}

	c.JSON(http.StatusOK, coordinates)
}