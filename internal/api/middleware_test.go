package httphandler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestAdminMiddleware_NoAuth(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/test", http.NoBody)

	middleware := AdminMiddleware()
	middleware(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAdminMiddleware_NonAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/test", http.NoBody)
	c.Set("email", "user@example.com")

	middleware := AdminMiddleware()
	middleware(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestAdminMiddleware_AdminUser(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("email", "ryancpatton0@gmail.com")
		c.Next()
	})
	router.Use(AdminMiddleware())
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

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		router := gin.New()
		limiter := newRateLimiter(100)
		t.Cleanup(limiter.stop)
		router.Use(RateLimitMiddlewareWithLimiter(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make a few requests - should all pass
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected status %d, got %d", i, http.StatusOK, w.Code)
			}
		}
	})

	t.Run("rate limits after exceeding limit", func(t *testing.T) {
		router := gin.New()
		limiter := newRateLimiter(2) // Very low limit for testing
		t.Cleanup(limiter.stop)
		router.Use(RateLimitMiddlewareWithLimiter(limiter))
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// First requests should pass
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			router.ServeHTTP(w, req)
		}

		// Next request should be rate limited
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	limiter := newRateLimiter(10)
	t.Cleanup(limiter.stop)

	// Use up some tokens
	for i := 0; i < 8; i++ {
		if !limiter.allow("test-client") {
			t.Fatalf("expected request %d to be allowed", i)
		}
	}

	// Wait for tokens to refill (simulated)
	time.Sleep(200 * time.Millisecond)

	// Should be allowed again after refill
	if !limiter.allow("test-client") {
		t.Fatalf("expected request to be allowed after token refill")
	}
}

func TestRateLimiter_DifferentClients(t *testing.T) {
	limiter := newRateLimiter(5)
	t.Cleanup(limiter.stop)

	// Each client should have its own bucket
	for i := 0; i < 5; i++ {
		if !limiter.allow("client-a") {
			t.Fatalf("expected client-a request %d to be allowed", i)
		}
		if !limiter.allow("client-b") {
			t.Fatalf("expected client-b request %d to be allowed", i)
		}
	}
}

func TestIOSHeadersMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(iOSHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	router.ServeHTTP(w, req)

	if w.Header().Get("X-App-Version") != "1.0.0" {
		t.Fatalf("expected X-App-Version header")
	}
	if w.Header().Get("X-Platform") != "iOS" {
		t.Fatalf("expected X-Platform header")
	}
}

func TestIOSHeadersMiddleware_AuthRoutes_NoCaching(t *testing.T) {
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
		t.Fatalf("expected Cache-Control header for auth routes")
	}
}

func TestGetClientIPFromContext(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		expectedIP string
		clientIP   string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"},
			expectedIP: "1.2.3.4",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "9.10.11.12"},
			expectedIP: "9.10.11.12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			c.Request = req

			ip := getClientIPFromContext(c)
			if ip != tt.expectedIP {
				t.Fatalf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
