package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func setupForecastController(repo *mockrepo.ForecastRepo, loc *mockrepo.LocationRepo) *ForecastController {
	if loc == nil {
		loc = &mockrepo.LocationRepo{}
	}
	forecastSvc, _ := service.NewForecastService(repo, loc)
	tideSvc := service.NewTideService(&config.Config{Env: config.EnvDevelopment})
	return NewForecastController(forecastSvc, tideSvc)
}

func TestForecastController_GetSpotForecast(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{
					CountryRegionSpot: country + "_" + region + "_" + spot,
					ForecastTimestamp: "1700000000#imi_swan+weatherkit",
					Source:            "imi_swan+weatherkit",
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data: map[string]interface{}{
						"swellHeight": 2.5,
						"windSpeed":   15.0,
					},
				},
			}, nil
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 forecast response, got %d", len(response))
	}
	if response[0]["data"] == nil || response[0]["forecast_timestamp"] == nil || response[0]["spot_id"] == nil {
		t.Fatalf("expected iOS-compatible forecast keys, got: %#v", response[0])
	}
}

func TestForecastController_GetSpotForecast_InvalidGranularity(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, _, _, _ string, _ []string, _ string) ([]*model.Forecast, error) {
			t.Fatal("repository should not be called when granularity is invalid")
			return nil, nil
		},
	}
	controller := setupForecastController(repo, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Bundoran&granularity=invalid", http.NoBody)
	controller.GetSpotForecast(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestForecastController_GetSpotForecast_NotFound(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, _, _, _ string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{}, nil
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Unknown", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestForecastController_GetSpotForecast_SourcesAll_ReturnsGrouped(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: "stormglass", SpotID: country + "#" + region + "#" + spot},
				{ForecastTimestamp: "1700000000#imi_swan+weatherkit", Source: "imi_swan+weatherkit", SpotID: country + "#" + region + "#" + spot},
			}, nil
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Bundoran&sources=all", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 group, got %d", len(response))
	}
	group := response[0]
	if group["sources"] == nil {
		t.Fatalf("expected sources map in grouped response")
	}
	sources, ok := group["sources"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sources to be map, got %T", group["sources"])
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources (stormglass + imi_swan+weatherkit), got %d", len(sources))
	}
	if group["primary_source"] != "imi_swan+weatherkit" {
		t.Fatalf("expected primary_source imi_swan+weatherkit, got %v", group["primary_source"])
	}
}

func TestForecastController_GetSpotForecast_SourceParam_ReturnsRequestedSource(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{ForecastTimestamp: "1700000000#stormglass", Source: "stormglass", SpotID: country + "#" + region + "#" + spot},
				{ForecastTimestamp: "1700000000#weatherkit", Source: "weatherkit", SpotID: country + "#" + region + "#" + spot},
			}, nil
		},
	}
	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/forecast?country=Ireland&region=Donegal&spot=Bundoran&source=weatherkit", http.NoBody)

	controller.GetSpotForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected 1 forecast, got %d", len(response))
	}
	if response[0]["source"] != "weatherkit" {
		t.Fatalf("expected source=weatherkit from query param, got %v", response[0]["source"])
	}
}

func TestForecastController_GetListSpotsForecast(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{
				{
					CountryRegionSpot: country + "_" + region + "_" + spot,
					ForecastTimestamp: "1700000000#imi_swan+weatherkit",
					Source:            "imi_swan+weatherkit",
					Country:           country,
					Region:            region,
					Spot:              spot,
					Data:              map[string]interface{}{"swellHeight": 1.0},
				},
			}, nil
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	requestPath := "/listSpotsForecast?country=Ireland&region=Donegal&spots=Bundoran,Rossnowlagh"
	c.Request = httptest.NewRequest(http.MethodGet, requestPath, http.NoBody)

	controller.GetListSpotsForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response [][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 spot forecasts, got %d", len(response))
	}
}

