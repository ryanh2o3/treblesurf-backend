package api

import (
	"log"
	"net/http"
	"strings"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware handles CORS for the API
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        email, exists := c.Get("email")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
            return
        }
        
        if !isAdminUser(email.(string)) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
            return
        }
        
        c.Next()
    }
}

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
            "Theme":      "dark",
            "Picture":    "https://via.placeholder.com/150",
        })
        
        // Check if this is a validate request and create a mock session if needed
        // This replicates the old DevAuthMiddleware functionality
        if strings.Contains(c.Request.URL.Path, "/auth/validate") {
            // Import the auth package to access the session service
            // Note: This creates a dependency on the auth package
            if err := auth.CreateDevSession(email, c); err != nil {
                // Log the error but don't fail the request
                // This is development mode, so we want to be lenient
                log.Printf("Failed to create dev session: %v", err)
            }
        }
        
        c.Next()
    }
}

// DevAdminAuthMiddleware automatically authenticates requests as an admin in local development
func DevAdminAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Set a mock authenticated admin user
        email := "admin@example.com"
        c.Set("email", email)
        c.Set("authenticated", true)
        c.Set("user", map[string]interface{}{
            "email":      email,
            "name":       "Admin User",
            "GivenName":  "Admin",
            "FamilyName": "User",
            "Theme":      "dark",
            "Picture":    "https://via.placeholder.com/150",
            "Role":       "admin",
        })
        
        // Check if this is a validate request and create a mock session if needed
        if strings.Contains(c.Request.URL.Path, "/auth/validate") {
            if err := auth.CreateDevSession(email, c); err != nil {
                log.Printf("Failed to create dev admin session: %v", err)
            }
        }
        
        c.Next()
    }
}

// Helper functions for middleware

// isAdminUser checks if a user has admin privileges
func isAdminUser(email string) bool {
	adminUsers := map[string]bool{
		"ryancpatton0@gmail.com": true,
	}
	
	return adminUsers[email]
}

// validateAPIKey validates an API key against DynamoDB
func validateAPIKey(keyValue string, requiredScope string) (*model.APIKey, bool) {
	// TODO: This function needs access to the database client
	// For now, return false to indicate invalid key
	// In production, this should use the proper database client
	return nil, false
}

