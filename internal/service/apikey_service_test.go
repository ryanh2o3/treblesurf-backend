package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestAPIKeyService_ValidateAPIKey_ScopeCheck(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.APIKeyRepo{
		GetByKeyFn: func(_ context.Context, key string) (*model.APIKey, error) {
			return &model.APIKey{
				KeyValue:  key,
				ExpiresAt: time.Now().Add(1 * time.Hour),
				Scopes:    []string{"stream", "read"},
			}, nil
		},
	}

	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if _, ok := service.ValidateAPIKey(ctx, "key", "stream"); !ok {
		t.Fatalf("expected key to be valid for stream scope")
	}
	if _, ok := service.ValidateAPIKey(ctx, "key", "write"); ok {
		t.Fatalf("expected key to be invalid for write scope")
	}
}

func TestAPIKeyService_ValidateAPIKey_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.APIKeyRepo{
		GetByKeyFn: func(_ context.Context, key string) (*model.APIKey, error) {
			return &model.APIKey{
				KeyValue:  key,
				ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
				Scopes:    []string{"stream"},
			}, nil
		},
	}

	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if _, ok := service.ValidateAPIKey(ctx, "key", "stream"); ok {
		t.Fatalf("expected expired key to be invalid")
	}
}

func TestAPIKeyService_GenerateAPIKey(t *testing.T) {
	repo := &mockrepo.APIKeyRepo{}
	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	apiKey, err := service.GenerateAPIKey("test key", "admin@example.com", 30, []string{"stream", "read"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if apiKey.KeyID == "" {
		t.Fatalf("expected key ID to be generated")
	}
	if apiKey.KeyValue == "" {
		t.Fatalf("expected key value to be generated")
	}
	if apiKey.Description != "test key" {
		t.Fatalf("unexpected description: %s", apiKey.Description)
	}
	if apiKey.CreatedBy != "admin@example.com" {
		t.Fatalf("unexpected created by: %s", apiKey.CreatedBy)
	}
	if len(apiKey.Scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(apiKey.Scopes))
	}
}

func TestAPIKeyService_StoreAPIKey(t *testing.T) {
	ctx := context.Background()
	stored := false
	repo := &mockrepo.APIKeyRepo{
		CreateFn: func(_ context.Context, _ *model.APIKey) error {
			stored = true
			return nil
		},
	}

	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	apiKey := &model.APIKey{KeyID: "test-key-id"}
	if err := service.StoreAPIKey(ctx, apiKey); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stored {
		t.Fatalf("expected StoreAPIKey to be called")
	}
}

func TestAPIKeyService_RevokeAPIKey(t *testing.T) {
	ctx := context.Background()
	revoked := false
	repo := &mockrepo.APIKeyRepo{
		RevokeFn: func(_ context.Context, keyID string) error {
			revoked = true
			if keyID != "test-key-id" {
				t.Fatalf("unexpected key ID: %s", keyID)
			}
			return nil
		},
	}

	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if err := service.RevokeAPIKey(ctx, "test-key-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revoked {
		t.Fatalf("expected RevokeAPIKey to be called")
	}
}

func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	ctx := context.Background()
	allKeys := []*model.APIKey{
		{KeyID: "key1", CreatedBy: "user1@example.com"},
		{KeyID: "key2", CreatedBy: "user2@example.com"},
		{KeyID: "key3", CreatedBy: "user1@example.com"},
	}
	repo := &mockrepo.APIKeyRepo{
		ListFn: func(_ context.Context) ([]*model.APIKey, error) {
			return allKeys, nil
		},
	}

	service, err := NewAPIKeyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	// Filter by user1@example.com
	got, err := service.ListAPIKeys(ctx, "user1@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 keys for user1, got %d", len(got))
	}

	// Verify all returned keys belong to user1
	for _, key := range got {
		if key.CreatedBy != "user1@example.com" {
			t.Fatalf("expected key created by user1@example.com, got %s", key.CreatedBy)
		}
	}

	// Filter by user2@example.com
	got, err = service.ListAPIKeys(ctx, "user2@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 key for user2, got %d", len(got))
	}

	// Filter by non-existent user
	got, err = service.ListAPIKeys(ctx, "nonexistent@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 keys for nonexistent user, got %d", len(got))
	}
}

func TestNewAPIKeyService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewAPIKeyService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}
