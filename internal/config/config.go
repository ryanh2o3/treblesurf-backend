// Package config provides application configuration management.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	AWS       AWSConfig
	WebSocket WebSocketConfig
	Server    ServerConfig
	Env       Environment
	Auth      AuthConfig
	Security  SecurityConfig
}

// Environment represents the application environment.
type Environment string

const (
	// EnvDevelopment is the development environment.
	EnvDevelopment Environment = "development"
	// EnvProduction is the production environment.
	EnvProduction Environment = "production"
)

// AWSConfig holds AWS-related configuration.
type AWSConfig struct {
	Region     string
	BucketName string
}

// AuthConfig holds authentication-related configuration.
type AuthConfig struct {
	JWTSecret       string
	GoogleClientIDs []string
	CookieSecure    bool
}

// WebSocketConfig holds WebSocket-related configuration.
type WebSocketConfig struct {
	Endpoint string
	Stage    string
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port string
}

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	AdminEmails    []string
	AllowedOrigins []string
	RateLimitRPS   int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	env := Environment(getEnvOrDefault("GO_ENV", string(EnvProduction)))

	cfg := &Config{
		Env: env,
		AWS: AWSConfig{
			Region:     getEnvOrDefault("AWS_REGION", "eu-west-1"),
			BucketName: getEnvOrDefault("S3_BUCKET_NAME", "treblesurf-images"),
		},
		WebSocket: WebSocketConfig{
			Endpoint: os.Getenv("WEBSOCKET_API_ENDPOINT"),
			Stage:    getEnvOrDefault("WEBSOCKET_API_STAGE", "production"),
		},
		Server: ServerConfig{
			Port: getEnvOrDefault("PORT", "8080"),
		},
		Security: SecurityConfig{
			RateLimitRPS: 100, // Default rate limit
		},
	}

	// Load JWT secret
	cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.Auth.JWTSecret == "" && env == EnvProduction {
		return nil, fmt.Errorf("JWT_SECRET is required in production")
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "dev-secret-do-not-use-in-prod"
	}

	// Load Google client IDs
	if ids := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_IDS")); ids != "" {
		cfg.Auth.GoogleClientIDs = splitCommaSeparated(ids)
	} else {
		legacyIDs := []string{
			strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
			strings.TrimSpace(os.Getenv("GOOGLE_IOS_CLIENT_ID")),
		}
		for _, id := range legacyIDs {
			if id != "" {
				cfg.Auth.GoogleClientIDs = append(cfg.Auth.GoogleClientIDs, id)
			}
		}
	}

	// Load cookie security settings (defaults to secure in production)
	cfg.Auth.CookieSecure = !cfg.IsDevelopment()
	if secure, ok := getEnvBool("COOKIE_SECURE"); ok {
		cfg.Auth.CookieSecure = secure
	}

	// Load admin emails from environment
	if admins := strings.TrimSpace(os.Getenv("ADMIN_EMAILS")); admins != "" {
		cfg.Security.AdminEmails = splitCommaSeparated(admins)
	}

	// Load allowed origins for CORS
	if origins := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); origins != "" {
		cfg.Security.AllowedOrigins = splitCommaSeparated(origins)
	} else if env == EnvProduction {
		// Default production origins
		cfg.Security.AllowedOrigins = []string{
			"https://treblesurf.com",
			"https://www.treblesurf.com",
			"https://app.treblesurf.com",
		}
	}

	return cfg, nil
}

// IsAdmin checks if the given email is an admin.
func (c *Config) IsAdmin(email string) bool {
	for _, admin := range c.Security.AdminEmails {
		if strings.EqualFold(admin, email) {
			return true
		}
	}
	return false
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func (c *Config) IsDevelopment() bool {
	return c.Env == EnvDevelopment
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) (bool, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false, false
	}
	switch strings.ToLower(value) {
	case "1", "true", "t", "yes", "y":
		return true, true
	case "0", "false", "f", "no", "n":
		return false, true
	default:
		return false, false
	}
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
