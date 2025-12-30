// Package auth provides authentication and authorization middleware and services.
package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware function that validates user sessions.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		setCacheHeaders(c)
		logRequestDetails(c)

		if !handleSessionAuth(c) {
			log.Printf("Authentication failed - denying access")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		}
	}
}

// setCacheHeaders sets cache control headers.
func setCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
}

// logRequestDetails logs request details for debugging.
func logRequestDetails(c *gin.Context) {
	log.Printf("=== Auth Middleware ===")
	log.Printf("Path: %s", c.Request.URL.Path)
	log.Printf("User-Agent: %s", c.Request.UserAgent())
	log.Printf("Method: %s", c.Request.Method)
	log.Printf("Origin: %s", c.GetHeader("Origin"))
	log.Printf("Referer: %s", c.GetHeader("Referer"))
}

// handleSessionAuth handles session authentication and returns true if successful.
func handleSessionAuth(c *gin.Context) bool {
	if sessionService == nil {
		log.Printf("Session service is nil")
		return false
	}

	userSession, err := sessionService.GetUserSession(c.Request)
	switch {
	case err != nil:
		log.Printf("Session service error: %v", err)
		return false
	case userSession != nil:
		return processValidSession(c, userSession)
	default:
		log.Printf("No valid session found")
		return false
	}
}

// processValidSession processes a valid user session.
func processValidSession(c *gin.Context, userSession *user.Session) bool {
	log.Printf("Valid session found for user: %s", userSession.UserID)
	c.Set("email", userSession.UserID)
	c.Set("session", userSession)

	processSessionData(c, userSession)

	// Extend session
	if err := sessionService.ExtendUserSession(userSession, c.Request, c.Writer); err != nil {
		log.Printf("Failed to extend user session: %v", err)
	}

	c.Next()
	return true
}

// processSessionData processes session JSON data and updates CSRF token if needed.
func processSessionData(c *gin.Context, userSession *user.Session) {
	var sessionData SessionJSON
	if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err != nil {
		return
	}

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

	updatedJSON, err := json.Marshal(sessionData)
	if err != nil {
		log.Printf("Failed to marshal session data: %v", err)
	} else {
		userSession.JSON = string(updatedJSON)
	}

	if sessionData.CSRF != "" {
		c.Header("X-CSRF-Token", sessionData.CSRF)
		log.Printf("CSRF token set in header: %s", sessionData.CSRF)
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
