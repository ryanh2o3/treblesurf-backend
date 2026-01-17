// Package httphandler provides HTTP API middleware for authentication, CORS, and request handling.
package httphandler

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

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
		c.Header("Access-Control-Allow-Headers",
			"Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cookie")
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

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// AdminMiddleware returns a Gin middleware function that validates admin user permissions.
// Uses config-based admin list instead of hardcoded values.
func AdminMiddleware() gin.HandlerFunc {
	return AdminMiddlewareWithConfig(nil)
}

// AdminMiddlewareWithConfig returns admin middleware with explicit config.
func AdminMiddlewareWithConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, exists := c.Get("email")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		emailStr, ok := email.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		isAdmin := false
		if cfg != nil {
			isAdmin = cfg.IsAdmin(emailStr)
		} else {
			isAdmin = isAdminUser(emailStr)
		}

		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}

// APIKeyAuthMiddleware returns a Gin middleware function that validates API key
// authentication with the specified scope.

func APIKeyAuthMiddleware(apiKeyService *service.APIKeyService, requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "ApiKey ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key format"})
			return
		}

		keyValue := strings.TrimPrefix(authHeader, "ApiKey ")
		apiKey, valid := validateAPIKey(c.Request.Context(), apiKeyService, keyValue, requiredScope)

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
func DevAuthMiddleware(authService *auth.Service) gin.HandlerFunc {
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

		// Check if this is a validate request and create a mock session if needed
		// This replicates the old DevAuthMiddleware functionality
		if strings.Contains(c.Request.URL.Path, "/auth/validate") {
			if err := authService.CreateDevSession(email, c); err != nil {
				// Log the error but don't fail the request
				// This is development mode, so we want to be lenient
				slog.Warn("failed to create dev session", slog.Any("error", err))
			}
		}

		c.Next()
	}
}

// DevAdminAuthMiddleware automatically authenticates requests as an admin in local development
func DevAdminAuthMiddleware(authService *auth.Service) gin.HandlerFunc {
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
			"Theme":      constants.DefaultUserTheme,
			"Picture":    "https://via.placeholder.com/150",
			"Role":       "admin",
		})

		// Check if this is a validate request and create a mock session if needed
		if strings.Contains(c.Request.URL.Path, "/auth/validate") {
			if err := authService.CreateDevSession(email, c); err != nil {
				slog.Warn("failed to create dev admin session", slog.Any("error", err))
			}
		}

		c.Next()
	}
}

// Helper functions for middleware

// isAdminUser checks if an email is in the hardcoded admin list.
// Deprecated: Use config.IsAdmin() instead for production.
func isAdminUser(email string) bool {
	// Fallback admin users - prefer using config.IsAdmin() in production
	adminUsers := map[string]bool{
		"ryancpatton0@gmail.com": true,
	}
	return adminUsers[email]
}

// RateLimitMiddleware implements a simple rate limiter using token bucket algorithm.
func RateLimitMiddleware(requestsPerSecond int) gin.HandlerFunc {
	if requestsPerSecond <= 0 {
		requestsPerSecond = 100 // Default
	}

	limiter := newRateLimiter(requestsPerSecond)

	return func(c *gin.Context) {
		ip := getClientIPFromContext(c)
		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}

// rateLimiter implements a simple token bucket rate limiter.
type rateLimiter struct {
	mu             sync.Mutex
	clients        map[string]*clientBucket
	rps            int
	cleanupTicker  *time.Ticker
}

type clientBucket struct {
	tokens    float64
	lastCheck time.Time
}

func newRateLimiter(rps int) *rateLimiter {
	rl := &rateLimiter{
		clients:       make(map[string]*clientBucket),
		rps:           rps,
		cleanupTicker: time.NewTicker(time.Minute),
	}

	// Cleanup old entries periodically
	go func() {
		for range rl.cleanupTicker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.clients[clientID]

	if !exists {
		rl.clients[clientID] = &clientBucket{
			tokens:    float64(rl.rps) - 1,
			lastCheck: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.lastCheck).Seconds()
	bucket.tokens += elapsed * float64(rl.rps)
	if bucket.tokens > float64(rl.rps) {
		bucket.tokens = float64(rl.rps)
	}
	bucket.lastCheck = now

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--
	return true
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for id, bucket := range rl.clients {
		if bucket.lastCheck.Before(cutoff) {
			delete(rl.clients, id)
		}
	}
}

func getClientIPFromContext(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func validateAPIKey(
	ctx context.Context,
	apiKeyService *service.APIKeyService,
	keyValue, requiredScope string,
) (*model.APIKey, bool) {
	if apiKeyService == nil {
		slog.Warn("APIKeyService is not initialized")
		return nil, false
	}

	return apiKeyService.ValidateAPIKey(ctx, keyValue, requiredScope)
}
