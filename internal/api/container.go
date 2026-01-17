// Package httphandler provides the API container for dependency injection and route setup.
package httphandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/controller"
	repodynamo "treblesurf-backend/internal/repository/dynamodb"
	repos3 "treblesurf-backend/internal/repository/s3"
	"treblesurf-backend/internal/service"
	storagepkg "treblesurf-backend/internal/storage"
	localstorage "treblesurf-backend/local/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Container struct {
	// Storage
	DynamoDBStorage storagepkg.DynamoDBStorage
	S3Storage       storagepkg.S3Storage
	
	// Services
	ForecastService *service.ForecastService
	TideService     *service.TideService
	LocationService *service.LocationService
	ReportService   *service.ReportService
	APIKeyService   *service.APIKeyService
	WebSocketService *service.WebSocketService
	SwellPredictionService *service.SwellPredictionService
	AuthService *auth.Service
	
	// Controllers
	ForecastController *controller.ForecastController
	SwellPredictionController *controller.SwellPredictionController
	UserController *controller.UserController
	ReportController *controller.ReportController
	LocationController *controller.LocationController
	APIKeyController *controller.APIKeyController
	BuoyController *controller.BuoyController
	StreamController *controller.StreamController
	SnapshotController *controller.SnapshotController
}

type containerConfig struct {
	region     string
	bucketName string
	isLocal    bool
	jwtSecret  string
}

type containerStorage struct {
	dynamoDBClient   *dynamodb.DynamoDB
	rekognitionClient *rekognition.Rekognition
	s3Client         *s3.S3
	dbStorage        storagepkg.DynamoDBStorage
	s3Storage        storagepkg.S3Storage
}

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
}

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

func NewContainer(cfg *config.Config) (*Container, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	containerCfg := loadContainerConfig(cfg)
	
	storage, err := initializeStorage(containerCfg)
	if err != nil {
		return nil, err
	}
	
	services, err := initializeServices(storage, containerCfg)
	if err != nil {
		return nil, err
	}
	
	controllers := initializeControllers(services, storage, containerCfg)
	
	return buildContainer(storage, services, controllers), nil
}

func loadContainerConfig(cfg *config.Config) containerConfig {
	return containerConfig{
		region:     cfg.AWS.Region,
		bucketName: cfg.AWS.BucketName,
		isLocal:    cfg.IsDevelopment(),
		jwtSecret:  cfg.Auth.JWTSecret,
	}
}

func initializeStorage(cfg containerConfig) (*containerStorage, error) {
	storage := &containerStorage{}
	
	if cfg.isLocal {
		return initializeLocalStorage(storage)
	}
	
	return initializeProductionStorage(storage, cfg.region)
}

func initializeLocalStorage(storage *containerStorage) (*containerStorage, error) {
	slog.Info("using local development storage clients")
	
	localDB := getLocalDynamoDB()
	if localDB == nil {
		return nil, fmt.Errorf("local DynamoDB client not initialized. Make sure to call storage.InitLocal() first")
	}
	storage.dynamoDBClient = localDB
	storage.dbStorage = &localDynamoDBWrapper{client: localDB}
	
	localS3 := getLocalS3Client()
	if localS3 == nil {
		return nil, fmt.Errorf("local S3 client not initialized. Make sure to call storage.InitLocal() first")
	}
	storage.s3Client = localS3
	storage.s3Storage = &localS3Wrapper{client: localS3}
	
	localRekognition := getLocalRekognitionClient()
	if localRekognition == nil {
		return nil, fmt.Errorf("local Rekognition client not initialized. Make sure to call storage.InitLocal() first")
	}
	storage.rekognitionClient = localRekognition
	
	return storage, nil
}

func initializeProductionStorage(storage *containerStorage, region string) (*containerStorage, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	storage.dynamoDBClient = dynamodb.New(sess)
	storage.rekognitionClient = rekognition.New(sess)
	storage.s3Client = s3.New(sess)
	
	var err error
	storage.dbStorage, err = storagepkg.NewDynamoDBStorage(region)
	if err != nil {
		return nil, err
	}
	
	storage.s3Storage, err = storagepkg.NewS3Storage(region)
	if err != nil {
		return nil, err
	}
	
	return storage, nil
}

