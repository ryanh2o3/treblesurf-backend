package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/config"

	"github.com/gin-gonic/gin"
)

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token1 == "" {
		t.Fatalf("expected non-empty token")
	}

	// Token should be base64 encoded (32 bytes = 44 chars in base64 with padding)
	if len(token1) < 40 {
		t.Fatalf("token too short: %s", token1)
	}

	// Generate another token - should be different
	token2, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token1 == token2 {
		t.Fatalf("tokens should be unique")
	}
}

func TestNewService_NilDependencies(t *testing.T) {
	t.Run("empty jwt secret", func(t *testing.T) {
		_, err := NewService(&config.Config{}, nil, nil)
		if err == nil {
			t.Fatalf("expected error for empty jwt secret")
		}
	})

	t.Run("nil user repository", func(t *testing.T) {
		cfg := &config.Config{
			Env:  config.EnvDevelopment,
			Auth: config.AuthConfig{JWTSecret: "test-secret"},
		}
		_, err := NewService(cfg, nil, nil)
		if err == nil {
			t.Fatalf("expected error for nil user repository")
		}
	})
}

func TestToAuthUser_NilInput(t *testing.T) {
	result := toAuthUser(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestDevLoginHandler_DisabledOutsideDevelopment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &Service{isDevelopment: false}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body, _ := json.Marshal(map[string]string{"email": "dev@example.com"})
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/dev-session", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	svc.DevLoginHandler(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