func TestForecastController_GetCurrentWeather(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetCurrentConditionsFn: func(_ context.Context, country, region, spot string, _ []string) (*model.Forecast, error) {
			return &model.Forecast{
				CountryRegionSpot: country + "_" + region + "_" + spot,
				ForecastTimestamp: "1700000000#imi_swan+weatherkit",
				Source:            "imi_swan+weatherkit",
				Country:           country,
				Region:            region,
				Spot:              spot,
				Data: map[string]interface{}{
					"swellHeight": 2.0, "temperature": 18.5,
				},
			}, nil
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/currentConditions?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.GetCurrentWeather(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(response) != 1 || response[0]["data"] == nil {
		t.Fatalf("expected iOS-compatible current conditions, got: %#v", response)
	}
}

func TestForecastController_GetCurrentWeather_NotFound(t *testing.T) {
	repo := &mockrepo.ForecastRepo{
		GetCurrentConditionsFn: func(_ context.Context, _, _, _ string, _ []string) (*model.Forecast, error) {
			return nil, repository.ErrNotFound
		},
	}

	controller := setupForecastController(repo, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/currentConditions?country=Ireland&region=Donegal&spot=Unknown", http.NoBody)

	controller.GetCurrentWeather(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestForecastController_GetRegionForecast(t *testing.T) {
	loc := &mockrepo.LocationRepo{
		GetSpotsFn: func(_ context.Context, country, region string) ([]*model.LocationInfo, error) {
			return []*model.LocationInfo{
				{CountryRegionSpot: country + "/" + region + "/Bundoran"},
				{CountryRegionSpot: country + "/" + region + "/Rossnowlagh"},
			}, nil
		},
	}
	repo := &mockrepo.ForecastRepo{
		GetSpotForecastFn: func(_ context.Context, country, region, spot string, _ []string, _ string) ([]*model.Forecast, error) {
			return []*model.Forecast{{
				CountryRegionSpot: country + "_" + region + "_" + spot,
				Source:            "imi_swan+weatherkit",
			}}, nil
		},
	}

	controller := setupForecastController(repo, loc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionForecast?country=Ireland&region=Donegal", http.NoBody)

	controller.GetRegionForecast(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestForecastController_GetBeforeAfterTides(t *testing.T) {
	controller := setupForecastController(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/beforeAfterTides?locationName=Bundoran", http.NoBody)

	controller.GetBeforeAfterTides(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["prevTide"] == nil || response["nextTide"] == nil {
		t.Error("expected prevTide and nextTide in response")
	}
}

func TestForecastController_GetDayTides(t *testing.T) {
	controller := setupForecastController(nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/dayTides/Bundoran/2024-01-15", http.NoBody)
	c.Params = []gin.Param{{Key: "startDay", Value: "2024-01-15"}}

	controller.GetDayTides(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestForecastController_CompareSpot(t *testing.T) {
	t.Run("both nil returns false", func(t *testing.T) {
		result := compareSpot(nil, nil)
		if result {
			t.Error("expected false when both are nil")
		}
	})

	t.Run("first nil returns false", func(t *testing.T) {
		b := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Bundoran"}
		result := compareSpot(nil, b)
		if result {
			t.Error("expected false when first is nil")
		}
	})

	t.Run("second nil returns true", func(t *testing.T) {
		a := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Bundoran"}
		result := compareSpot(a, nil)
		if !result {
			t.Error("expected true when second is nil")
		}
	})

	t.Run("a < b returns true", func(t *testing.T) {
		a := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Bundoran"}
		b := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Rossnowlagh"}
		result := compareSpot(a, b)
		if !result {
			t.Error("expected true when a < b")
		}
	})

	t.Run("a > b returns false", func(t *testing.T) {
		a := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Rossnowlagh"}
		b := &model.Forecast{CountryRegionSpot: "Ireland_Donegal_Bundoran"}
		result := compareSpot(a, b)
		if result {
			t.Error("expected false when a > b")
		}
	})
}
