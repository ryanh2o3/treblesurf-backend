package main

import (
	"log"
	"os"
	internalapi "treblesurf-backend/internal/api"
	"treblesurf-backend/internal/auth"
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
    os.Setenv("GO_ENV", "development")
    
    // Set JWT_SECRET explicitly if not already set
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        jwtSecret = "dev-jwt-secret-for-local-development"
        os.Setenv("JWT_SECRET", jwtSecret)
        log.Println("Using default JWT_SECRET for development")
    }

    log.Println("Starting Treble Surf backend in local development mode")

	auth.InitJWTSecret()

    // Load local configuration
    cfg, err := config.Load(true)
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize local storage (DynamoDB, S3, etc.)
    if err := storage.InitLocal(cfg); err != nil {
        log.Fatalf("Failed to initialize storage: %v", err)
    }

	// Initialize the new container
	container, err := internalapi.NewContainer()
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	// Initialize API session service
	if err := auth.InitSessionService(); err != nil {
		log.Printf("Failed to initialize session service: %v", err)
	}

	r := internalapi.SetupRouter(container)
    
    // Start server
    port := cfg.Port
    if err := r.Run(":" + port); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}