// Package config provides configuration management for local development.
package config

import (
	"os"
)

// Config holds application configuration
type Config struct {
    // AWS related configs
    AWSRegion string
    
    // DynamoDB settings
    DynamoDBEndpoint string
    
    // S3 settings
    S3Endpoint string
    
    // App settings
    Port        string
    JWTSecret   string
    Environment string
    
    // Development flags
    SkipAuth bool
}

// Load configuration from environment variables or use development defaults
func Load(_ bool) (*Config, error) {
    cfg := &Config{
        Environment: "development",
        
        // Set AWS region
        AWSRegion: getEnv("AWS_REGION", "eu-west-1"),
        
        // Set endpoints for local services
        DynamoDBEndpoint: getEnv("DYNAMODB_ENDPOINT", "http://localhost:8000"),
        S3Endpoint: getEnv("S3_ENDPOINT", "http://localhost:4566"),
        
        // JWT secret for local development
        JWTSecret: getEnv("JWT_SECRET", "dev-jwt-secret-for-local-development"),
        
        // Port settings
        Port: getEnv("PORT", "8080"),
        
        // Skip authentication for easier local development
        SkipAuth: getEnv("SKIP_AUTH", "false") == "true",
    }
    
    return cfg, nil
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}