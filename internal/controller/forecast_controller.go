package controller

import (
	"log"
	"net/http"
	"sort"
	"strings"

	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ForecastController struct {
	forecastService *service.ForecastService
	tideService     *service.TideService
}

func NewForecastController(
	forecastService *service.ForecastService,
	tideService *service.TideService,
) *ForecastController {
	return &ForecastController{
		forecastService: forecastService,
		tideService:     tideService,
	}
}

func (c *ForecastController) GetSpotForecast(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecast, err := c.forecastService.GetSpotForecast(spotName, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if forecast != nil {
		ctx.JSON(http.StatusOK, forecast)
		return
	}
	
	ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func (c *ForecastController) GetListSpotsForecast(ctx *gin.Context) {
	spotsStr := ctx.Query("spots")
	spots := strings.Split(spotsStr, ",")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	
	log.Print(spots)

	forecasts, err := c.forecastService.GetListSpotsForecast(spots, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(forecasts) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecasts found"})
		return
	}

	ctx.JSON(http.StatusOK, forecasts)
}

func (c *ForecastController) GetRegionForecast(ctx *gin.Context) {
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecasts, err := c.forecastService.GetRegionForecast(regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(forecasts) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found"})
		return
	}

	// Sort the forecasts by country, region, and spot
	sort.Slice(forecasts, func(i, j int) bool {
		return forecasts[i]["country_region_spot"].(string) < forecasts[j]["country_region_spot"].(string)
	})

	ctx.JSON(http.StatusOK, forecasts)
}

func (c *ForecastController) GetCurrentWeather(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecast, err := c.forecastService.GetCurrentWeather(spotName, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if forecast != nil {
		ctx.JSON(http.StatusOK, forecast)
		return
	}

	ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func (c *ForecastController) GetBeforeAfterTides(ctx *gin.Context) {
	locationName := ctx.Query("locationName")

	prevTide, nextTide := c.tideService.GetBeforeAfterTides(locationName)

	ctx.JSON(http.StatusOK, gin.H{"prevTide": prevTide, "nextTide": nextTide})
}

func (c *ForecastController) GetDayTides(ctx *gin.Context) {
	locationName := ctx.Query("locationName")
	startDay := ctx.Param("startDay")

	tideData := c.tideService.GetDayTides(locationName, startDay)

	ctx.JSON(http.StatusOK, tideData)
}