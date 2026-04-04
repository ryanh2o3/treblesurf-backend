package controller

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
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
	c.handleForecastRequest(
		ctx,
		"forecast.spot",
		c.forecastService.GetSpotForecast,
		"failed to get spot forecast",
		"Failed to get forecast",
	)
}

func (c *ForecastController) GetListSpotsForecast(ctx *gin.Context) {
	spotsStr := ctx.Query("spots")
	spots := strings.Split(spotsStr, ",")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	
	log := requestLogger(ctx)
	log.Info("forecast spots requested", slog.Any("spots", spots))

	source := strings.TrimSpace(ctx.Query("source"))
	listStart := time.Now()
	var fetchDur, mapDur time.Duration
	var groupCount int
	defer func() {
		log.Info("forecast request timing",
			slog.String("op", "forecast.list_spots"),
			slog.String("country", countryName),
			slog.String("region", regionName),
			slog.String("source", source),
			slog.Int("spot_count", len(spots)),
			slog.Duration("fetch", fetchDur),
			slog.Duration("map_response", mapDur),
			slog.Duration("total", time.Since(listStart)),
			slog.Int("forecast_groups", groupCount),
		)
	}()

	t0 := time.Now()
	forecasts, err := c.forecastService.GetListSpotsForecast(ctx.Request.Context(), spots, regionName, countryName, source)
	fetchDur = time.Since(t0)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecasts not found"})
			return
		}
		log.Warn("failed to get spot forecasts", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get forecasts"})
		return
	}

	if len(forecasts) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "No forecasts found"})
		return
	}

	t1 := time.Now()
	payload := mapForecastGroupsToClient(forecasts)
	mapDur = time.Since(t1)
	groupCount = len(payload)

	ctx.JSON(http.StatusOK, payload)
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
	c.handleForecastRequest(
		ctx,
		"forecast.current_conditions",
		c.forecastService.GetCurrentWeather,
		"failed to get current weather",
		"Failed to get current conditions",
	)
}

func (c *ForecastController) handleForecastRequest(
	ctx *gin.Context,
	op string,
	fetchFunc func(context.Context, string, string, string, string) ([]*model.Forecast, error),
	logMsg string,
	errorMsg string,
) {
	spotName := ctx.Query("spot")
	regionName := ctx.Query("region")
	countryName := ctx.Query("country")
	source := strings.TrimSpace(ctx.Query("source"))
	log := requestLogger(ctx)
	handlerStart := time.Now()

	var fetchDur, mapDur time.Duration
	var resultCount int

	defer func() {
		log.Info("forecast request timing",
			slog.String("op", op),
			slog.String("country", countryName),
			slog.String("region", regionName),
			slog.String("spot", spotName),
			slog.String("source", source),
			slog.Duration("fetch", fetchDur),
			slog.Duration("map_response", mapDur),
			slog.Duration("total", time.Since(handlerStart)),
			slog.Int("items_returned", resultCount),
		)
	}()

	t0 := time.Now()
	forecast, err := fetchFunc(ctx.Request.Context(), spotName, regionName, countryName, source)
	fetchDur = time.Since(t0)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Forecast not found"})
			return
		}
		log.Warn(logMsg, slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	if len(forecast) > 0 {
		t1 := time.Now()
		payload := mapForecastsToClient(forecast)
		mapDur = time.Since(t1)
		resultCount = len(payload)
		ctx.JSON(http.StatusOK, payload)
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