// Package controller provides HTTP handlers for API endpoints.
package controller

import (
	"log/slog"
	"net/http"

	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyController handles API key routes.
type APIKeyController struct {
	apiKeys *service.APIKeyService
}

func NewAPIKeyController(apiKeys *service.APIKeyService) *APIKeyController {
	return &APIKeyController{apiKeys: apiKeys}
}

// CreateAPIKeyHandler handles requests to create a new API key
func (ac *APIKeyController) CreateAPIKeyHandler(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var request struct {
		Description string   `json:"description" binding:"required"`
		Scopes      []string `json:"scopes" binding:"required"`
		ExpiryDays  int      `json:"expiry_days"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	apiKey, err := ac.apiKeys.GenerateAPIKey(
		request.Description,
		emailStr,
		request.ExpiryDays,
		request.Scopes,
	)
	if err != nil {
		requestLogger(c).Warn("failed to generate API key", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Store the API key
	err = ac.apiKeys.StoreAPIKey(c.Request.Context(), apiKey)
	if err != nil {
		requestLogger(c).Warn("failed to store API key", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store API key"})
		return
	}

	// This is the only time the client will see the full key
	c.JSON(http.StatusCreated, gin.H{
		"message": "API key created successfully",
		"key": gin.H{
			"key_id":      apiKey.KeyID,
			"key_value":   apiKey.KeyValue,
			"description": apiKey.Description,
			"created_by":  apiKey.CreatedBy,
			"created_at":  apiKey.CreatedAt,
			"expires_at":  apiKey.ExpiresAt,
			"scopes":      apiKey.Scopes,
		},
	})
}

// ListAPIKeysHandler returns all API keys for the current user
func (ac *APIKeyController) ListAPIKeysHandler(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	apiKeys, err := ac.apiKeys.ListAPIKeys(c.Request.Context(), emailStr)
	if err != nil {
		requestLogger(c).Warn("failed to list API keys", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API keys"})
		return
	}

	// Convert to response format (without key values for security)
	var responseKeys []gin.H
	for _, key := range apiKeys {
		responseKeys = append(responseKeys, gin.H{
			"key_id":      key.KeyID,
			"description": key.Description,
			"created_by":  key.CreatedBy,
			"created_at":  key.CreatedAt,
			"expires_at":  key.ExpiresAt,
			"scopes":      key.Scopes,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"keys":  responseKeys,
		"count": len(responseKeys),
	})
}

// RevokeAPIKeyHandler deletes an API key
func (ac *APIKeyController) RevokeAPIKeyHandler(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	keyID := c.Param("keyID")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID required"})
		return
	}

	apiKeys, err := ac.apiKeys.ListAPIKeys(c.Request.Context(), emailStr)
	if err != nil {
		requestLogger(c).Warn("failed to verify API key ownership", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify key ownership"})
		return
	}

	// Check if the key exists and belongs to the user
	keyExists := false
	for _, key := range apiKeys {
		if key.KeyID == keyID {
			keyExists = true
			break
		}
	}

	if !keyExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found or access denied"})
		return
	}

	// Delete the API key
	err = ac.apiKeys.RevokeAPIKey(c.Request.Context(), keyID)
	if err != nil {
		requestLogger(c).Warn("failed to revoke API key", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}