func initializeServices(storage *containerStorage, cfg containerConfig) (*containerServices, error) {
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

	authService, err := auth.NewService(cfg.jwtSecret, userRepo, sessionRepo, slog.Default())
	if err != nil {
		return nil, err
	}

	services := &containerServices{
		forecastService:        service.NewForecastService(forecastRepo),
		tideService:            service.NewTideService(),
		locationService:        service.NewLocationService(locationRepo, mediaRepo),
		userService:            service.NewUserService(userRepo),
		apiKeyService:          service.NewAPIKeyService(apiKeyRepo),
		swellPredictionService: service.NewSwellPredictionService(swellPredictionRepo),
		authService:            authService,
	}
	
	services.reportService = service.NewReportService(
		mediaRepo,
		reportRepo,
		buoyRepo,
		locationRepo,
		forecastRepo,
		storage.rekognitionClient,
		services.userService,
	)
	
	services.websocketService = service.NewWebSocketService(websocketRepo, subscriptionRepo, []byte(cfg.jwtSecret))
	
	return services, nil
}

func initializeControllers(services *containerServices, storage *containerStorage, cfg containerConfig) *containerControllers {
	return &containerControllers{
		forecastController:        controller.NewForecastController(services.forecastService, services.tideService),
		swellPredictionController: controller.NewSwellPredictionController(services.swellPredictionService),
		userController:            controller.NewUserController(services.userService),
		reportController:          controller.NewReportController(services.reportService, services.userService),
		locationController:        controller.NewLocationController(services.locationService),
		apiKeyController:          controller.NewAPIKeyController(services.apiKeyService),
		buoyController:            controller.NewBuoyController(repodynamo.NewBuoyRepo(storage.dynamoDBClient, "BuoyData", "BuoyLocations")),
		streamController:          controller.NewStreamController(storage.dynamoDBClient),
		snapshotController:        controller.NewSnapshotController(storage.dynamoDBClient, storage.s3Client, cfg.bucketName),
	}
}

func buildContainer(
	storage *containerStorage,
	services *containerServices,
	controllers *containerControllers,
) *Container {
	return &Container{
		DynamoDBStorage:         storage.dbStorage,
		S3Storage:               storage.s3Storage,
		ForecastService:         services.forecastService,
		TideService:             services.tideService,
		LocationService:         services.locationService,
		ReportService:           services.reportService,
		APIKeyService:           services.apiKeyService,
		WebSocketService:        services.websocketService,
		SwellPredictionService:  services.swellPredictionService,
		AuthService:             services.authService,
		ForecastController:      controllers.forecastController,
		SwellPredictionController: controllers.swellPredictionController,
		UserController:          controllers.userController,
		ReportController:        controllers.reportController,
		LocationController:      controllers.locationController,
		APIKeyController:        controllers.apiKeyController,
		BuoyController:          controllers.buoyController,
		StreamController:        controllers.streamController,
		SnapshotController:      controllers.snapshotController,
	}
}

func getLocalDynamoDB() *dynamodb.DynamoDB {
	return localstorage.DB
}

func getLocalS3Client() *s3.S3 {
	return localstorage.S3Client
}

func getLocalRekognitionClient() *rekognition.Rekognition {
	return localstorage.RekognitionClient
}

type localDynamoDBWrapper struct {
	client *dynamodb.DynamoDB
}

func (l *localDynamoDBWrapper) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return l.client.Scan(input)
}

func (l *localDynamoDBWrapper) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	return l.client.Query(input)
}

func (l *localDynamoDBWrapper) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return l.client.GetItem(input)
}

func (l *localDynamoDBWrapper) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	// Add a timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, _ := l.client.PutItemRequest(input)
	req.SetContext(ctx)
	
	err := req.Send()
	if err != nil {
		return nil, err
	}
	
	output, ok := req.Data.(*dynamodb.PutItemOutput)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from PutItem")
	}
	
	return output, nil
}

func (l *localDynamoDBWrapper) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return l.client.UpdateItem(input)
}

func (l *localDynamoDBWrapper) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	return l.client.DeleteItem(input)
}

type localS3Wrapper struct {
	client *s3.S3
}

func (l *localS3Wrapper) GetObject(bucket, key string) ([]byte, error) {
	result, err := l.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %v", err)
	}
	defer func() {
		if closeErr := result.Body.Close(); closeErr != nil {
			slog.Warn("failed to close S3 response body", slog.Any("error", closeErr))
		}
	}()

	data, readErr := io.ReadAll(result.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %v", readErr)
	}
	return data, nil
}

func (l *localS3Wrapper) PutObject(bucket, key string, data []byte, contentType string) error {
	_, err := l.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to put object to S3: %v", err)
	}
	return nil
}

func (l *localS3Wrapper) DeleteObject(bucket, key string) error {
	_, err := l.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %v", err)
	}
	return nil
}

func (l *localS3Wrapper) GeneratePresignedUploadURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := l.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return presignedURL, nil
}

func (l *localS3Wrapper) GeneratePresignedViewURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := l.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned view URL: %v", err)
	}

	return presignedURL, nil
}
