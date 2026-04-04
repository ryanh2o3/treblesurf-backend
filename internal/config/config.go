// Package config provides application configuration management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	AWS       AWSConfig
	WebSocket WebSocketConfig
	Env       Environment
	Security  SecurityConfig
	Server    ServerConfig
	Auth      AuthConfig
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
	Region            string
	BucketName        string
	SpotImagesBaseURL string // SPOT_IMAGES_BASE_URL: CDN or public origin for spot-images/* (no trailing slash)
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
	Port           string
	RequestTimeout time.Duration
}

// SecurityConfig holds security-related configuration.
type SecurityConfig struct {
	RateLimitMode  string
	DevUserEmail   string
	DevAdminEmail  string
	AdminEmails    []string
	AllowedOrigins []string
	RateLimitRPS   int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	env := Environment(getEnvOrDefault("GO_ENV", string(EnvProduction)))
	cfg := newBaseConfig(env)

	if err := loadAuthConfig(cfg, env); err != nil {
		return nil, err
	}
	loadSecurityConfig(cfg, env)
	loadServerConfig(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
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

// ResolvedSpotImagesBaseURL returns the base URL for spot JPEGs (spot-images/...).
// If SPOT_IMAGES_BASE_URL is unset, uses virtual-hosted S3 URL for the configured bucket and region.
func (c *Config) ResolvedSpotImagesBaseURL() string {
	if c == nil {
		return ""
	}
	base := strings.TrimSpace(c.AWS.SpotImagesBaseURL)
	base = strings.TrimSuffix(base, "/")
	if base != "" {
		return base
	}
	bucket := strings.TrimSpace(c.AWS.BucketName)
	region := strings.TrimSpace(c.AWS.Region)
	if bucket == "" || region == "" {
		return ""
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
}

func (c *Config) Validate() error {
	var missing []string

	if strings.TrimSpace(c.AWS.Region) == "" {
		missing = append(missing, "AWS_REGION")
	}
	if strings.TrimSpace(c.AWS.BucketName) == "" {
		missing = append(missing, "S3_BUCKET_NAME")
	}
	if strings.TrimSpace(c.WebSocket.Endpoint) == "" {
		missing = append(missing, "WEBSOCKET_API_ENDPOINT")
	}
	if len(c.Security.AdminEmails) == 0 {
		missing = append(missing, "ADMIN_EMAILS")
	}

	if c.Env == EnvProduction && len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string) (value, ok bool) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return false, false
	}
	switch strings.ToLower(rawValue) {
	case "1", "true", "t", "yes", "y":
		return true, true
	case "0", "false", "f", "no", "n":
		return false, true
	default:
		return false, false
	}
}

func getEnvInt(key string) (int, bool) {
	rawValue := strings.TrimSpace(os.Getenv(key))
	if rawValue == "" {
		return 0, false
	}
	value, err := strconv.Atoi(rawValue)
	if err != nil {
		return 0, false
	}
	return value, true
}

func newBaseConfig(env Environment) *Config {
	rateLimitMode := "in-memory"
	if env == EnvProduction {
		rateLimitMode = "api-gateway"
	}

	return &Config{
		Env: env,
		AWS: AWSConfig{
			Region:            getEnvOrDefault("AWS_REGION", "eu-west-1"),
			BucketName:        getEnvOrDefault("S3_BUCKET_NAME", "treblesurf-images"),
			SpotImagesBaseURL: strings.TrimSpace(os.Getenv("SPOT_IMAGES_BASE_URL")),
		},
		WebSocket: WebSocketConfig{
			Endpoint: os.Getenv("WEBSOCKET_API_ENDPOINT"),
			Stage:    getEnvOrDefault("WEBSOCKET_API_STAGE", "production"),
		},
		Server: ServerConfig{
			Port:           getEnvOrDefault("PORT", "8080"),
			RequestTimeout: 30 * time.Second,
		},
		Security: SecurityConfig{
			RateLimitRPS:  100, // Default rate limit
			RateLimitMode: rateLimitMode,
		},
	}
}

func loadAuthConfig(cfg *Config, env Environment) error {
	cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.Auth.JWTSecret == "" && env == EnvProduction {
		return fmt.Errorf("JWT_SECRET is required in production")
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "dev-secret-do-not-use-in-prod"
	}

	cfg.Auth.GoogleClientIDs = loadGoogleClientIDs()

	cfg.Auth.CookieSecure = !cfg.IsDevelopment()
	if secure, ok := getEnvBool("COOKIE_SECURE"); ok {
		cfg.Auth.CookieSecure = secure
	}

	return nil
}

func loadGoogleClientIDs() []string {
	if ids := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_IDS")); ids != "" {
		return splitCommaSeparated(ids)
	}

	legacyIDs := []string{
		strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
		strings.TrimSpace(os.Getenv("GOOGLE_IOS_CLIENT_ID")),
	}
	out := make([]string, 0, len(legacyIDs))
	for _, id := range legacyIDs {
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func loadSecurityConfig(cfg *Config, env Environment) {
	if admins := strings.TrimSpace(os.Getenv("ADMIN_EMAILS")); admins != "" {
		cfg.Security.AdminEmails = splitCommaSeparated(admins)
	}

	if origins := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")); origins != "" {
		cfg.Security.AllowedOrigins = splitCommaSeparated(origins)
	} else if env == EnvProduction {
		cfg.Security.AllowedOrigins = []string{
			"https://treblesurf.com",
			"https://www.treblesurf.com",
			"https://app.treblesurf.com",
		}
	}

	if rateLimitRPS, ok := getEnvInt("RATE_LIMIT_RPS"); ok && rateLimitRPS > 0 {
		cfg.Security.RateLimitRPS = rateLimitRPS
	}

	if mode := strings.TrimSpace(os.Getenv("RATE_LIMIT_MODE")); mode != "" {
		cfg.Security.RateLimitMode = mode
	}

	if cfg.Security.DevUserEmail == "" {
		cfg.Security.DevUserEmail = strings.TrimSpace(os.Getenv("DEV_USER_EMAIL"))
	}
	if cfg.Security.DevAdminEmail == "" {
		cfg.Security.DevAdminEmail = strings.TrimSpace(os.Getenv("DEV_ADMIN_EMAIL"))
	}
	if cfg.Security.DevUserEmail == "" && env == EnvDevelopment {
		cfg.Security.DevUserEmail = "testuser@example.com"
	}
	if cfg.Security.DevAdminEmail == "" && env == EnvDevelopment {
		cfg.Security.DevAdminEmail = "admin@example.com"
	}
}

func loadServerConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if timeoutSeconds, ok := getEnvInt("REQUEST_TIMEOUT_SECONDS"); ok && timeoutSeconds > 0 {
		cfg.Server.RequestTimeout = time.Duration(timeoutSeconds) * time.Second
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
