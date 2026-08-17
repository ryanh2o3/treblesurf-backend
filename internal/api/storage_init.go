// Package httphandler provides storage initialization for the API container.
package httphandler

import (
	"fmt"
	"log/slog"

	storagepkg "treblesurf-backend/internal/storage"
	localstorage "treblesurf-backend/local/storage"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

// containerStorage holds initialized storage clients and wrappers.
type containerStorage struct {
	dynamoDBClient    *dynamodb.DynamoDB
	rekognitionClient *rekognition.Rekognition
	s3Client          *s3.S3
	dbStorage         storagepkg.DynamoDBStorage
	s3Storage         storagepkg.S3Storage
}

// initializeStorage creates and configures all storage clients based on environment.
func initializeStorage(cfg *containerConfig) (*containerStorage, error) {
	storage := &containerStorage{}

	if cfg.isLocal {
		return initializeLocalStorage(storage)
	}

	return initializeProductionStorage(storage, cfg.region)
}

// initializeLocalStorage sets up storage clients for local development.
func initializeLocalStorage(storage *containerStorage) (*containerStorage, error) {
	slog.Info("using local development storage clients")

	localDB := getLocalDynamoDB()
	if localDB == nil {
		return nil, fmt.Errorf("local DynamoDB client not initialized: call storage.InitLocal() first")
	}
	storage.dynamoDBClient = localDB
	storage.dbStorage = &localDynamoDBWrapper{client: localDB}

	localS3 := getLocalS3Client()
	if localS3 == nil {
		return nil, fmt.Errorf("local S3 client not initialized: call storage.InitLocal() first")
	}
	storage.s3Client = localS3
	storage.s3Storage = &localS3Wrapper{client: localS3}

	localRekognition := getLocalRekognitionClient()
	if localRekognition == nil {
		return nil, fmt.Errorf("local Rekognition client not initialized: call storage.InitLocal() first")
	}
	storage.rekognitionClient = localRekognition

	return storage, nil
}

// initializeProductionStorage sets up storage clients for production AWS environment.
func initializeProductionStorage(storage *containerStorage, region string) (*containerStorage, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	// One DynamoDB client per process (shared across all repos). Do not create per request.
	storage.dynamoDBClient = dynamodb.New(sess)
	storage.rekognitionClient = rekognition.New(sess)
	storage.s3Client = s3.New(sess)

	var err error
	storage.dbStorage, err = storagepkg.NewDynamoDBStorage(region)
	if err != nil {
		return nil, fmt.Errorf("creating dynamodb storage: %w", err)
	}

	storage.s3Storage, err = storagepkg.NewS3Storage(region)
	if err != nil {
		return nil, fmt.Errorf("creating s3 storage: %w", err)
	}

	return storage, nil
}

// Local client accessors

func getLocalDynamoDB() *dynamodb.DynamoDB {
	return localstorage.DB
}

func getLocalS3Client() *s3.S3 {
	return localstorage.S3Client
}

func getLocalRekognitionClient() *rekognition.Rekognition {
	return localstorage.RekognitionClient
}
