// Package service provides business logic services for the application.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

type APIKeyService struct {
	apiKeys repository.APIKeyRepository
}

// NewAPIKeyService creates a new API key service instance.
func NewAPIKeyService(apiKeys repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{
		apiKeys: apiKeys,
	}
}

// GenerateAPIKey creates a new API key with specified parameters
func (s *APIKeyService) GenerateAPIKey(
	description string,
	createdBy string,
	durationDays int,
	scopes []string,
) (*model.APIKey, error) {
	// Generate a random key
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	
	// Generate a unique ID
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	
	keyID := fmt.Sprintf("key_%s", base64.URLEncoding.EncodeToString(idBytes)[:16])
	keyValue := base64.URLEncoding.EncodeToString(b)
	
	// Create expiration date (or far future if 0 days)
	var expiresAt time.Time
	if durationDays <= 0 {
		expiresAt = time.Now().AddDate(10, 0, 0) // 10 years in the future
	} else {
		expiresAt = time.Now().AddDate(0, 0, durationDays)
	}
	
	apiKey := &model.APIKey{
		KeyID:       keyID,
		KeyValue:    keyValue,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Scopes:      scopes,
	}
	
	return apiKey, nil
}

// StoreAPIKey stores an API key using the repository.
func (s *APIKeyService) StoreAPIKey(ctx context.Context, apiKey *model.APIKey) error {
	return s.apiKeys.Create(ctx, apiKey)
}

// ValidateAPIKey validates an API key using the repository.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, keyValue, requiredScope string) (*model.APIKey, bool) {
	apiKey, err := s.apiKeys.GetByKey(ctx, keyValue)
	if err != nil {
		return nil, false
	}
	
	// Check if the key is expired
	if time.Now().After(apiKey.ExpiresAt) {
		return nil, false
	}
	
	// Check if the key has the required scope
	for _, scope := range apiKey.Scopes {
		if scope == requiredScope {
			return apiKey, true
		}
	}
	
	return nil, false
}

// ListAPIKeys retrieves all API keys for a user
func (s *APIKeyService) ListAPIKeys(ctx context.Context, createdBy string) ([]*model.APIKey, error) {
	apiKeys, err := s.apiKeys.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]*model.APIKey, 0, len(apiKeys))
	for _, key := range apiKeys {
		if key.CreatedBy == createdBy {
			filtered = append(filtered, key)
		}
	}
	return filtered, nil
}

// RevokeAPIKey deletes an API key
func (s *APIKeyService) RevokeAPIKey(ctx context.Context, keyID string) error {
	return s.apiKeys.Revoke(ctx, keyID)
}
