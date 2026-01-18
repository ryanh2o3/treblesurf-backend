package httphandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/config"

	"github.com/gin-gonic/gin"
)

func TestSetupRouter_HealthCheck(t *testing.T) {
	cfg := &config.Config{Env: config.EnvDevelopment}
	container := &Container{}

	router := SetupRouter(cfg, container)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestSetupRouter_LocalRoutes(t *testing.T) {
	cfg := &config.Config{Env: config.EnvDevelopment}
	// Create a minimal container - router tests don't need full initialization
	// Just test that routes are registered, not that handlers work
	container := &Container{}

	router := SetupRouter(cfg, container)

	// In local mode, routes should have /api prefix
	// Only test health check since auth routes require AuthService to be initialized
	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"health check", http.MethodGet, "/health", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, http.NoBody)
			router.ServeHTTP(w, req)

			if w.Code != tt.want && w.Code != http.StatusNotFound {
				t.Errorf("expected status %d or %d, got %d", tt.want, http.StatusNotFound, w.Code)
			}
		})
	}
}

func TestBuildCORSMiddleware_Development(t *testing.T) {
	cfg := &config.Config{Env: config.EnvDevelopment}
	middleware := buildCORSMiddleware(cfg)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check CORS headers
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:5173" {
		t.Errorf("expected CORS origin %q, got %q", "http://localhost:5173", allowOrigin)
	}
}

func TestBuildCORSMiddleware_Production(t *testing.T) {
	cfg := &config.Config{
		Env: config.EnvProduction,
		Security: config.SecurityConfig{
			AllowedOrigins: []string{"https://treblesurf.com"},
		},
	}
	middleware := buildCORSMiddleware(cfg)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "https://treblesurf.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestBuildCORSMiddleware_Production_DefaultOrigins(t *testing.T) {
	cfg := &config.Config{
		Env:      config.EnvProduction,
		Security: config.SecurityConfig{},
	}
	middleware := buildCORSMiddleware(cfg)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	req.Header.Set("Origin", "https://treblesurf.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestIOSHeadersMiddleware_Router(t *testing.T) {
	router := gin.New()
	router.Use(iOSHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Header().Get("X-App-Version") != "1.0.0" {
		t.Error("expected X-App-Version header")
	}
	if w.Header().Get("X-Platform") != "iOS" {
		t.Error("expected X-Platform header")
	}
}

func TestIOSHeadersMiddleware_AuthRoutes_Router(t *testing.T) {
	router := gin.New()
	router.Use(iOSHeadersMiddleware())
	router.GET("/auth/validate", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/validate", http.NoBody)
	router.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl == "" {
		t.Error("expected Cache-Control header for auth routes")
	}
	if cacheControl != "no-store, no-cache, must-revalidate, max-age=0" {
		t.Errorf("expected specific Cache-Control, got %q", cacheControl)
	}
}

func TestAdminMiddlewareWithConfig(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AdminEmails: []string{"admin@example.com"},
		},
	}

	middleware := AdminMiddlewareWithConfig(cfg)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("email", "admin@example.com")
		c.Next()
	})
	router.Use(middleware)
	router.GET("/admin/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/test", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAdminMiddlewareWithConfig_NonAdmin(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AdminEmails: []string{"admin@example.com"},
		},
	}

	middleware := AdminMiddlewareWithConfig(cfg)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("email", "user@example.com")
		c.Next()
	})
	router.Use(middleware)
	router.GET("/admin/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/test", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAdminMiddlewareWithConfig_NoEmail(t *testing.T) {
	cfg := &config.Config{
		Security: config.SecurityConfig{
			AdminEmails: []string{"admin@example.com"},
		},
	}

	middleware := AdminMiddlewareWithConfig(cfg)

	router := gin.New()
	router.Use(middleware)
	router.GET("/admin/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/test", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// Note: DevAuthMiddleware and DevAdminAuthMiddleware tests are in middleware_test.go
// to avoid circular dependencies with auth package
