package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	locationTestCountry = "Ireland"
	locationTestRegion  = "Donegal"
	locationTestSpot    = "Bundoran"
)

func setupLocationController(repo *mockrepo.LocationRepo) *LocationController {
	svc, _ := service.NewLocationService(repo, "https://cdn.example.com")
	return NewLocationController(svc)
}

func TestLocationController_GetRegions(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetRegionsFn: func(_ context.Context, country string) ([]string, error) {
			if country != locationTestCountry {
				t.Errorf("unexpected country: %s", country)
			}
			return []string{locationTestRegion, "Clare", "Kerry"}, nil
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regions?country="+locationTestCountry, http.NoBody)

	controller.GetRegions(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 3 {
		t.Fatalf("expected 3 regions, got %d", len(response))
	}
}

func TestLocationController_GetRegions_MissingCountry(t *testing.T) {
	repo := &mockrepo.LocationRepo{}
	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regions", http.NoBody)

	controller.GetRegions(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLocationController_GetRegions_ServiceError(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetRegionsFn: func(_ context.Context, _ string) ([]string, error) {
			return nil, context.DeadlineExceeded
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/regions?country="+locationTestCountry, http.NoBody)

	controller.GetRegions(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLocationController_GetSpots(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetSpotsFn: func(_ context.Context, country, region string) ([]*model.LocationInfo, error) {
			if country != locationTestCountry || region != locationTestRegion {
				t.Errorf("unexpected location: %s/%s", country, region)
			}
			return []*model.LocationInfo{
				{CountryRegionSpot: country + "_" + region + "_" + locationTestSpot, Latitude: 54.5, Longitude: -8.3},
				{CountryRegionSpot: country + "_" + region + "_Rossnowlagh", Latitude: 54.6, Longitude: -8.4},
			}, nil
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/spots?country="+locationTestCountry+"&region="+locationTestRegion, http.NoBody)

	controller.GetSpots(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []model.LocationInfo
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 spots, got %d", len(response))
	}
}

func TestLocationController_GetSpots_MissingParams(t *testing.T) {
	repo := &mockrepo.LocationRepo{}
	controller := setupLocationController(repo)

	tests := []struct {
		name string
		url  string
	}{
		{"missing country", "/spots?region=" + locationTestRegion},
		{"missing region", "/spots?country=" + locationTestCountry},
		{"missing both", "/spots"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)

			controller.GetSpots(c)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestLocationController_GetCoordinates(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, country, region, spot string) (float64, float64, error) {
			if country != locationTestCountry || region != locationTestRegion || spot != locationTestSpot {
				t.Errorf("unexpected location: %s/%s/%s", country, region, spot)
			}
			return 54.5, -8.3, nil
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	locationURL := "/location?country=" + locationTestCountry +
		"&region=" + locationTestRegion + "&spot=" + locationTestSpot
	c.Request = httptest.NewRequest(http.MethodGet, locationURL, http.NoBody)

	controller.GetCoordinates(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []float64
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(response))
	}
	if response[0] != 54.5 {
		t.Errorf("expected latitude 54.5, got %f", response[0])
	}
	if response[1] != -8.3 {
		t.Errorf("expected longitude -8.3, got %f", response[1])
	}
}

func TestLocationController_GetLocationInfo(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetLocationInfoFn: func(_ context.Context, country, region, spot string) (*model.LocationInfo, error) {
			if country != locationTestCountry || region != locationTestRegion || spot != locationTestSpot {
				t.Errorf("unexpected location: %s/%s/%s", country, region, spot)
			}
			return &model.LocationInfo{
				CountryRegionSpot: country + "_" + region + "_" + spot,
				Latitude:          54.5,
				Longitude:         -8.3,
			}, nil
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	locationInfoURL := "/locationInfo?country=" + locationTestCountry +
		"&region=" + locationTestRegion + "&spot=" + locationTestSpot
	c.Request = httptest.NewRequest(http.MethodGet, locationInfoURL, http.NoBody)

	controller.GetLocationInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response *model.LocationInfo
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Latitude != 54.5 {
		t.Errorf("expected latitude 54.5, got %f", response.Latitude)
	}
}

func TestLocationController_GetLocationInfo_NotFound(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
			return nil, repository.ErrNotFound
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	missingInfoURL := "/locationInfo?country=" + locationTestCountry +
		"&region=" + locationTestRegion + "&spot=Unknown"
	c.Request = httptest.NewRequest(http.MethodGet, missingInfoURL, http.NoBody)

	controller.GetLocationInfo(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestLocationController_GetSpots_ServiceError(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetSpotsFn: func(_ context.Context, _, _ string) ([]*model.LocationInfo, error) {
			return nil, context.DeadlineExceeded
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/spots?country="+locationTestCountry+"&region="+locationTestRegion, http.NoBody)

	controller.GetSpots(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLocationController_GetLocationInfo_ServiceError(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
			return nil, context.DeadlineExceeded
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	infoServiceURL := "/locationInfo?country=" + locationTestCountry +
		"&region=" + locationTestRegion + "&spot=" + locationTestSpot
	c.Request = httptest.NewRequest(http.MethodGet, infoServiceURL, http.NoBody)

	controller.GetLocationInfo(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLocationController_GetLocationInfo_MissingParams(t *testing.T) {
	repo := &mockrepo.LocationRepo{}
	controller := setupLocationController(repo)

	tests := []struct {
		name string
		url  string
	}{
		{"missing country", "/locationInfo?region=" + locationTestRegion + "&spot=" + locationTestSpot},
		{"missing region", "/locationInfo?country=" + locationTestCountry + "&spot=" + locationTestSpot},
		{"missing spot", "/locationInfo?country=" + locationTestCountry + "&region=" + locationTestRegion},
		{"missing all", "/locationInfo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)

			controller.GetLocationInfo(c)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestLocationController_GetCoordinates_ServiceError(t *testing.T) {
	repo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, _, _, _ string) (float64, float64, error) {
			return 0, 0, context.DeadlineExceeded
		},
	}

	controller := setupLocationController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	coordinatesURL := "/coordinates?country=" + locationTestCountry +
		"&region=" + locationTestRegion + "&spot=" + locationTestSpot
	c.Request = httptest.NewRequest(http.MethodGet, coordinatesURL, http.NoBody)

	controller.GetCoordinates(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLocationController_GetCoordinates_MissingParams(t *testing.T) {
	repo := &mockrepo.LocationRepo{}
	controller := setupLocationController(repo)

	tests := []struct {
		name string
		url  string
	}{
		{"missing country", "/coordinates?region=" + locationTestRegion + "&spot=" + locationTestSpot},
		{"missing region", "/coordinates?country=" + locationTestCountry + "&spot=" + locationTestSpot},
		{"missing spot", "/coordinates?country=" + locationTestCountry + "&region=" + locationTestRegion},
		{"missing all", "/coordinates"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, tt.url, http.NoBody)

			controller.GetCoordinates(c)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}
