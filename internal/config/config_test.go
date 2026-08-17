package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment for testing defaults
	if err := os.Unsetenv("GO_ENV"); err != nil {
		t.Fatalf("failed to unset GO_ENV: %v", err)
	}
	if err := os.Unsetenv("AWS_REGION"); err != nil {
		t.Fatalf("failed to unset AWS_REGION: %v", err)
	}
	if err := os.Unsetenv("S3_BUCKET_NAME"); err != nil {
		t.Fatalf("failed to unset S3_BUCKET_NAME: %v", err)
	}
	if err := os.Unsetenv("JWT_SECRET"); err != nil {
		t.Fatalf("failed to unset JWT_SECRET: %v", err)
	}
	if err := os.Unsetenv("PORT"); err != nil {
		t.Fatalf("failed to unset PORT: %v", err)
	}

	// Should fail in production without JWT_SECRET
	cfg, err := Load()
	if err == nil && cfg.Env == EnvProduction {
		t.Fatalf("expected error for missing JWT_SECRET in production")
	}

	// Set environment to development
	if setErr := os.Setenv("GO_ENV", "development"); setErr != nil {
		t.Fatalf("failed to set GO_ENV: %v", setErr)
	}
	defer func() {
		if unsetErr := os.Unsetenv("GO_ENV"); unsetErr != nil {
			t.Fatalf("failed to unset GO_ENV: %v", unsetErr)
		}
	}()

	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AWS.Region != "eu-west-1" {
		t.Fatalf("expected default region eu-west-1, got %s", cfg.AWS.Region)
	}
	if cfg.AWS.BucketName != "treblesurf-images" {
		t.Fatalf("expected default bucket name, got %s", cfg.AWS.BucketName)
	}
	if cfg.Server.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Server.Port)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	if err := os.Setenv("GO_ENV", "development"); err != nil {
		t.Fatalf("failed to set GO_ENV: %v", err)
	}
	if err := os.Setenv("AWS_REGION", "us-east-1"); err != nil {
		t.Fatalf("failed to set AWS_REGION: %v", err)
	}
	if err := os.Setenv("S3_BUCKET_NAME", "custom-bucket"); err != nil {
		t.Fatalf("failed to set S3_BUCKET_NAME: %v", err)
	}
	if err := os.Setenv("PORT", "3000"); err != nil {
		t.Fatalf("failed to set PORT: %v", err)
	}
	if err := os.Setenv("ADMIN_EMAILS", "admin1@example.com, admin2@example.com"); err != nil {
		t.Fatalf("failed to set ADMIN_EMAILS: %v", err)
	}
	if err := os.Setenv("ALLOWED_ORIGINS", "https://example.com, https://app.example.com"); err != nil {
		t.Fatalf("failed to set ALLOWED_ORIGINS: %v", err)
	}

	defer func() {
		if err := os.Unsetenv("GO_ENV"); err != nil {
			t.Fatalf("failed to unset GO_ENV: %v", err)
		}
		if err := os.Unsetenv("AWS_REGION"); err != nil {
			t.Fatalf("failed to unset AWS_REGION: %v", err)
		}
		if err := os.Unsetenv("S3_BUCKET_NAME"); err != nil {
			t.Fatalf("failed to unset S3_BUCKET_NAME: %v", err)
		}
		if err := os.Unsetenv("PORT"); err != nil {
			t.Fatalf("failed to unset PORT: %v", err)
		}
		if err := os.Unsetenv("ADMIN_EMAILS"); err != nil {
			t.Fatalf("failed to unset ADMIN_EMAILS: %v", err)
		}
		if err := os.Unsetenv("ALLOWED_ORIGINS"); err != nil {
			t.Fatalf("failed to unset ALLOWED_ORIGINS: %v", err)
		}
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AWS.Region != "us-east-1" {
		t.Fatalf("expected region us-east-1, got %s", cfg.AWS.Region)
	}
	if cfg.AWS.BucketName != "custom-bucket" {
		t.Fatalf("expected bucket name custom-bucket, got %s", cfg.AWS.BucketName)
	}
	if cfg.Server.Port != "3000" {
		t.Fatalf("expected port 3000, got %s", cfg.Server.Port)
	}
	if len(cfg.Security.AdminEmails) != 2 {
		t.Fatalf("expected 2 admin emails, got %d", len(cfg.Security.AdminEmails))
	}
	if len(cfg.Security.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 allowed origins, got %d", len(cfg.Security.AllowedOrigins))
	}
}

func TestConfig_IsAdmin(t *testing.T) {
	cfg := &Config{
		Security: SecurityConfig{
			AdminEmails: []string{"admin@example.com", "superadmin@example.com"},
		},
	}

	if !cfg.IsAdmin("admin@example.com") {
		t.Fatalf("expected admin@example.com to be admin")
	}
	if !cfg.IsAdmin("ADMIN@EXAMPLE.COM") { // Case insensitive
		t.Fatalf("expected case-insensitive admin check")
	}
	if cfg.IsAdmin("user@example.com") {
		t.Fatalf("expected user@example.com to not be admin")
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{Env: EnvDevelopment}
	if !cfg.IsDevelopment() {
		t.Fatalf("expected IsDevelopment to return true")
	}

	cfg = &Config{Env: EnvProduction}
	if cfg.IsDevelopment() {
		t.Fatalf("expected IsDevelopment to return false")
	}
}

func TestMustLoad_Panics(t *testing.T) {
	// Set production environment without JWT_SECRET
	if err := os.Setenv("GO_ENV", "production"); err != nil {
		t.Fatalf("failed to set GO_ENV: %v", err)
	}
	if err := os.Unsetenv("JWT_SECRET"); err != nil {
		t.Fatalf("failed to unset JWT_SECRET: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("GO_ENV"); err != nil {
			t.Fatalf("failed to unset GO_ENV: %v", err)
		}
	}()

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected MustLoad to panic")
		}
	}()

	MustLoad()
}

func TestSplitCommaSeparated(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{"  a  ,  b  ,  c  ", []string{"a", "b", "c"}},
		{"", []string{}},
		{"a", []string{"a"}},
		{",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitCommaSeparated(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d items, got %d", len(tt.expected), len(result))
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Fatalf("expected %s at index %d, got %s", tt.expected[i], i, v)
				}
			}
		})
	}
}
