package api

import (
	"log"
	"net/http"
	"os"
	"strings"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/controller"
	"treblesurf-backend/internal/model"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware handles CORS for the API with iOS app support
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For iOS app, you might want to restrict origins in production
		origin := c.GetHeader("Origin")
		env := os.Getenv("GO_ENV")
		
		// Allow all origins in development or when GO_ENV is not set (local development)
		if env == "development" || env == "" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			// In production, restrict to your domains
			allowedOrigins := []string{
				"https://treblesurf.com",
				"https://www.treblesurf.com",
				// Add your iOS app's custom URL scheme if you have one
			}
			
			if contains(allowedOrigins, origin) {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cookie")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-CSRF-Token, Set-Cookie")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// iOSHeadersMiddleware adds headers that help with iOS app integration
func iOSHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add headers that help with iOS app integration
		c.Header("X-App-Version", "1.0.0")
		c.Header("X-Platform", "iOS")
		
		// Ensure proper cache control for iOS
		if strings.Contains(c.Request.URL.Path, "/auth/") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		
		c.Next()
	}
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func AdminMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        email, exists := c.Get("email")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
            return
        }
        
        emailStr, ok := email.(string)
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
            return
        }
        
        if !isAdminUser(emailStr) {
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
	// Use the APIKeyService from the service registry
	if controller.APIKeyService == nil {
		log.Printf("APIKeyService is not initialized")
		return nil, false
	}
	
	return controller.APIKeyService.ValidateAPIKey(keyValue, requiredScope)
}

