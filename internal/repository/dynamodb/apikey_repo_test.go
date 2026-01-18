package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

const apiKeyTableName = "TestAPIKeysTable"

func TestNewAPIKeyRepo(t *testing.T) {
	repo := NewAPIKeyRepo(nil, apiKeyTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != apiKeyTableName {
		t.Fatalf("expected table name %s, got %s", apiKeyTableName, repo.tableName)
	}
}

func TestAPIKeyItem_FromModel_ToModel(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour)
	apiKey := &model.APIKey{
		KeyID:       "test-key-id-123",
		KeyValue:    "test-key-value",
		Description: "Test API Key",
		CreatedBy:   "test@example.com",
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
		Scopes:      []string{"read", "write"},
	}

	// Test conversion: Model -> Item -> Model
	item := apiKeyItemFromModel(apiKey)
	convertedKey := item.toModel()

	if convertedKey.KeyID != apiKey.KeyID {
		t.Errorf("expected KeyID %s, got %s", apiKey.KeyID, convertedKey.KeyID)
	}
	if convertedKey.KeyValue != apiKey.KeyValue {
		t.Errorf("expected KeyValue %s, got %s", apiKey.KeyValue, convertedKey.KeyValue)
	}
	if convertedKey.Description != apiKey.Description {
		t.Errorf("expected Description %s, got %s", apiKey.Description, convertedKey.Description)
	}
	if convertedKey.CreatedBy != apiKey.CreatedBy {
		t.Errorf("expected CreatedBy %s, got %s", apiKey.CreatedBy, convertedKey.CreatedBy)
	}
	if !convertedKey.CreatedAt.Equal(apiKey.CreatedAt) {
		t.Errorf("expected CreatedAt %v, got %v", apiKey.CreatedAt, convertedKey.CreatedAt)
	}
	if !convertedKey.ExpiresAt.Equal(apiKey.ExpiresAt) {
		t.Errorf("expected ExpiresAt %v, got %v", apiKey.ExpiresAt, convertedKey.ExpiresAt)
	}
	if len(convertedKey.Scopes) != len(apiKey.Scopes) {
		t.Errorf("expected %d scopes, got %d", len(apiKey.Scopes), len(convertedKey.Scopes))
	}
	if len(convertedKey.Scopes) == 2 && (convertedKey.Scopes[0] != apiKey.Scopes[0] || convertedKey.Scopes[1] != apiKey.Scopes[1]) {
		t.Errorf("expected scopes %v, got %v", apiKey.Scopes, convertedKey.Scopes)
	}
}

func TestAPIKeyItem_FromModel_NilInput(t *testing.T) {
	item := apiKeyItemFromModel(nil)
	if item.KeyID != "" {
		t.Error("expected empty KeyID for nil input")
	}
	if item.KeyValue != "" {
		t.Error("expected empty KeyValue for nil input")
	}
	if item.Description != "" {
		t.Error("expected empty Description for nil input")
	}
}

func TestAPIKeyItem_ToModel_EmptyItem(t *testing.T) {
	item := apiKeyItem{}
	apiKey := item.toModel()

	if apiKey == nil {
		t.Fatal("expected non-nil apiKey")
	}
	if apiKey.KeyID != "" {
		t.Error("expected empty KeyID")
	}
	if apiKey.KeyValue != "" {
		t.Error("expected empty KeyValue")
	}
	if !apiKey.CreatedAt.IsZero() {
		t.Error("expected zero CreatedAt")
	}
}

func TestAPIKeyItem_ScopesHandling(t *testing.T) {
	now := time.Now()
	apiKey := &model.APIKey{
		KeyID:     "test-key-id",
		KeyValue:  "test-key-value",
		CreatedAt: now,
		Scopes:    []string{"read", "write", "admin"},
	}

	item := apiKeyItemFromModel(apiKey)
	convertedKey := item.toModel()

	if len(convertedKey.Scopes) != 3 {
		t.Errorf("expected 3 scopes, got %d", len(convertedKey.Scopes))
	}

	// Test empty scopes
	apiKey.Scopes = []string{}
	item = apiKeyItemFromModel(apiKey)
	convertedKey = item.toModel()

	if len(convertedKey.Scopes) != 0 {
		t.Errorf("expected 0 scopes, got %d", len(convertedKey.Scopes))
	}

	// Test nil scopes
	apiKey.Scopes = nil
	item = apiKeyItemFromModel(apiKey)
	convertedKey = item.toModel()

	if convertedKey.Scopes != nil && len(convertedKey.Scopes) != 0 {
		t.Errorf("expected nil or empty scopes, got %v", convertedKey.Scopes)
	}
}

func TestAPIKeyItem_ExpiresAtHandling(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(30 * 24 * time.Hour)

	apiKey := &model.APIKey{
		KeyID:     "test-key-id",
		KeyValue:  "test-key-value",
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	item := apiKeyItemFromModel(apiKey)
	convertedKey := item.toModel()

	if !convertedKey.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expected ExpiresAt %v, got %v", expiresAt, convertedKey.ExpiresAt)
	}

	// Test zero ExpiresAt
	apiKey.ExpiresAt = time.Time{}
	item = apiKeyItemFromModel(apiKey)
	convertedKey = item.toModel()

	if !convertedKey.ExpiresAt.IsZero() {
		t.Errorf("expected zero ExpiresAt, got %v", convertedKey.ExpiresAt)
	}
}

// Note: Full repository method tests (Create, GetByKey, List, Revoke) would require
// DynamoDB client mocking or Localstack integration tests.
// These tests focus on model/item conversions and validation logic that can be
// tested without AWS SDK dependencies.
