package api

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

func SetDynamoDB(dynamoClient *dynamodb.DynamoDB) {
    db = dynamoClient
}

// SetS3Client allows overriding the S3 client from outside packages
func SetS3Client(s3c *s3.S3) {
    s3Client = s3c
}

// SetRekognitionClient allows overriding the Rekognition client from outside packages
func SetRekognitionClient(rekClient *rekognition.Rekognition) {
    rekognitionClient = rekClient
}

func init() {
    // Check if we're running locally first
    if os.Getenv("GO_ENV") == "development" {
        log.Println("Using local AWS services...")
        sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String("eu-west-1"),
    }))
        db = dynamodb.New(sess)
        s3Client = s3.New(sess)
        rekognitionClient = rekognition.New(sess) // Use same session for simplicity
    } else {
        // Production initialization
        sess := session.Must(session.NewSession(&aws.Config{
            Region: aws.String("eu-west-1"),
        }))
        db = dynamodb.New(sess)
        s3Client = s3.New(sess)
        rekognitionClient = rekognition.New(sess)
    }
}