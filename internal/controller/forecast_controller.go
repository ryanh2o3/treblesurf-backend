package controller

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"treblesurf-backend/internal/model"
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

	forecast, err := c.forecastService.GetSpotForecast(ctx.Request.Context(), spotName, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if len(forecast) > 0 {
		ctx.JSON(http.StatusOK, mapForecastsToClient(forecast))
		return
	}
	
	ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func (c *ForecastController) GetListSpotsForecast(ctx *gin.Context) {
	spotsStr := ctx.Query("spots")
	spots := strings.Split(spotsStr, ",")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	
	slog.Info("forecast spots requested", slog.Any("spots", spots))

	forecasts, err := c.forecastService.GetListSpotsForecast(ctx.Request.Context(), spots, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(forecasts) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecasts found"})
		return
	}

	ctx.JSON(http.StatusOK, mapForecastGroupsToClient(forecasts))
}

func (c *ForecastController) GetRegionForecast(ctx *gin.Context) {
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecasts, err := c.forecastService.GetRegionForecast(ctx.Request.Context(), regionName, countryName)
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
		return compareSpot(forecasts[i], forecasts[j])
	})

	ctx.JSON(http.StatusOK, mapForecastsToClient(forecasts))
}

func (c *ForecastController) GetCurrentWeather(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecast, err := c.forecastService.GetCurrentWeather(ctx.Request.Context(), spotName, regionName, countryName)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if len(forecast) > 0 {
		ctx.JSON(http.StatusOK, mapForecastsToClient(forecast))
		return
	}

	ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
}

func compareSpot(a, b *model.Forecast) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return false
	}
	if b == nil {
		return true
	}
	return a.CountryRegionSpot < b.CountryRegionSpot
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