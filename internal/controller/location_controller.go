package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// This controller uses the shared service registry

// GetRegions returns a list of regions for the specified country.
func GetRegions(c *gin.Context) {
	countryName := c.Query("country")
	if countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country parameter is required"})
		return
	}

	regions, err := LocationService.GetRegions(countryName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, regions)
}

// GetSpots returns a list of spots for the specified country and region.
func GetSpots(c *gin.Context) {
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both country and region parameters are required"})
		return
	}

	spots, err := LocationService.GetSpots(countryName, regionName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, spots)
}

// GetLocationInfo retrieves detailed information for a specific location.
func GetLocationInfo(c *gin.Context) {
	spotName := c.Query("spot")
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if spotName == "" || regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country, region, and spot parameters are required"})
		return
	}

	location, err := LocationService.GetLocationInfo(countryName, regionName, spotName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, location)
}

// GetCoordinates retrieves latitude and longitude coordinates for a specific location.
func GetCoordinates(c *gin.Context) {
	spotName := c.Query("spot")
	regionName := c.Query("region")
	countryName := c.Query("country")
	
	if spotName == "" || regionName == "" || countryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country, region, and spot parameters are required"})
		return
	}

	coordinates, err := LocationService.GetCoordinates(countryName, regionName, spotName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, coordinates)
}