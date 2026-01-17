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

	service := NewAPIKeyService(repo)
	if _, ok := service.ValidateAPIKey(ctx, "key", "stream"); !ok {
		t.Fatalf("expected key to be valid for stream scope")
	}
	if _, ok := service.ValidateAPIKey(ctx, "key", "write"); ok {
		t.Fatalf("expected key to be invalid for write scope")
	}
}
