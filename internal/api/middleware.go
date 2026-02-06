// Package httphandler provides HTTP API middleware for authentication, CORS, and request handling.
package httphandler

import (
	"compress/gzip"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/logging"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GzipMiddleware compresses API responses when the client supports gzip.
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		writer := &gzipResponseWriter{
			ResponseWriter: c.Writer,
		}
		writer.gzipWriter = gzip.NewWriter(writer.ResponseWriter)
		c.Writer = writer

		c.Next()

		if writer.compressed {
			_ = writer.gzipWriter.Close()
		}
	}
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	gzipWriter *gzip.Writer
	compressed bool
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	header := w.Header()
	contentType := header.Get("Content-Type")
	contentEncoding := header.Get("Content-Encoding")

	if code == http.StatusNoContent || code == http.StatusNotModified ||
		strings.HasPrefix(contentType, "text/event-stream") ||
		contentEncoding != "" {
		w.ResponseWriter.WriteHeader(code)
		return
	}

	w.compressed = true
	header.Del("Content-Length")
	header.Set("Content-Encoding", "gzip")
	header.Set("Vary", addVaryHeaderValue(header.Get("Vary"), "Accept-Encoding"))
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.Written() {
		w.WriteHeader(http.StatusOK)
	}

	if w.compressed {
		return w.gzipWriter.Write(data)
	}

	return w.ResponseWriter.Write(data)
}

func addVaryHeaderValue(existing, value string) string {
	if existing == "" {
		return value
	}

	for _, part := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return existing
		}
	}

	return existing + ", " + value
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

// RequestLoggerMiddleware attaches a request-scoped logger to context.
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		logger := logging.WithRequestID(slog.Default(), requestID)
		ctx := logging.WithLogger(c.Request.Context(), logger)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", requestID)
		c.Header("X-Request-Id", requestID)

		c.Next()
	}
}

// RequestTimeoutMiddleware enforces per-request timeouts.
func RequestTimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if timeout <= 0 {
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) && !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": "Request timed out",
			})
		}
	}
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
func DevAuthMiddleware(cfg *config.Config, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg != nil && !cfg.IsDevelopment() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "development auth is disabled"})
			return
		}

		// Set a mock authenticated user
		email := ""
		if cfg != nil {
			email = cfg.Security.DevUserEmail
		}
		if email == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "development user email not configured"})
			return
		}
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
func DevAdminAuthMiddleware(cfg *config.Config, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg != nil && !cfg.IsDevelopment() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "development auth is disabled"})
			return
		}

		// Set a mock authenticated admin user
		email := ""
		if cfg != nil {
			email = cfg.Security.DevAdminEmail
		}
		if email == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "development admin email not configured"})
			return
		}
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
//
// Deprecated: Use config.IsAdmin() instead for production.
func isAdminUser(email string) bool {
	// Fallback admin users - prefer using config.IsAdmin() in production
	adminUsers := map[string]bool{
		"ryancpatton0@gmail.com": true,
	}
	return adminUsers[email]
}

// RateLimitMiddlewareWithLimiter implements a simple rate limiter using token bucket algorithm.
func RateLimitMiddlewareWithLimiter(limiter *rateLimiter) gin.HandlerFunc {
	if limiter == nil {
		limiter = newRateLimiter(0)
	}

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
	clients       map[string]*clientBucket
	cleanupTicker *time.Ticker
	done          chan struct{}
	rps           int
	mu            sync.Mutex
	stopOnce      sync.Once
}

type clientBucket struct {
	lastCheck time.Time
	tokens    float64
}

func newRateLimiter(rps int) *rateLimiter {
	if rps <= 0 {
		rps = 100
	}
	rl := &rateLimiter{
		clients:       make(map[string]*clientBucket),
		rps:           rps,
		cleanupTicker: time.NewTicker(time.Minute),
		done:          make(chan struct{}),
	}

	// Cleanup old entries periodically
	go func() {
		for {
			select {
			case <-rl.done:
				return
			case <-rl.cleanupTicker.C:
				rl.cleanup()
			}
		}
	}()

	return rl
}

func (rl *rateLimiter) stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
		if rl.cleanupTicker != nil {
			rl.cleanupTicker.Stop()
		}
	})
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
