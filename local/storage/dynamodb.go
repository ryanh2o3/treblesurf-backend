package storage

import (
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
    tables := []struct {
        name       string
        keySchema  []*dynamodb.KeySchemaElement
        attributes []*dynamodb.AttributeDefinition
    }{
        {
            name: "LocationData",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("country_region_spot"),
                    KeyType:       aws.String("HASH"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("country_region_spot"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "SpotForecastData",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("spot_id"),
                    KeyType:       aws.String("HASH"),
                },
                {
                    AttributeName: aws.String("forecast_timestamp"),
                    KeyType:       aws.String("RANGE"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("spot_id"),
                    AttributeType: aws.String("S"),
                },
                {
                    AttributeName: aws.String("forecast_timestamp"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "BuoyData",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("region_buoy"),
                    KeyType:       aws.String("HASH"),
                },
                {
                    AttributeName: aws.String("dataDateTime"),
                    KeyType:       aws.String("RANGE"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("region_buoy"),
                    AttributeType: aws.String("S"),
                },
                {
                    AttributeName: aws.String("dataDateTime"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "BuoyLocations",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("region_buoy"),
                    KeyType:       aws.String("HASH"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("region_buoy"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "SurfReports",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("country_region_spot"),
                    KeyType:       aws.String("HASH"),
                },
                {
                    AttributeName: aws.String("dateReported"),
                    KeyType:       aws.String("RANGE"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("country_region_spot"),
                    AttributeType: aws.String("S"),
                },
                {
                    AttributeName: aws.String("dateReported"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "Users",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("email"),
                    KeyType:       aws.String("HASH"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("email"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "SpotSnapshots",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("spot_id"),
                    KeyType:       aws.String("HASH"),
                },
                {
                    AttributeName: aws.String("timestamp"),
                    KeyType:       aws.String("RANGE"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("spot_id"),
                    AttributeType: aws.String("S"),
                },
                {
                    AttributeName: aws.String("timestamp"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "StreamRequests",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("spot_id"),
                    KeyType:       aws.String("HASH"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("spot_id"),
                    AttributeType: aws.String("S"),
                },
            },
        },
        {
            name: "ApiKeys",
            keySchema: []*dynamodb.KeySchemaElement{
                {
                    AttributeName: aws.String("key_id"),
                    KeyType:       aws.String("HASH"),
                },
            },
            attributes: []*dynamodb.AttributeDefinition{
                {
                    AttributeName: aws.String("key_id"),
                    AttributeType: aws.String("S"),
                },
            },
        },
    }

    for _, table := range tables {
        // Check if table exists
        _, err := DB.DescribeTable(&dynamodb.DescribeTableInput{
            TableName: aws.String(table.name),
        })
        
        if err == nil {
            log.Printf("Table %s already exists", table.name)
            continue
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
    }
    
    return nil
}