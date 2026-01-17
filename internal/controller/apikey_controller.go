// Package controller provides HTTP handlers for API endpoints.
package controller

import (
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
	email, exists := c.Get("email")
	if !exists {
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

	// Generate the API key
	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}
	apiKey, err := ac.apiKeys.GenerateAPIKey(
		request.Description,
		emailStr,
		request.ExpiryDays,
		request.Scopes,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Store the API key
	err = ac.apiKeys.StoreAPIKey(apiKey)
	if err != nil {
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
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}

	apiKeys, err := ac.apiKeys.ListAPIKeys(emailStr)
	if err != nil {
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
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	keyID := c.Param("keyID")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID required"})
		return
	}

	// Verify the key belongs to the current user before deleting
	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}
	apiKeys, err := ac.apiKeys.ListAPIKeys(emailStr)
	if err != nil {
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
	err = ac.apiKeys.RevokeAPIKey(keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}
