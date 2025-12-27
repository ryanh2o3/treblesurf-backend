// Package auth provides authentication and authorization middleware and services.
package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware returns a Gin middleware function that validates user sessions.
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        c.Header("Pragma", "no-cache")

        // Log request details for debugging
        log.Printf("=== Auth Middleware ===")
        log.Printf("Path: %s", c.Request.URL.Path)
        log.Printf("User-Agent: %s", c.Request.UserAgent())
        log.Printf("Method: %s", c.Request.Method)
        log.Printf("Origin: %s", c.GetHeader("Origin"))
        log.Printf("Referer: %s", c.GetHeader("Referer"))

        // Only try session auth, no fallback to JWT
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            switch {
            case err != nil:
                log.Printf("Session service error: %v", err)
            case userSession != nil:
                log.Printf("Valid session found for user: %s", userSession.UserID)
                // Session is valid
                c.Set("email", userSession.UserID)
                c.Set("session", userSession)
                
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
                    sessionData.LastActive = time.Now()
                    
                    // Refresh CSRF token if it's getting old (older than 1 hour)
                    if time.Since(sessionData.CreatedAt) > time.Hour {
                        log.Printf("Refreshing CSRF token for user: %s", userSession.UserID)
                        newCSRFToken, err := GenerateCSRFToken()
                        if err == nil {
                            sessionData.CSRF = newCSRFToken
                            sessionData.CreatedAt = time.Now()
                            log.Printf("CSRF token refreshed successfully")
                        }
                    }
                    
                    updatedJSON, _ := json.Marshal(sessionData)
                    userSession.JSON = string(updatedJSON)

                    if sessionData.CSRF != "" {
                        c.Header("X-CSRF-Token", sessionData.CSRF)
                        log.Printf("CSRF token set in header: %s", sessionData.CSRF)
                    }
                }
                
                // Extend session
                _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)
                
                c.Next()
                return
            default:
                log.Printf("No valid session found")
            }
        } else {
            log.Printf("Session service is nil")
        }

        // If no valid session, deny access - don't fall back to JWT
        log.Printf("Authentication failed - denying access")
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
    }
}

// CSRFMiddleware returns a Gin middleware function that adds CSRF protection to routes that modify state.
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only check POST, PUT, DELETE requests
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }

        // Get token from header
        clientToken := c.GetHeader("X-CSRF-Token")
        
        // Try to get token from session data first (preferred method)
        var serverToken string
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            if err == nil && userSession != nil {
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if json.Unmarshal([]byte(userSession.JSON), &sessionData) == nil {
                    serverToken = sessionData.CSRF
                }
            }
        }
        
        // If no session token, try to get from cookie as fallback
        if serverToken == "" {
            if cookieToken, err := c.Cookie("csrf_token"); err == nil {
                serverToken = cookieToken
            }
        }
        
        // If still no server token, deny access
        if serverToken == "" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token missing",
            })
            return
        }

        // Validate token - must match exactly
        if clientToken == "" || clientToken != serverToken {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token invalid",
            })
            return
        }

        c.Next()
    }
}