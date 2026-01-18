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

const (
	testSwellCountry     = "Ireland"
	testSwellRegion      = "Donegal"
	testSwellSpot        = "Bundoran"
	testSwellSpotIDHash  = "Ireland#Donegal#Bundoran"
	testSwellSpotID      = "Ireland_Donegal_Bundoran"
	testSwellSpotIDRoss  = "Ireland_Donegal_Rossnowlagh"
)

func setupSwellPredictionController(repo *mockrepo.SwellPredictionRepo) *SwellPredictionController {
	svc, _ := service.NewSwellPredictionService(repo)
	return NewSwellPredictionController(svc)
}

func TestSwellPredictionController_GetSpotSwellPrediction(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionsFn: func(_ context.Context, spotID string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			if spotID != testSwellSpotIDHash {
				t.Errorf("unexpected spot ID: %s", spotID)
			}
			return []model.SwellPrediction{
				{SpotID: spotID, Data: map[string]interface{}{"wave_height": 2.5}},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	predictionURL := "/swellPrediction?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spot=" + testSwellSpot
	c.Request = httptest.NewRequest(http.MethodGet, predictionURL, http.NoBody)

	controller.GetSpotSwellPrediction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []model.SwellPrediction
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) == 0 {
		t.Fatal("expected non-empty response")
	}
}

func TestSwellPredictionController_GetSpotSwellPrediction_MissingParams(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{}
	controller := setupSwellPredictionController(repo)

	tests := []struct {
		name string
		url  string
	}{
		{"missing spot", "/swellPrediction?country=" + testSwellCountry + "&region=" + testSwellRegion},
		{"missing region", "/swellPrediction?country=" + testSwellCountry + "&spot=" + testSwellSpot},
		{"missing country", "/swellPrediction?region=" + testSwellRegion + "&spot=" + testSwellSpot},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)

			controller.GetSpotSwellPrediction(c)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestSwellPredictionController_GetSpotSwellPrediction_NotFound(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionsFn: func(_ context.Context, _ string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			return []model.SwellPrediction{}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	missingURL := "/swellPrediction?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spot=Unknown"
	c.Request = httptest.NewRequest(http.MethodGet, missingURL, http.NoBody)

	controller.GetSpotSwellPrediction(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSwellPredictionController_GetListSpotsSwellPrediction(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetListSpotsPredictionsFn: func(_ context.Context, spotIDs []string, _ time.Time, _ int) ([][]model.SwellPrediction, error) {
			result := make([][]model.SwellPrediction, len(spotIDs))
			for i, spotID := range spotIDs {
				result[i] = []model.SwellPrediction{
					{SpotID: spotID, Data: map[string]interface{}{"wave_height": 2.5}},
				}
			}
			return result, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	listURL := "/listSpotsSwellPrediction?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spots=" + testSwellSpot + ",Rossnowlagh"
	c.Request = httptest.NewRequest(http.MethodGet, listURL, http.NoBody)

	controller.GetListSpotsSwellPrediction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response [][]model.SwellPrediction
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 spot predictions, got %d", len(response))
	}
}

func TestSwellPredictionController_GetRegionSwellPrediction(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetRegionPredictionsFn: func(_ context.Context, country, region string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			if country != testSwellCountry || region != testSwellRegion {
				t.Errorf("unexpected location: %s/%s", country, region)
			}
			return []model.SwellPrediction{
				{SpotID: testSwellSpotID, Data: map[string]interface{}{"wave_height": 2.5}},
				{SpotID: testSwellSpotIDRoss, Data: map[string]interface{}{"wave_height": 3.0}},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regionSwellPrediction?country="+testSwellCountry+"&region="+testSwellRegion, http.NoBody)

	controller.GetRegionSwellPrediction(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSwellPredictionController_GetSpotSwellPredictionRange(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionRangeFn: func(_ context.Context, spotID string, _, _ time.Time) ([]model.SwellPrediction, error) {
			if spotID != testSwellSpotIDHash {
				t.Errorf("unexpected spot ID: %s", spotID)
			}
			return []model.SwellPrediction{
				{SpotID: spotID, Data: map[string]interface{}{"wave_height": 2.5}},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Use the correct time format: 2006-01-02T15:00:00Z
	rangeURL := "/swellPredictionRange?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spot=" + testSwellSpot +
		"&startTime=2024-01-01T00:00:00Z&endTime=2024-01-02T00:00:00Z"
	c.Request = httptest.NewRequest(http.MethodGet, rangeURL, http.NoBody)

	controller.GetSpotSwellPredictionRange(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSwellPredictionController_GetRecentSwellPredictions(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetRecentPredictionsFn: func(_ context.Context, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			return []model.SwellPrediction{
				{SpotID: testSwellSpotID, Data: map[string]interface{}{"wave_height": 2.5}},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/recentSwellPredictions?limit=10", http.NoBody)

	controller.GetRecentSwellPredictions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSwellPredictionController_GetSwellPredictionStatus(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionsFn: func(_ context.Context, spotID string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			return []model.SwellPrediction{
				{SpotID: spotID, Data: map[string]interface{}{"wave_height": 2.5}},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	statusURL := "/swellPredictionStatus?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spot=" + testSwellSpot
	c.Request = httptest.NewRequest(http.MethodGet, statusURL, http.NoBody)

	controller.GetSwellPredictionStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSwellPredictionController_GetClosestAIPredictionForSpot(t *testing.T) {
	repo := &mockrepo.SwellPredictionRepo{
		GetClosestPredictionFn: func(_ context.Context, spotID string, _ time.Time) (*model.SwellPrediction, error) {
			if spotID != testSwellSpotIDHash {
				t.Errorf("unexpected spot ID: %s", spotID)
			}
			return &model.SwellPrediction{
				SpotID: spotID,
				Data:   map[string]interface{}{"wave_height": 2.5},
			}, nil
		},
	}

	controller := setupSwellPredictionController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	closestURL := "/closestAIPrediction?country=" + testSwellCountry +
		"&region=" + testSwellRegion + "&spot=" + testSwellSpot
	c.Request = httptest.NewRequest(http.MethodGet, closestURL, http.NoBody)

	controller.GetClosestAIPredictionForSpot(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
