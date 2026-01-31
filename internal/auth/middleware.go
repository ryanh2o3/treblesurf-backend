// Package auth provides authentication and authorization middleware and services.
package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware function that validates user sessions.
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		setCacheHeaders(c)
		s.logRequestDetails(c)

		if !s.handleSessionAuth(c) {
			slog.Warn("authentication failed - denying access")
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
func (s *Service) logRequestDetails(c *gin.Context) {
	slog.Debug("auth middleware request",
		slog.String("path", c.Request.URL.Path),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.String("method", c.Request.Method),
		slog.String("origin", c.GetHeader("Origin")),
		slog.String("referer", c.GetHeader("Referer")),
	)
}

// handleSessionAuth handles session authentication and returns true if successful.
func (s *Service) handleSessionAuth(c *gin.Context) bool {
	if s.sessionService == nil {
		slog.Warn("session service is nil")
		return false
	}

	userSession, err := s.sessionService.GetUserSession(c.Request)
	switch {
	case err != nil:
		slog.Warn("session service error", slog.Any("error", err))
		return false
	case userSession != nil:
		return s.processValidSession(c, userSession)
	default:
		slog.Debug("no valid session found")
		return false
	}
}

// processValidSession processes a valid user session.
func (s *Service) processValidSession(c *gin.Context, userSession *user.Session) bool {
	slog.Debug("valid session found", slog.String("user_id", userSession.UserID))
	c.Set("email", userSession.UserID)
	c.Set("session", userSession)

	s.processSessionData(c, userSession)

	// Extend session
	if err := s.sessionService.ExtendUserSession(userSession, c.Request, c.Writer); err != nil {
		slog.Warn("failed to extend user session", slog.Any("error", err))
	}

	c.Next()
	return true
}

// processSessionData processes session JSON data and updates CSRF token if needed.
func (s *Service) processSessionData(c *gin.Context, userSession *user.Session) {
	var sessionData SessionJSON
	if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err != nil {
		return
	}

	sessionData.LastActive = time.Now()

	// Refresh CSRF token if it's getting old (older than 1 hour)
	if time.Since(sessionData.CreatedAt) > time.Hour {
		slog.Debug("refreshing CSRF token", slog.String("user_id", userSession.UserID))
		newCSRFToken, err := GenerateCSRFToken()
		if err == nil {
			sessionData.CSRF = newCSRFToken
			sessionData.CreatedAt = time.Now()
			slog.Debug("CSRF token refreshed successfully")
		}
	}

	updatedJSON, err := json.Marshal(sessionData)
	if err != nil {
		slog.Warn("failed to marshal session data", slog.Any("error", err))
	} else {
		userSession.JSON = string(updatedJSON)
	}

	if sessionData.CSRF != "" {
		c.Header("X-CSRF-Token", sessionData.CSRF)
		slog.Debug("CSRF token set in header")
	}
}

// CSRFMiddleware returns a Gin middleware function that adds CSRF protection to routes that modify state.
func (s *Service) CSRFMiddleware() gin.HandlerFunc {
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
		if s.sessionService != nil {
			userSession, err := s.sessionService.GetUserSession(c.Request)
			if err == nil && userSession != nil {
				// Parse session JSON to get CSRF token
				var sessionData SessionJSON
				if json.Unmarshal([]byte(userSession.JSON), &sessionData) == nil {
					serverToken = sessionData.CSRF
				}
			}
		}

		// If no session token, try to get from cookie as fallback
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
