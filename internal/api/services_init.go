// Package httphandler provides service initialization for the API container.
package httphandler

import (
	"fmt"

	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/controller"
	"treblesurf-backend/internal/repository"
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
	contentReportService   *service.ContentReportService
	moderationService      *service.ModerationService
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
	contentReportController   *controller.ContentReportController
	moderationController      *controller.ModerationController
}

type containerRepositories struct {
	userRepo             repository.UserRepository
	sessionRepo          repository.SessionRepository
	apiKeyRepo           repository.APIKeyRepository
	locationRepo         repository.LocationRepository
	forecastRepo         repository.ForecastRepository
	forecastDataRepo     repository.ForecastDataRepository
	websocketRepo        repository.WebSocketRepository
	subscriptionRepo     repository.SpotSubscriptionRepository
	swellPredictionRepo  repository.SwellPredictionRepository
	reportRepo           repository.ReportRepository
	buoyRepo             repository.BuoyRepository
	mediaRepo            repository.MediaRepository
	streamRequestRepo    repository.StreamRequestRepository
	snapshotRepo         repository.SnapshotRepository
	contentReportRepo    repository.ContentReportRepository
	moderationActionRepo repository.ModerationActionRepository
}

// initializeServices creates all application services with their dependencies.
func initializeServices(storage *containerStorage, cfg *containerConfig, appCfg *config.Config) (*containerServices, error) {
	repos := initRepositories(storage, cfg)
	return buildServices(repos, storage, cfg, appCfg)
}

func initRepositories(storage *containerStorage, cfg *containerConfig) *containerRepositories {
	forecastRepo := repodynamo.NewForecastRepo(storage.dynamoDBClient, cfg.forecastTable)
	return &containerRepositories{
		userRepo:             repodynamo.NewUserRepo(storage.dynamoDBClient, "Users"),
		sessionRepo:          repodynamo.NewSessionRepo(storage.dynamoDBClient, "Sessions"),
		apiKeyRepo:           repodynamo.NewAPIKeyRepo(storage.dynamoDBClient, "ApiKeys"),
		locationRepo:         repodynamo.NewLocationRepo(storage.dynamoDBClient, "LocationData"),
		forecastRepo:         forecastRepo,
		forecastDataRepo:     forecastRepo,
		websocketRepo:        repodynamo.NewWebSocketRepo(storage.dynamoDBClient, "WebSocketConnections"),
		subscriptionRepo:     repodynamo.NewSpotSubscriptionRepo(storage.dynamoDBClient, "SpotSubscriptions"),
		swellPredictionRepo:  repodynamo.NewSwellPredictionRepo(storage.dynamoDBClient, "SwellPredictions"),
		reportRepo:           repodynamo.NewReportRepo(storage.dynamoDBClient, "SurfReports"),
		buoyRepo:             repodynamo.NewBuoyRepo(storage.dynamoDBClient, "BuoyData", "BuoyLocations"),
		mediaRepo:            repos3.NewMediaRepo(storage.s3Client, cfg.bucketName),
		streamRequestRepo:    repodynamo.NewStreamRequestRepo(storage.dynamoDBClient, "StreamRequests"),
		snapshotRepo:         repodynamo.NewSnapshotRepo(storage.dynamoDBClient, "SpotSnapshots"),
		contentReportRepo:    repodynamo.NewContentReportRepo(storage.dynamoDBClient, "ContentReports"),
		moderationActionRepo: repodynamo.NewModerationActionRepo(storage.dynamoDBClient, "ModerationActions"),
	}
}

func buildServices(
	repos *containerRepositories,
	storage *containerStorage,
	cfg *containerConfig,
	appCfg *config.Config,
) (*containerServices, error) {
	services := &containerServices{}

	tideService, err := service.NewTideService(appCfg)
	if err != nil {
		return nil, fmt.Errorf("creating tide service: %w", err)
	}
	services.tideService = tideService

	if err := initAuthAndUserServices(appCfg, repos, services); err != nil {
		return nil, err
	}
	if err := initDomainServices(repos, services, appCfg); err != nil {
		return nil, err
	}

	websocketService, err := service.NewWebSocketService(
		repos.websocketRepo,
		repos.subscriptionRepo,
		[]byte(cfg.jwtSecret),
		cfg.websocketEndpoint,
		cfg.websocketStage,
	)
	if err != nil {
		return nil, fmt.Errorf("creating websocket service: %w", err)
	}
	services.websocketService = websocketService

	reportService, err := service.NewReportService(
		repos.mediaRepo,
		repos.reportRepo,
		repos.buoyRepo,
		repos.locationRepo,
		repos.forecastDataRepo,
		storage.rekognitionClient,
		services.userService,
		services.websocketService,
	)
	if err != nil {
		return nil, fmt.Errorf("creating report service: %w", err)
	}
	services.reportService = reportService

	contentReportService, err := service.NewContentReportService(
		repos.contentReportRepo,
		repos.reportRepo,
		repos.userRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("creating content report service: %w", err)
	}
	services.contentReportService = contentReportService

	moderationService, err := service.NewModerationService(
		repos.contentReportRepo,
		repos.moderationActionRepo,
		repos.userRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("creating moderation service: %w", err)
	}
	services.moderationService = moderationService

	return services, nil
}

func initAuthAndUserServices(
	appCfg *config.Config,
	repos *containerRepositories,
	services *containerServices,
) error {
	authService, err := auth.NewService(appCfg, repos.userRepo, repos.sessionRepo)
	if err != nil {
		return fmt.Errorf("creating auth service: %w", err)
	}
	userService, err := service.NewUserService(repos.userRepo)
	if err != nil {
		return fmt.Errorf("creating user service: %w", err)
	}
	services.authService = authService
	services.userService = userService
	return nil
}

func initDomainServices(repos *containerRepositories, services *containerServices, appCfg *config.Config) error {
	forecastService, err := service.NewForecastService(repos.forecastRepo, repos.locationRepo)
	if err != nil {
		return fmt.Errorf("creating forecast service: %w", err)
	}
	locationService, err := service.NewLocationService(repos.locationRepo, appCfg.ResolvedSpotImagesBaseURL())
	if err != nil {
		return fmt.Errorf("creating location service: %w", err)
	}
	apiKeyService, err := service.NewAPIKeyService(repos.apiKeyRepo)
	if err != nil {
		return fmt.Errorf("creating apikey service: %w", err)
	}
	swellPredictionService, err := service.NewSwellPredictionService(repos.swellPredictionRepo)
	if err != nil {
		return fmt.Errorf("creating swell prediction service: %w", err)
	}
	streamService, err := service.NewStreamService(repos.streamRequestRepo)
	if err != nil {
		return fmt.Errorf("creating stream service: %w", err)
	}
	snapshotService, err := service.NewSnapshotService(repos.snapshotRepo)
	if err != nil {
		return fmt.Errorf("creating snapshot service: %w", err)
	}
	buoyService, err := service.NewBuoyService(repos.buoyRepo)
	if err != nil {
		return fmt.Errorf("creating buoy service: %w", err)
	}

	services.forecastService = forecastService
	services.locationService = locationService
	services.apiKeyService = apiKeyService
	services.swellPredictionService = swellPredictionService
	services.streamService = streamService
	services.snapshotService = snapshotService
	services.buoyService = buoyService
	return nil
}

// initializeControllers creates all controllers with their service dependencies.
func initializeControllers(
	services *containerServices,
	storage *containerStorage,
	cfg *containerConfig,
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
		contentReportController:   controller.NewContentReportController(services.contentReportService, services.userService),
		moderationController:      controller.NewModerationController(services.moderationService),
	}
}
