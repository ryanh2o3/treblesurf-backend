package controller

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

// Global dependencies shared across controllers
var (
	// Database
	DB *dynamodb.DynamoDB
	
	// Storage
	S3Client *s3.S3
	
	// AI Services
	RekognitionClient *rekognition.Rekognition
)

// SetGlobalDependencies sets the shared dependencies for all controllers
func SetGlobalDependencies(db *dynamodb.DynamoDB, s3Client *s3.S3, rekognitionClient *rekognition.Rekognition) {
	DB = db
	S3Client = s3Client
	RekognitionClient = rekognitionClient
}
