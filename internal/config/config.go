package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AWS       AWSConfig
	Auth      AuthConfig
	WebSocket WebSocketConfig
	Server    ServerConfig
	Env       Environment
}

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvProduction  Environment = "production"
)

type AWSConfig struct {
	Region     string
	BucketName string
}

type AuthConfig struct {
	JWTSecret       string
	GoogleClientIDs []string
}

type WebSocketConfig struct {
	Endpoint string
	Stage    string
}

type ServerConfig struct {
	Port string
}

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
	}

	cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	if cfg.Auth.JWTSecret == "" && env == EnvProduction {
		return nil, fmt.Errorf("JWT_SECRET is required in production")
	}
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "dev-secret-do-not-use-in-prod"
	}

	if ids := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_IDS")); ids != "" {
		cfg.Auth.GoogleClientIDs = splitCommaSeparated(ids)
	}

	return cfg, nil
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
