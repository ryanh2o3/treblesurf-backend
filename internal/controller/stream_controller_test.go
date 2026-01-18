package controller

import (
	"bytes"
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

func setupStreamController(streamSvc *service.StreamService) *StreamController {
	return NewStreamController(streamSvc)
}

func createTestStreamService(repo *mockrepo.StreamRequestRepo) *service.StreamService {
	svc, _ := service.NewStreamService(repo)
	return svc
}

func TestStreamController_RequestStreamHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	repo := &mockrepo.StreamRequestRepo{
		SaveFn: func(_ context.Context, request *model.StreamRequest) error {
			if request.SpotID != "Ireland_Donegal_Bundoran" {
				t.Errorf("unexpected spot ID: %s", request.SpotID)
			}
			if request.RequestedBy != "test@example.com" {
				t.Errorf("unexpected requested by: %s", request.RequestedBy)
			}
			return nil
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	requestData := map[string]interface{}{
		"spot_id": "Ireland_Donegal_Bundoran",
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/requestStream", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.RequestStreamHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "Stream requested successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}

	if response["expires_at"] == nil {
		t.Error("expected expires_at in response")
	}

	// Verify expires_at is a valid RFC3339 timestamp
	expiresAtStr, ok := response["expires_at"].(string)
	if !ok {
		t.Error("expected expires_at to be a string")
	}

	_, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		t.Errorf("expected expires_at to be valid RFC3339 format: %v", err)
	}

	// Verify expiration is in the future (within 6 minutes, considering 5 minute TTL + buffer)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
	if expiresAt.Before(now.Add(4 * time.Minute)) || expiresAt.After(now.Add(6*time.Minute)) {
		t.Errorf("expected expiration to be around 5 minutes from now, got %v", expiresAt)
	}
}

func TestStreamController_RequestStreamHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	requestData := map[string]interface{}{
		"spot_id": "Ireland_Donegal_Bundoran",
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/requestStream", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	// No email in context

	controller.RequestStreamHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestStreamController_RequestStreamHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/requestStream", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.RequestStreamHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestStreamController_RequestStreamHandler_MissingSpotID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	requestData := map[string]interface{}{
		// Missing spot_id
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/requestStream", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.RequestStreamHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestStreamController_RequestStreamHandler_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{
		SaveFn: func(_ context.Context, request *model.StreamRequest) error {
			return context.DeadlineExceeded // Simulate service error
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	requestData := map[string]interface{}{
		"spot_id": "Ireland_Donegal_Bundoran",
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/requestStream", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.RequestStreamHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}
}

func TestStreamController_CheckStreamRequestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	repo := &mockrepo.StreamRequestRepo{
		GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
			if spotID != "Ireland_Donegal_Bundoran" {
				t.Errorf("unexpected spot ID: %s", spotID)
			}
			return &model.StreamRequest{
				SpotID:      spotID,
				RequestedBy: "test@example.com",
				RequestedAt: now,
				Expiration:  now.Add(5 * time.Minute).Unix(),
			}, nil
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/checkStreamRequest?spot_id=Ireland_Donegal_Bundoran", http.NoBody)

	controller.CheckStreamRequestHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["stream_requested"] == nil {
		t.Error("expected stream_requested in response")
	}

	streamRequested, ok := response["stream_requested"].(bool)
	if !ok {
		t.Error("expected stream_requested to be a boolean")
	}

	if !streamRequested {
		t.Error("expected stream_requested to be true")
	}
}

func TestStreamController_CheckStreamRequestHandler_MissingSpotID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/checkStreamRequest", http.NoBody)

	controller.CheckStreamRequestHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestStreamController_CheckStreamRequestHandler_NotRequested(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{
		GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
			return nil, nil // No stream request found
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/checkStreamRequest?spot_id=Ireland_Donegal_Bundoran", http.NoBody)

	controller.CheckStreamRequestHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	streamRequested, ok := response["stream_requested"].(bool)
	if !ok {
		t.Error("expected stream_requested to be a boolean")
	}

	if streamRequested {
		t.Error("expected stream_requested to be false")
	}
}

func TestStreamController_CheckStreamRequestHandler_Expired(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	repo := &mockrepo.StreamRequestRepo{
		GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
			return &model.StreamRequest{
				SpotID:      spotID,
				RequestedBy: "test@example.com",
				RequestedAt: now.Add(-10 * time.Minute), // Requested 10 minutes ago
				Expiration:  now.Add(-5 * time.Minute).Unix(), // Expired 5 minutes ago
			}, nil
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/checkStreamRequest?spot_id=Ireland_Donegal_Bundoran", http.NoBody)

	controller.CheckStreamRequestHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	streamRequested, ok := response["stream_requested"].(bool)
	if !ok {
		t.Error("expected stream_requested to be a boolean")
	}

	if streamRequested {
		t.Error("expected stream_requested to be false for expired request")
	}
}

func TestStreamController_CheckStreamRequestHandler_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{
		GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
			return nil, context.DeadlineExceeded // Simulate service error
		},
	}

	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/checkStreamRequest?spot_id=Ireland_Donegal_Bundoran", http.NoBody)

	controller.CheckStreamRequestHandler(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}
}

// Note: GetStreamingCredentials and GetStreamPlaybackURL directly use AWS SDK clients
// (STS and KinesisVideo) which are difficult to mock without refactoring to use interfaces.
// These tests focus on authentication/authorization validation.
// Full integration testing would require AWS mocks or localstack.

func TestStreamController_GetStreamingCredentials_NoAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/streamingCredentials", http.NoBody)
	// No API key in context

	controller.GetStreamingCredentials(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestStreamController_GetStreamingCredentials_InvalidAPIKeyType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/streamingCredentials", http.NoBody)
	c.Set("apiKey", "not-an-apikey-object") // Wrong type

	controller.GetStreamingCredentials(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestStreamController_GetStreamPlaybackURL_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.StreamRequestRepo{}
	streamSvc := createTestStreamService(repo)
	controller := setupStreamController(streamSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/streamPlaybackURL", http.NoBody)
	// No email in context

	controller.GetStreamPlaybackURL(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
