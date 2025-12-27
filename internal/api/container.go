// Package api provides the API container for dependency injection and route setup.
package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/controller"
	"treblesurf-backend/internal/service"
	storagepkg "treblesurf-backend/internal/storage"
	localstorage "treblesurf-backend/local/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Container holds all the dependencies for the API (storage, services, controllers).
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
	
	// Controllers
	ForecastController *controller.ForecastController
	SwellPredictionController *controller.SwellPredictionController
}

// containerConfig holds configuration needed for container initialization.
type containerConfig struct {
	region     string
	bucketName string
	isLocal    bool
}

// containerStorage holds initialized storage clients.
type containerStorage struct {
	dynamoDBClient   *dynamodb.DynamoDB
	rekognitionClient *rekognition.Rekognition
	dbStorage        storagepkg.DynamoDBStorage
	s3Storage        storagepkg.S3Storage
}

// containerServices holds all service instances.
type containerServices struct {
	forecastService        *service.ForecastService
	tideService            *service.TideService
	locationService        *service.LocationService
	userService            *service.UserService
	reportService          *service.ReportService
	apiKeyService          *service.APIKeyService
	swellPredictionService *service.SwellPredictionService
	websocketService       *service.WebSocketService
}

// containerControllers holds all controller instances.
type containerControllers struct {
	forecastController        *controller.ForecastController
	swellPredictionController *controller.SwellPredictionController
}

// NewContainer creates and initializes a new API container with all dependencies.
func NewContainer() (*Container, error) {
	cfg := loadContainerConfig()
	
	storage, err := initializeStorage(cfg)
	if err != nil {
		return nil, err
	}
	
	services, err := initializeServices(storage, cfg)
	if err != nil {
		return nil, err
	}
	
	controllers := initializeControllers(services)
	
	setupGlobalDependencies(storage, services, cfg)
	
	return buildContainer(storage, services, controllers), nil
}

// loadContainerConfig loads configuration from environment variables.
func loadContainerConfig() containerConfig {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1"
	}
	
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "treblesurf-images"
	}
	
	isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
	
	return containerConfig{
		region:     region,
		bucketName: bucketName,
		isLocal:    isLocal,
	}
}

// initializeStorage initializes storage clients based on environment.
func initializeStorage(cfg containerConfig) (*containerStorage, error) {
	storage := &containerStorage{}
	
	if cfg.isLocal {
		return initializeLocalStorage(storage)
	}
	
	return initializeProductionStorage(storage, cfg.region)
}

// initializeLocalStorage initializes storage clients for local development.
func initializeLocalStorage(storage *containerStorage) (*containerStorage, error) {
	log.Println("Using local development storage clients")
	
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
	storage.s3Storage = &localS3Wrapper{client: localS3}
	
	localRekognition := getLocalRekognitionClient()
	if localRekognition == nil {
		return nil, fmt.Errorf("local Rekognition client not initialized. Make sure to call storage.InitLocal() first")
	}
	storage.rekognitionClient = localRekognition
	
	return storage, nil
}

// initializeProductionStorage initializes storage clients for production.
func initializeProductionStorage(storage *containerStorage, region string) (*containerStorage, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	storage.dynamoDBClient = dynamodb.New(sess)
	storage.rekognitionClient = rekognition.New(sess)
	
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

// initializeServices initializes all service instances.
func initializeServices(storage *containerStorage, cfg containerConfig) (*containerServices, error) {
	services := &containerServices{
		forecastService:        service.NewForecastService(storage.dynamoDBClient),
		tideService:            service.NewTideService(),
		locationService:        service.NewLocationService(storage.dbStorage, storage.s3Storage, cfg.bucketName),
		userService:            service.NewUserService(storage.dynamoDBClient),
		apiKeyService:          service.NewAPIKeyService(storage.dbStorage),
		swellPredictionService: service.NewSwellPredictionService(storage.dynamoDBClient),
	}
	
	services.reportService = service.NewReportService(
		storage.dbStorage,
		storage.s3Storage,
		storage.rekognitionClient,
		cfg.bucketName,
		services.userService,
	)
	
	jwtSecret, err := getJWTSecret(cfg.isLocal)
	if err != nil {
		return nil, err
	}
	services.websocketService = service.NewWebSocketService(storage.dbStorage, []byte(jwtSecret))
	
	return services, nil
}

// getJWTSecret retrieves or generates JWT secret for WebSocket service.
func getJWTSecret(isLocal bool) (string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if isLocal {
			jwtSecret = "default-jwt-secret" //nolint:gosec // Local development only
			log.Println(
				"WARNING: Using default JWT secret for local development. " +
					"Set JWT_SECRET environment variable in production.",
			)
			return jwtSecret, nil
		}
		return "", fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return jwtSecret, nil
}

// initializeControllers initializes all controller instances.
func initializeControllers(services *containerServices) *containerControllers {
	return &containerControllers{
		forecastController:        controller.NewForecastController(services.forecastService, services.tideService),
		swellPredictionController: controller.NewSwellPredictionController(services.swellPredictionService),
	}
}

// setupGlobalDependencies configures global dependencies and registries.
func setupGlobalDependencies(storage *containerStorage, services *containerServices, cfg containerConfig) {
	auth.InitJWTSecret()
	auth.SetDynamoDB(storage.dynamoDBClient)
	
	if err := auth.InitSessionService(); err != nil {
		log.Printf("Warning: Failed to initialize session service: %v", err)
	}
	
	s3ClientForControllers := getS3ClientForControllers(cfg)
	controller.SetGlobalDependencies(storage.dynamoDBClient, s3ClientForControllers, storage.rekognitionClient)
	
	controller.SetUserService(services.userService)
	controller.SetReportService(services.reportService)
	controller.SetLocationService(services.locationService)
	controller.SetAPIKeyService(services.apiKeyService)
	controller.SetWebSocketService(services.websocketService)
}

// getS3ClientForControllers gets the S3 client to use for controllers.
func getS3ClientForControllers(cfg containerConfig) *s3.S3 {
	if cfg.isLocal {
		return getLocalS3Client()
	}
	
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cfg.region),
	}))
	return s3.New(sess)
}

// buildContainer constructs the final Container from its components.
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
		ForecastController:      controllers.forecastController,
		SwellPredictionController: controllers.swellPredictionController,
	}
}

// Helper functions to access local storage clients
func getLocalDynamoDB() *dynamodb.DynamoDB {
	return localstorage.DB
}

func getLocalS3Client() *s3.S3 {
	return localstorage.S3Client
}

func getLocalRekognitionClient() *rekognition.Rekognition {
	return localstorage.RekognitionClient
}

// Local storage wrappers that implement the storage interfaces
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
	
	// Create a request with context
	req, _ := l.client.PutItemRequest(input)
	req.SetContext(ctx)
	
	err := req.Send()
	if err != nil {
		return nil, err
	}
	
	return req.Data.(*dynamodb.PutItemOutput), nil
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
			log.Printf("Warning: Failed to close S3 response body: %v", closeErr)
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
