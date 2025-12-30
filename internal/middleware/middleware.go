// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"strings"
	"treblesurf-backend/internal/constants"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware handles CORS for the API
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers",
			"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// AdminMiddleware returns a Gin middleware function that validates admin user permissions.
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		email, exists := c.Get("email")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		
		emailStr, ok := email.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
			return
		}
		
		if !isAdminUser(emailStr) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		
		c.Next()
	}
}

// APIKeyAuthMiddleware returns a Gin middleware function that validates API key authentication
// with the specified scope.
func APIKeyAuthMiddleware(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "ApiKey ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format"})
			return
		}
		
		keyValue := strings.TrimPrefix(authHeader, "ApiKey ")
		apiKey, valid := validateAPIKey(keyValue, requiredScope)
		
		if !valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired API key"})
			return
		}
		
		// Set the API key in the context for use by handlers
		c.Set("apiKey", apiKey)
		c.Next()
	}
}

// DevAuthMiddleware automatically authenticates requests in local development
func DevAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a mock authenticated user
		email := "testuser@example.com"
		c.Set("email", email)
		c.Set("authenticated", true)
		c.Set("user", map[string]interface{}{
			"email":      email,
			"name":       "Test User",
			"GivenName":  "Test",
			"FamilyName": "User",
			"Theme":      constants.DefaultUserTheme,
			"Picture":    "https://via.placeholder.com/150",
		})
		
		c.Next()
	}
}

// DevAdminAuthMiddleware automatically authenticates requests as an admin in local development
func DevAdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a mock authenticated admin user
		c.Set("email", "admin@example.com")
		c.Set("authenticated", true)
		c.Set("user", map[string]interface{}{
			"email":      "admin@example.com",
			"name":       "Admin User",
			"GivenName":  "Admin",
			"FamilyName": "User",
			"Theme":      constants.DefaultUserTheme,
			"Picture":    "https://via.placeholder.com/150",
			"Role":       "admin",
		})
		c.Next()
	}
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement JWT validation
		c.Next()
	}
}

// CSRFMiddleware validates CSRF tokens
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement CSRF validation
		c.Next()
	}
}

// Helper functions (these would need to be implemented or moved to services)
func isAdminUser(_ string) bool {
	// TODO: Implement admin user check
	return false
}

func validateAPIKey(_, _ string) (interface{}, bool) {
	// TODO: Implement API key validation
	return nil, false
}
