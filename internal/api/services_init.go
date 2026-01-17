// Package httphandler provides service initialization for the API container.
package httphandler

import (
	"fmt"
	"log/slog"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/controller"
	repodynamo "treblesurf-backend/internal/repository/dynamodb"
	repos3 "treblesurf-backend/internal/repository/s3"
	"treblesurf-backend/internal/service"
)

// containerServices holds all initialized services.
type containerServices struct {
	forecastService        *service.ForecastService
	tideService            *service.TideService
	locationService        *service.LocationService
	userService            *service.UserService
	reportService          *service.ReportService
	apiKeyService          *service.APIKeyService
	swellPredictionService *service.SwellPredictionService
	websocketService       *service.WebSocketService
	authService            *auth.Service
	streamService          *service.StreamService
	snapshotService        *service.SnapshotService
	buoyService            *service.BuoyService
}

// containerControllers holds all initialized controllers.
type containerControllers struct {
	forecastController        *controller.ForecastController
	swellPredictionController *controller.SwellPredictionController
	userController            *controller.UserController
	reportController          *controller.ReportController
	locationController        *controller.LocationController
	apiKeyController          *controller.APIKeyController
	buoyController            *controller.BuoyController
	streamController          *controller.StreamController
	snapshotController        *controller.SnapshotController
}

// initializeServices creates all application services with their dependencies.
func initializeServices(storage *containerStorage, cfg containerConfig) (*containerServices, error) {
	// Initialize repositories
	userRepo := repodynamo.NewUserRepo(storage.dynamoDBClient, "Users")
	sessionRepo := repodynamo.NewSessionRepo(storage.dynamoDBClient, "Sessions")
	apiKeyRepo := repodynamo.NewAPIKeyRepo(storage.dynamoDBClient, "ApiKeys")
	locationRepo := repodynamo.NewLocationRepo(storage.dynamoDBClient, "LocationData")
	forecastRepo := repodynamo.NewForecastRepo(storage.dynamoDBClient, "SpotForecastData")
	websocketRepo := repodynamo.NewWebSocketRepo(storage.dynamoDBClient, "WebSocketConnections")
	subscriptionRepo := repodynamo.NewSpotSubscriptionRepo(storage.dynamoDBClient, "SpotSubscriptions")
	swellPredictionRepo := repodynamo.NewSwellPredictionRepo(storage.dynamoDBClient, "SwellPredictions")
	reportRepo := repodynamo.NewReportRepo(storage.dynamoDBClient, "SurfReports")
	buoyRepo := repodynamo.NewBuoyRepo(storage.dynamoDBClient, "BuoyData", "BuoyLocations")
	mediaRepo := repos3.NewMediaRepo(storage.s3Client, cfg.bucketName)
	streamRequestRepo := repodynamo.NewStreamRequestRepo(storage.dynamoDBClient, "StreamRequests")
	snapshotRepo := repodynamo.NewSnapshotRepo(storage.dynamoDBClient, "SpotSnapshots")

	// Initialize auth service first (other services may depend on it)
	authService, err := auth.NewService(cfg.jwtSecret, userRepo, sessionRepo, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("creating auth service: %w", err)
	}

	// Initialize user service
	userService, err := service.NewUserService(userRepo)
	if err != nil {
		return nil, fmt.Errorf("creating user service: %w", err)
	}

	// Initialize other services
	forecastService, err := service.NewForecastService(forecastRepo)
	if err != nil {
		return nil, fmt.Errorf("creating forecast service: %w", err)
	}

	locationService, err := service.NewLocationService(locationRepo, mediaRepo)
	if err != nil {
		return nil, fmt.Errorf("creating location service: %w", err)
	}

	apiKeyService, err := service.NewAPIKeyService(apiKeyRepo)
	if err != nil {
		return nil, fmt.Errorf("creating apikey service: %w", err)
	}

	swellPredictionService, err := service.NewSwellPredictionService(swellPredictionRepo)
	if err != nil {
		return nil, fmt.Errorf("creating swell prediction service: %w", err)
	}

	streamService, err := service.NewStreamService(streamRequestRepo)
	if err != nil {
		return nil, fmt.Errorf("creating stream service: %w", err)
	}

	snapshotService, err := service.NewSnapshotService(snapshotRepo)
	if err != nil {
		return nil, fmt.Errorf("creating snapshot service: %w", err)
	}

	buoyService, err := service.NewBuoyService(buoyRepo)
	if err != nil {
		return nil, fmt.Errorf("creating buoy service: %w", err)
	}

	services := &containerServices{
		forecastService:        forecastService,
		tideService:            service.NewTideService(),
		locationService:        locationService,
		userService:            userService,
		apiKeyService:          apiKeyService,
		swellPredictionService: swellPredictionService,
		authService:            authService,
		streamService:          streamService,
		snapshotService:        snapshotService,
		buoyService:            buoyService,
	}

	// Initialize report service (depends on other services)
	services.reportService = service.NewReportService(
		mediaRepo,
		reportRepo,
		buoyRepo,
		locationRepo,
		forecastRepo,
		storage.rekognitionClient,
		services.userService,
	)

	// Initialize websocket service
	services.websocketService = service.NewWebSocketService(websocketRepo, subscriptionRepo, []byte(cfg.jwtSecret))

	return services, nil
}

// initializeControllers creates all controllers with their service dependencies.
func initializeControllers(
	services *containerServices,
	storage *containerStorage,
	cfg containerConfig,
) *containerControllers {
	return &containerControllers{
		forecastController:        controller.NewForecastController(services.forecastService, services.tideService),
		swellPredictionController: controller.NewSwellPredictionController(services.swellPredictionService),
		userController:            controller.NewUserController(services.userService),
		reportController:          controller.NewReportController(services.reportService, services.userService),
		locationController:        controller.NewLocationController(services.locationService),
		apiKeyController:          controller.NewAPIKeyController(services.apiKeyService),
		buoyController:            controller.NewBuoyController(services.buoyService),
		streamController:          controller.NewStreamController(services.streamService),
		snapshotController:        controller.NewSnapshotController(services.snapshotService, storage.s3Client, cfg.bucketName),
	}
}
