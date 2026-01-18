package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func setupForecastController(repo *mockrepo.ForecastRepo) *ForecastController {
	forecastSvc, _ := service.NewForecastService(repo)
	tideSvc := service.NewTideService()
	return NewForecastController(forecastSvc, tideSvc)
}

func TestForecastController_GetSpotForecast(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{
					CountryRegionSpot: country + "_" + region + "_" + spot,
					WaveHeight:        2.5,
					WindSpeed:         15.0,
				},
			}, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestForecastController_GetSpotForecast_NotFound(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, _, _, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{}, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Unknown", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestForecastController_GetListSpotsForecast(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{CountryRegionSpot: country + "_" + region + "_" + spot},
			}, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	requestPath := "/listSpotsForecast?country=Ireland&region=Donegal&spots=Bundoran,Rossnowlagh"
	c.Request = httptest.NewRequest(http.MethodGet, requestPath, http.NoBody)

	controller.GetListSpotsForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response [][]*model.Forecast
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 spot forecasts, got %d", len(response))
	}
}

func TestForecastController_GetCurrentWeather(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetCurrentConditionsFn: func(_ context.Context, country, region, spot string) (*model.Forecast, error) {
			return &model.Forecast{
				CountryRegionSpot: country + "_" + region + "_" + spot,
				Temperature:       18.5,
			}, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/currentConditions?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.GetCurrentWeather(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestForecastController_GetCurrentWeather_NotFound(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetCurrentConditionsFn: func(_ context.Context, _, _, _ string) (*model.Forecast, error) {
			return nil, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/currentConditions?country=Ireland&region=Donegal&spot=Unknown", http.NoBody)

	controller.GetCurrentWeather(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestForecastController_GetRegionForecast(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetRegionForecastFn: func(_ context.Context, country, region string, _ time.Time) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{CountryRegionSpot: country + "_" + region + "_Bundoran"},
				{CountryRegionSpot: country + "_" + region + "_Rossnowlagh"},
			}, nil
		},
	}

	controller := setupForecastController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionForecast?country=Ireland&region=Donegal", http.NoBody)

	controller.GetRegionForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
