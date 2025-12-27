// Package storage provides local implementations of storage interfaces for development.
package storage

import (
	"fmt"
	"log"
	"treblesurf-backend/local/config"
	"treblesurf-backend/models"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

// AWS clients that will replace the global ones in the api package
var (
    DB                *dynamodb.DynamoDB
    S3Client          *s3.S3
    RekognitionClient *rekognition.Rekognition
)

// InitLocal initializes AWS clients for local development
func InitLocal(cfg *config.Config) error {
    log.Println("Initializing local AWS services...")
    
    // Create AWS session with local endpoints
    awsConfig := &aws.Config{
        Region:      aws.String(cfg.AWSRegion),
        Endpoint:    aws.String(cfg.DynamoDBEndpoint),
        Credentials: credentials.NewStaticCredentials("test", "test", ""),
        DisableSSL:  aws.Bool(true),
    }
    
    sess, err := session.NewSession(awsConfig)
    if err != nil {
        return err
    }
    
    // Initialize DynamoDB client
    DB = dynamodb.New(sess)
    
    // Initialize S3 client with potentially different endpoint
    s3Config := &aws.Config{
        Region:           aws.String(cfg.AWSRegion),
        Endpoint:         aws.String(cfg.S3Endpoint),
        Credentials:      credentials.NewStaticCredentials("test", "test", ""),
        DisableSSL:       aws.Bool(true),
        S3ForcePathStyle: aws.Bool(true), // Needed for LocalStack
    }
    
    s3Session, err := session.NewSession(s3Config)
    if err != nil {
        return err
    }
    S3Client = s3.New(s3Session)
    
    // Initialize mock Rekognition client
    RekognitionClient = rekognition.New(sess)
    
    // Apply the overriding to API package
    if err := applyOverrides(); err != nil {
        return err
    }
    
    // Create required tables
    if err := createLocalTables(); err != nil {
        return err
    }
    
    return nil
}


// applyOverrides replaces the AWS clients in the api package with our local implementations
func applyOverrides() error {
    // Directly set the db, s3Client, and rekognitionClient variables in the api package
    models.Registry.DynamoDB = DB
    models.Registry.S3Client = S3Client
    models.Registry.Rekognition = RekognitionClient
    
    log.Println("Successfully overrode AWS clients with local implementations")
    return nil
}

// createLocalTables creates the necessary DynamoDB tables in local development
func createLocalTables() error {
	tables := getAllTableDefinitions()
	
	for _, table := range tables {
		if err := createTableIfNotExists(table); err != nil {
			return fmt.Errorf("failed to create table %s: %w", table.name, err)
		}
	}
	
	return nil
}

// createTableIfNotExists creates a DynamoDB table if it doesn't already exist.
func createTableIfNotExists(table tableDefinition) error {
	// Check if table exists
	_, err := DB.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(table.name),
	})
	
	if err == nil {
		log.Printf("Table %s already exists", table.name)
		return nil
	}
	
	// Create table
	_, err = DB.CreateTable(&dynamodb.CreateTableInput{
		TableName:            aws.String(table.name),
		KeySchema:            table.keySchema,
		AttributeDefinitions: table.attributes,
		BillingMode:          aws.String("PAY_PER_REQUEST"),
	})
	
	if err != nil {
		log.Printf("Error creating table %s: %v", table.name, err)
		return err
	}
	
	log.Printf("Created table %s", table.name)
	return nil
}