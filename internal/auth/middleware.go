package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        c.Header("Pragma", "no-cache")

        // Only try session auth, no fallback to JWT
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            if err == nil && userSession != nil {
                // Session is valid
                c.Set("email", userSession.UserID)
                c.Set("session", userSession)
                
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
                    sessionData.LastActive = time.Now()
                    updatedJSON, _ := json.Marshal(sessionData)
                    userSession.JSON = string(updatedJSON)

                    if sessionData.CSRF != "" {
                        c.Header("X-CSRF-Token", sessionData.CSRF)
                    }
                }
                
                // Extend session
                _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)
                
                c.Next()
                return
            }
        }

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