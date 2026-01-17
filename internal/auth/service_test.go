package auth

import (
	"testing"
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
		_, err := NewService("", nil, nil, nil)
		if err == nil {
			t.Fatalf("expected error for empty jwt secret")
		}
	})

	t.Run("nil user repository", func(t *testing.T) {
		_, err := NewService("test-secret", nil, nil, nil)
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
