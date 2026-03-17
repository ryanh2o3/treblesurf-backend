package controller

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ForecastController struct {
	forecastService *service.ForecastService
	tideService     *service.TideService
}

const sourcesAllParamValue = "all"

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
	if ctx.Query("sources") == sourcesAllParamValue {
		c.getSpotForecastGrouped(ctx)
		return
	}
	sourceParam := ctx.Query("source") // optional: client can request a specific source (e.g. ?source=weatherkit)
	c.handleForecastRequest(
		ctx,
		sourceParam,
		func(ctx context.Context, spot, region, country, source string) ([]*model.Forecast, error) {
			return c.forecastService.GetSpotForecast(ctx, spot, region, country, source)
		},
		"failed to get spot forecast",
		"Failed to get forecast",
	)
}

func (c *ForecastController) getSpotForecastGrouped(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	groups, err := c.forecastService.GetSpotForecastGrouped(ctx.Request.Context(), spotName, regionName, countryName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecast not found"})
			return
		}
		requestLogger(ctx).Warn("failed to get spot forecast grouped", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecast"})
		return
	}
	if len(groups) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found"})
		return
	}
	ctx.JSON(http.StatusOK, mapForecastGroupsToClientResponse(groups))
}

func (c *ForecastController) GetListSpotsForecast(ctx *gin.Context) {
	spotsStr := ctx.Query("spots")
	spots := strings.Split(spotsStr, ",")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	sourceParam := ctx.Query("source")

	requestLogger(ctx).Info("forecast spots requested", slog.Any("spots", spots))

	forecasts, err := c.forecastService.GetListSpotsForecast(ctx.Request.Context(), spots, regionName, countryName, sourceParam)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecasts not found"})
			return
		}
		requestLogger(ctx).Warn("failed to get spot forecasts", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecasts"})
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
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecast not found"})
			return
		}
		requestLogger(ctx).Warn("failed to get region forecast", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecast"})
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
	if ctx.Query("sources") == sourcesAllParamValue {
		c.getCurrentWeatherGrouped(ctx)
		return
	}
	sourceParam := ctx.Query("source")
	c.handleForecastRequest(
		ctx,
		sourceParam,
		func(ctx context.Context, spot, region, country, source string) ([]*model.Forecast, error) {
			return c.forecastService.GetCurrentWeather(ctx, spot, region, country, source)
		},
		"failed to get current weather",
		"Failed to get current conditions",
	)
}

func (c *ForecastController) getCurrentWeatherGrouped(ctx *gin.Context) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	groups, err := c.forecastService.GetSpotForecastGrouped(ctx.Request.Context(), spotName, regionName, countryName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecast not found"})
			return
		}
		requestLogger(ctx).Warn("failed to get current weather grouped", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current conditions"})
		return
	}
	if len(groups) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecast found in the last 48 hours"})
		return
	}
	// Current conditions: return only the first time slice (nearest future) with all sources.
	ctx.JSON(http.StatusOK, mapForecastGroupsToClientResponse(groups[:1]))
}

func (c *ForecastController) handleForecastRequest(
	ctx *gin.Context,
	preferredSource string,
	fetchFunc func(context.Context, string, string, string, string) ([]*model.Forecast, error),
	logMsg string,
	errorMsg string,
) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")

	forecast, err := fetchFunc(ctx.Request.Context(), spotName, regionName, countryName, preferredSource)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecast not found"})
			return
		}
		requestLogger(ctx).Warn(logMsg, slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
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