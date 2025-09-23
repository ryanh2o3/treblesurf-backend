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
	"treblesurf-backend/internal/controller"
	"treblesurf-backend/internal/service"
	"treblesurf-backend/internal/storage"
	localstorage "treblesurf-backend/local/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Container struct {
	// Storage
	DynamoDBStorage storage.DynamoDBStorage
	S3Storage       storage.S3Storage
	
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

func NewContainer() (*Container, error) {
	// Get configuration from environment
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1" // default
	}
	
	bucketName := os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "treblesurf-images" // default
	}

	// Check if we're running locally
	isLocal := os.Getenv("GO_ENV") == "development"
	
	var dynamoDBClient *dynamodb.DynamoDB
	var rekognitionClient *rekognition.Rekognition
	var dbStorage storage.DynamoDBStorage
	var s3Storage storage.S3Storage
	var err error

	if isLocal {
		// In local development, use the local storage clients
		log.Println("Using local development storage clients")
		
		// Check if local clients are available
		if localDB := getLocalDynamoDB(); localDB != nil {
			dynamoDBClient = localDB
			// Create a simple wrapper that implements the storage interface
			dbStorage = &localDynamoDBWrapper{client: localDB}
		} else {
			return nil, fmt.Errorf("local DynamoDB client not initialized. Make sure to call storage.InitLocal() first")
		}
		
		if localS3 := getLocalS3Client(); localS3 != nil {
			// Create a simple wrapper that implements the storage interface
			s3Storage = &localS3Wrapper{client: localS3}
		} else {
			return nil, fmt.Errorf("local S3 client not initialized. Make sure to call storage.InitLocal() first")
		}
		
		if localRekognition := getLocalRekognitionClient(); localRekognition != nil {
			rekognitionClient = localRekognition
		} else {
			return nil, fmt.Errorf("local Rekognition client not initialized. Make sure to call storage.InitLocal() first")
		}
	} else {
		// In production, create new AWS session
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		// Initialize AWS clients
		dynamoDBClient = dynamodb.New(sess)
		rekognitionClient = rekognition.New(sess)

		// Initialize storage clients
		dbStorage, err = storage.NewDynamoDBStorage(region)
		if err != nil {
			return nil, err
		}

		s3Storage, err = storage.NewS3Storage(region)
		if err != nil {
			return nil, err
		}
	}

	// Initialize services
	forecastService := service.NewForecastService(dynamoDBClient)
	tideService := service.NewTideService()
	locationService := service.NewLocationService(dbStorage, s3Storage, bucketName)
	userService := service.NewUserService(dynamoDBClient)
	reportService := service.NewReportService(dbStorage, s3Storage, rekognitionClient, bucketName, userService)
	apiKeyService := service.NewAPIKeyService(dbStorage)
	swellPredictionService := service.NewSwellPredictionService(dynamoDBClient)
	
	// Initialize WebSocket service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-jwt-secret" // fallback for local development
	}
	websocketService := service.NewWebSocketService(dbStorage, []byte(jwtSecret))

	// Initialize auth service
	auth.InitJWTSecret()
	auth.SetDynamoDB(dynamoDBClient)
	
	// Initialize session service
	if err := auth.InitSessionService(); err != nil {
		log.Printf("Warning: Failed to initialize session service: %v", err)
		// Continue without session service for now
	}

	// Initialize controllers
	forecastController := controller.NewForecastController(forecastService, tideService)
	swellPredictionController := controller.NewSwellPredictionController(swellPredictionService)

	// Set global dependencies for all controllers
	// For local development, use the local S3 client directly
	var s3ClientForControllers *s3.S3
	if isLocal {
		s3ClientForControllers = getLocalS3Client()
	} else {
		// For production, we need to get the S3 client from storage
		// This is a bit of a hack, but we'll create a temporary client
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))
		s3ClientForControllers = s3.New(sess)
	}
	controller.SetGlobalDependencies(dynamoDBClient, s3ClientForControllers, rekognitionClient)

	// Set services in the shared registry
	controller.SetUserService(userService)
	controller.SetReportService(reportService)
	controller.SetLocationService(locationService)
	controller.SetAPIKeyService(apiKeyService)
	controller.SetWebSocketService(websocketService)

	return &Container{
		// Storage
		DynamoDBStorage: dbStorage,
		S3Storage:       s3Storage,
		
		// Services
		ForecastService: forecastService,
		TideService:     tideService,
		LocationService: locationService,
		ReportService:   reportService,
		APIKeyService:   apiKeyService,
		WebSocketService: websocketService,
		SwellPredictionService: swellPredictionService,
		
		// Controllers
		ForecastController: forecastController,
		SwellPredictionController: swellPredictionController,
	}, nil
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
	defer result.Body.Close()

	return io.ReadAll(result.Body)
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
