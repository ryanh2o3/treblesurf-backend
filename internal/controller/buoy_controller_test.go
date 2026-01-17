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

func init() {
	gin.SetMode(gin.TestMode)
}

func setupBuoyController(repo *mockrepo.BuoyRepo) *BuoyController {
	svc, _ := service.NewBuoyService(repo)
	return NewBuoyController(svc)
}

func TestBuoyController_GetLiveBuoyData(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
			return &model.BuoyData{
				BuoyName:   buoyName,
				WaveHeight: 2.5,
				WavePeriod: 12.0,
				Timestamp:  time.Now(),
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getLiveBuoyData", nil)

	controller.GetLiveBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) == 0 {
		t.Fatalf("expected non-empty response")
	}
}

func TestBuoyController_GetSingleBuoyData(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
			if buoyName != "M4" {
				t.Fatalf("unexpected buoy name: %s", buoyName)
			}
			return &model.BuoyData{
				BuoyName:   buoyName,
				WaveHeight: 2.5,
				Timestamp:  time.Now(),
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getSingleBuoyData?buoyName=M4", nil)

	controller.GetSingleBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestBuoyController_GetRegionBuoys(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				"M4": {Name: "M4", Region: "Atlantic", Latitude: 51.0, Longitude: -10.0},
				"M5": {Name: "M5", Region: "Atlantic", Latitude: 51.5, Longitude: -9.5},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionBuoys?region=Atlantic", nil)

	controller.GetRegionBuoys(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []model.Buoy
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 buoys, got %d", len(response))
	}
}

func TestBuoyController_GetRegionBuoys_NotFound(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				"M4": {Name: "M4", Region: "Atlantic"},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionBuoys?region=Pacific", nil)

	controller.GetRegionBuoys(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestBuoyController_GetBuoyDataRange(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetDataRangeFn: func(_ context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error) {
			return []*model.BuoyData{
				{BuoyName: buoyName, WaveHeight: 2.0, Timestamp: start},
				{BuoyName: buoyName, WaveHeight: 2.5, Timestamp: end},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getBuoyDataRange?buoyName=M4&startTime=2024-01-01&endTime=2024-01-02", nil)

	controller.GetBuoyDataRange(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestBuoyController_GetBuoyDataRange_InvalidStartTime(t *testing.T) {
	repo := &mockrepo.BuoyRepo{}
	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getBuoyDataRange?buoyName=M4&startTime=invalid&endTime=2024-01-02", nil)

	controller.GetBuoyDataRange(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBuoyController_BuoyLocationInfo(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				"M4": {Name: "M4", Region: "Atlantic", Country: "Ireland", Latitude: 51.0, Longitude: -10.0},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/buoyLocationInfo", nil)

	controller.BuoyLocationInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 location, got %d", len(response))
	}
}
