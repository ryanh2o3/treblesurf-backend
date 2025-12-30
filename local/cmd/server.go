// Package main provides the local development server for the Treble Surf backend.
package main

import (
	"log"
	"os"
	httphandler "treblesurf-backend/internal/api"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/local/config"
	"treblesurf-backend/local/storage"

	"github.com/joho/godotenv"
)

func main() {
	// Set environment variable to indicate local development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, using default values")
	}

	// Set environment variable to indicate local development
	if err := os.Setenv("GO_ENV", constants.EnvDevelopment); err != nil {
		log.Printf("Warning: Failed to set GO_ENV: %v", err)
	}

	// Set JWT_SECRET explicitly if not already set
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Only use default in local development
		jwtSecret = "dev-jwt-secret-for-local-development" //nolint:gosec // Local development only
		if err := os.Setenv("JWT_SECRET", jwtSecret); err != nil {
			log.Printf("Warning: Failed to set JWT_SECRET environment variable: %v", err)
		}
		log.Println("WARNING: Using default JWT_SECRET for local development only")
	}

	log.Println("Starting Treble Surf backend in local development mode")

	auth.InitJWTSecret()

	// Load local configuration
	cfg, err := config.Load(true)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize local storage (DynamoDB, S3, etc.)
	if initErr := storage.InitLocal(cfg); initErr != nil {
		log.Fatalf("Failed to initialize storage: %v", initErr)
	}

	// Initialize the new container
	container, err := httphandler.NewContainer()
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	// Initialize API session service
	if err := auth.InitSessionService(); err != nil {
		log.Printf("Failed to initialize session service: %v", err)
	}

	r := httphandler.SetupRouter(container)

	// Start server
	port := cfg.Port
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
