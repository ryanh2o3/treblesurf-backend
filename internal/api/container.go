// Package httphandler provides the API container for dependency injection and route setup.
package httphandler

import (
	"fmt"
	"strings"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/controller"
	"treblesurf-backend/internal/service"
	storagepkg "treblesurf-backend/internal/storage"
)

// Container holds all application dependencies for dependency injection.
type Container struct {
	// Storage interfaces
	DynamoDBStorage storagepkg.DynamoDBStorage
	S3Storage       storagepkg.S3Storage

	// Services
	ForecastService        *service.ForecastService
	TideService            *service.TideService
	LocationService        *service.LocationService
	ReportService          *service.ReportService
	APIKeyService          *service.APIKeyService
	WebSocketService       *service.WebSocketService
	SwellPredictionService *service.SwellPredictionService
	AuthService            *auth.Service
	StreamService          *service.StreamService
	SnapshotService        *service.SnapshotService
	BuoyService            *service.BuoyService

	// Controllers
	ForecastController        *controller.ForecastController
	SwellPredictionController *controller.SwellPredictionController
	UserController            *controller.UserController
	ReportController          *controller.ReportController
	LocationController        *controller.LocationController
	APIKeyController          *controller.APIKeyController
	BuoyController            *controller.BuoyController
	StreamController          *controller.StreamController
	SnapshotController        *controller.SnapshotController

	rateLimiter *rateLimiter
}

// containerConfig holds configuration values needed for container initialization.
type containerConfig struct {
	region            string
	bucketName        string
	jwtSecret         string
	websocketEndpoint string
	websocketStage    string
	isLocal           bool
}

// NewContainer creates a new Container with all dependencies wired up.
func NewContainer(cfg *config.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	containerCfg := loadContainerConfig(cfg)

	storage, err := initializeStorage(containerCfg)
	if err != nil {
		return nil, fmt.Errorf("initializing storage: %w", err)
	}

	services, err := initializeServices(storage, containerCfg, cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing services: %w", err)
	}

	controllers := initializeControllers(services, storage, containerCfg)

	var rateLimiter *rateLimiter
	if strings.EqualFold(cfg.Security.RateLimitMode, "in-memory") {
		rateLimiter = newRateLimiter(cfg.Security.RateLimitRPS)
	}
	return buildContainer(storage, services, controllers, rateLimiter), nil
}

// loadContainerConfig extracts container configuration from the main config.
func loadContainerConfig(cfg *config.Config) *containerConfig {
	return &containerConfig{
		region:            cfg.AWS.Region,
		bucketName:        cfg.AWS.BucketName,
		isLocal:           cfg.IsDevelopment(),
		jwtSecret:         cfg.Auth.JWTSecret,
		websocketEndpoint: cfg.WebSocket.Endpoint,
		websocketStage:    cfg.WebSocket.Stage,
	}
}

// buildContainer assembles the final Container from initialized components.
func buildContainer(
	storage *containerStorage,
	services *containerServices,
	controllers *containerControllers,
	rateLimiter *rateLimiter,
) *Container {
	return &Container{
		DynamoDBStorage:           storage.dbStorage,
		S3Storage:                 storage.s3Storage,
		ForecastService:           services.forecastService,
		TideService:               services.tideService,
		LocationService:           services.locationService,
		ReportService:             services.reportService,
		APIKeyService:             services.apiKeyService,
		WebSocketService:          services.websocketService,
		SwellPredictionService:    services.swellPredictionService,
		AuthService:               services.authService,
		StreamService:             services.streamService,
		SnapshotService:           services.snapshotService,
		BuoyService:               services.buoyService,
		ForecastController:        controllers.forecastController,
		SwellPredictionController: controllers.swellPredictionController,
		UserController:            controllers.userController,
		ReportController:          controllers.reportController,
		LocationController:        controllers.locationController,
		APIKeyController:          controllers.apiKeyController,
		BuoyController:            controllers.buoyController,
		StreamController:          controllers.streamController,
		SnapshotController:        controllers.snapshotController,
		rateLimiter:               rateLimiter,
	}
}

func (c *Container) Close() {
	if c == nil || c.rateLimiter == nil {
		return
	}
	c.rateLimiter.stop()
}
