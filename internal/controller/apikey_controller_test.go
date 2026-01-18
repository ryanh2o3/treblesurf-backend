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

func setupAPIKeyController(repo *mockrepo.APIKeyRepo) *APIKeyController {
	svc, _ := service.NewAPIKeyService(repo)
	return NewAPIKeyController(svc)
}

func TestAPIKeyController_CreateAPIKeyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{
		CreateFn: func(_ context.Context, key *model.APIKey) error {
			if key.CreatedBy != "test@example.com" {
				t.Errorf("unexpected created by: %s", key.CreatedBy)
			}
			if key.Description != "Test API Key" {
				t.Errorf("unexpected description: %s", key.Description)
			}
			return nil
		},
	}

	controller := setupAPIKeyController(repo)

	requestData := map[string]interface{}{
		"description": "Test API Key",
		"scopes":      []string{"stream", "read"},
		"expiry_days": 30,
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/apiKeys", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.CreateAPIKeyHandler(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "API key created successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}

	keyData, ok := response["key"].(map[string]interface{})
	if !ok {
		t.Fatal("expected key data in response")
	}

	if keyData["description"] != "Test API Key" {
		t.Errorf("expected description 'Test API Key', got %v", keyData["description"])
	}

	if keyData["key_value"] == nil || keyData["key_value"] == "" {
		t.Error("expected key_value to be present")
	}
}

func TestAPIKeyController_CreateAPIKeyHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	requestData := map[string]interface{}{
		"description": "Test API Key",
		"scopes":      []string{"stream"},
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/apiKeys", bytes.NewBuffer(jsonData))
	// No email in context

	controller.CreateAPIKeyHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAPIKeyController_CreateAPIKeyHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/apiKeys", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.CreateAPIKeyHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIKeyController_CreateAPIKeyHandler_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	requestData := map[string]interface{}{
		"description": "Test API Key",
		// Missing scopes
	}

	jsonData, _ := json.Marshal(requestData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/apiKeys", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.CreateAPIKeyHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIKeyController_ListAPIKeysHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	repo := &mockrepo.APIKeyRepo{
		ListFn: func(_ context.Context) ([]*model.APIKey, error) {
			return []*model.APIKey{
				{
					KeyID:       "key_123",
					Description: "Test Key 1",
					CreatedBy:   "test@example.com",
					CreatedAt:   now,
					ExpiresAt:   now.Add(30 * 24 * time.Hour),
					Scopes:      []string{"stream"},
				},
				{
					KeyID:       "key_456",
					Description: "Test Key 2",
					CreatedBy:   "other@example.com",
					CreatedAt:   now,
					ExpiresAt:   now.Add(60 * 24 * time.Hour),
					Scopes:      []string{"read"},
				},
				{
					KeyID:       "key_789",
					Description: "Test Key 3",
					CreatedBy:   "test@example.com",
					CreatedAt:   now,
					ExpiresAt:   now.Add(90 * 24 * time.Hour),
					Scopes:      []string{"read", "write"},
				},
			}, nil
		},
	}

	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apiKeys", http.NoBody)
	c.Set("email", "test@example.com")

	controller.ListAPIKeysHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	keys, ok := response["keys"].([]interface{})
	if !ok {
		t.Fatal("expected keys array in response")
	}

	// Should only return keys created by test@example.com (2 keys)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	count, ok := response["count"].(float64)
	if !ok || int(count) != 2 {
		t.Errorf("expected count to be 2, got %v", count)
	}

	// Verify keys don't include key_value (for security)
	firstKey, ok := keys[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected first key to be an object")
	}

	if firstKey["key_value"] != nil {
		t.Error("key_value should not be present in list response for security")
	}
}

func TestAPIKeyController_ListAPIKeysHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/apiKeys", http.NoBody)
	// No email in context

	controller.ListAPIKeysHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAPIKeyController_RevokeAPIKeyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	keyToRevoke := "key_123"

	repo := &mockrepo.APIKeyRepo{
		ListFn: func(_ context.Context) ([]*model.APIKey, error) {
			return []*model.APIKey{
				{
					KeyID:     keyToRevoke,
					CreatedBy: "test@example.com",
					CreatedAt: now,
				},
			}, nil
		},
		RevokeFn: func(_ context.Context, keyID string) error {
			if keyID != keyToRevoke {
				t.Errorf("unexpected key ID to revoke: %s", keyID)
			}
			return nil
		},
	}

	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/apiKeys/"+keyToRevoke, http.NoBody)
	c.Set("email", "test@example.com")
	c.Params = gin.Params{{Key: "keyID", Value: keyToRevoke}}

	controller.RevokeAPIKeyHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "API key revoked successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}
}

func TestAPIKeyController_RevokeAPIKeyHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/apiKeys/key_123", http.NoBody)
	// No email in context

	controller.RevokeAPIKeyHandler(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAPIKeyController_RevokeAPIKeyHandler_MissingKeyID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{}
	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/apiKeys/", http.NoBody)
	c.Set("email", "test@example.com")
	c.Params = gin.Params{}

	controller.RevokeAPIKeyHandler(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAPIKeyController_RevokeAPIKeyHandler_KeyNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockrepo.APIKeyRepo{
		ListFn: func(_ context.Context) ([]*model.APIKey, error) {
			return []*model.APIKey{}, nil
		},
	}

	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/apiKeys/nonexistent_key", http.NoBody)
	c.Set("email", "test@example.com")
	c.Params = gin.Params{{Key: "keyID", Value: "nonexistent_key"}}

	controller.RevokeAPIKeyHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}

func TestAPIKeyController_RevokeAPIKeyHandler_OtherUserKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	repo := &mockrepo.APIKeyRepo{
		ListFn: func(_ context.Context) ([]*model.APIKey, error) {
			return []*model.APIKey{
				{
					KeyID:     "key_123",
					CreatedBy: "other@example.com", // Different user
					CreatedAt: now,
				},
			}, nil
		},
	}

	controller := setupAPIKeyController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/apiKeys/key_123", http.NoBody)
	c.Set("email", "test@example.com")
	c.Params = gin.Params{{Key: "keyID", Value: "key_123"}}

	controller.RevokeAPIKeyHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusNotFound, w.Code, w.Body.String())
	}
}
