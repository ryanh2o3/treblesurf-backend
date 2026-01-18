package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	testBuoyNameM4 = "M4"
	buoyTestRegion     = "Atlantic"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
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
	c.Request = httptest.NewRequest(http.MethodGet, "/getLiveBuoyData", http.NoBody)

	controller.GetLiveBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []buoyDataResponse
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
			if buoyName != testBuoyNameM4 {
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
	c.Request = httptest.NewRequest(http.MethodGet, "/getSingleBuoyData?buoyName=M4", http.NoBody)

	controller.GetSingleBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestBuoyController_GetRegionBuoys(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				testBuoyNameM4: {Name: testBuoyNameM4, Region: buoyTestRegion, Latitude: 51.0, Longitude: -10.0},
				"M5":           {Name: "M5", Region: buoyTestRegion, Latitude: 51.5, Longitude: -9.5},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionBuoys?region=Atlantic", http.NoBody)

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
				testBuoyNameM4: {Name: testBuoyNameM4, Region: buoyTestRegion},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionBuoys?region=Pacific", http.NoBody)

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
	c.Request = httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/getBuoyDataRange?buoyName=%s&startTime=2024-01-01&endTime=2024-01-02", testBuoyNameM4),
		http.NoBody,
	)

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
	c.Request = httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/getBuoyDataRange?buoyName=%s&startTime=invalid&endTime=2024-01-02", testBuoyNameM4),
		http.NoBody,
	)

	controller.GetBuoyDataRange(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBuoyController_BuoyLocationInfo(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				testBuoyNameM4: {
					Name:      testBuoyNameM4,
					Region:    buoyTestRegion,
					Country:   "Ireland",
					Latitude:  51.0,
					Longitude: -10.0,
				},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/buoyLocationInfo", http.NoBody)

	controller.BuoyLocationInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []buoyLocationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 location, got %d", len(response))
	}
}

func TestBuoyController_IndividualBuoyLocationInfo(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				testBuoyNameM4: {
					Name:      testBuoyNameM4,
					Region:    buoyTestRegion,
					Country:   "Ireland",
					Latitude:  51.0,
					Longitude: -10.0,
				},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/individualBuoyLocationInfo?buoyName=%s", testBuoyNameM4), http.NoBody)

	controller.IndividualBuoyLocationInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response buoyLocationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Name != testBuoyNameM4 {
		t.Fatalf("expected buoy name %s, got %s", testBuoyNameM4, response.Name)
	}
}

func TestBuoyController_IndividualBuoyLocationInfo_WithRegion(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				buoyTestRegion + "_" + testBuoyNameM4: {
					Name:      testBuoyNameM4,
					Region:    buoyTestRegion,
					Country:   "Ireland",
					Latitude:  51.0,
					Longitude: -10.0,
				},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/individualBuoyLocationInfo?buoyName=%s&region=%s", testBuoyNameM4, buoyTestRegion),
		http.NoBody,
	)

	controller.IndividualBuoyLocationInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestBuoyController_IndividualBuoyLocationInfo_NotFound(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/individualBuoyLocationInfo?buoyName=Unknown", http.NoBody)

	controller.IndividualBuoyLocationInfo(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestBuoyController_GetLast24HoursBuoyData(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetDataRangeFn: func(_ context.Context, buoyName string, _, _ time.Time) ([]*model.BuoyData, error) {
			if buoyName != testBuoyNameM4 {
				t.Fatalf("unexpected buoy name: %s", buoyName)
			}
			return []*model.BuoyData{
				{BuoyName: buoyName, WaveHeight: 2.0, Timestamp: time.Now().Add(-12 * time.Hour)},
				{BuoyName: buoyName, WaveHeight: 2.5, Timestamp: time.Now()},
			}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/getLast24HoursBuoyData?buoyName=%s", testBuoyNameM4), http.NoBody)

	controller.GetLast24HoursBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []buoyDataResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 data points, got %d", len(response))
	}
}

func TestBuoyController_GetLast24HoursBuoyData_NotFound(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetDataRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]*model.BuoyData, error) {
			return []*model.BuoyData{}, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/getLast24HoursBuoyData?buoyName=%s", testBuoyNameM4), http.NoBody)

	controller.GetLast24HoursBuoyData(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestBuoyController_GetMultipleBuoyData(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
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
	c.Request = httptest.NewRequest(http.MethodGet, "/getMultipleBuoyData?buoys=M4,M5,Blackstones", http.NoBody)

	controller.GetMultipleBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []buoyDataResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 3 {
		t.Fatalf("expected 3 data points, got %d", len(response))
	}
}

func TestBuoyController_GetMultipleBuoyData_EmptyQuery(t *testing.T) {
	repo := &mockrepo.BuoyRepo{
		GetLiveDataFn: func(_ context.Context, _ string) (*model.BuoyData, error) {
			return nil, nil
		},
	}

	controller := setupBuoyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getMultipleBuoyData?buoys=", http.NoBody)

	controller.GetMultipleBuoyData(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []buoyDataResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 0 {
		t.Fatalf("expected empty response, got %d items", len(response))
	}
}
