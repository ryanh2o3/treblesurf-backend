package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        c.Header("Pragma", "no-cache")

        // TODO: Implement proper session authentication
        // For now, deny access

        // If no valid session, deny access - don't fall back to JWT
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
    }
}

// CSRFMiddleware adds CSRF protection to routes that modify state
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only check POST, PUT, DELETE requests
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }

        // Get token from header
        clientToken := c.GetHeader("X-CSRF-Token")
        
        // Get token from session/cookie
        serverToken, err := c.Cookie("csrf_token")
        if err != nil {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token missing",
            })
            return
        }

        // Validate token
        if clientToken == "" || clientToken != serverToken {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token invalid",
            })
            return
        }

        c.Next()
    }
}