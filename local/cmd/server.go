// Package main provides the local development server for the Treble Surf backend.
package main

import (
	"log"
	"os"
	httphandler "treblesurf-backend/internal/api"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/logging"
	localconfig "treblesurf-backend/local/config"
	"treblesurf-backend/local/storage"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, using default values")
	}

	if err := os.Setenv("GO_ENV", constants.EnvDevelopment); err != nil {
		log.Printf("Warning: Failed to set GO_ENV: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-jwt-secret-for-local-development" //nolint:gosec // Local development only
		if err := os.Setenv("JWT_SECRET", jwtSecret); err != nil {
			log.Printf("Warning: Failed to set JWT_SECRET environment variable: %v", err)
		}
		log.Println("WARNING: Using default JWT_SECRET for local development only")
	}

	log.Println("Starting Treble Surf backend in local development mode")

	appCfg := config.MustLoad()
	logging.Init(appCfg)
	if err := auth.InitJWTSecret(appCfg.Auth.JWTSecret); err != nil {
		log.Fatalf("Failed to initialize JWT secret: %v", err)
	}

	cfg, err := localconfig.Load(true)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if initErr := storage.InitLocal(cfg); initErr != nil {
		log.Fatalf("Failed to initialize storage: %v", initErr)
	}

	container, err := httphandler.NewContainer(appCfg)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}

	if err := auth.InitSessionService(); err != nil {
		log.Printf("Failed to initialize session service: %v", err)
	}

	r := httphandler.SetupRouter(appCfg, container)

	port := cfg.Port
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
