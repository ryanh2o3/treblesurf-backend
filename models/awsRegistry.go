package models

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/aws/aws-sdk-go/service/s3"
)

// AWSClientRegistry holds AWS clients that can be used across packages
type AWSClientRegistry struct {
    DynamoDB      *dynamodb.DynamoDB
    S3Client      *s3.S3
    Rekognition   *rekognition.Rekognition
}

// Global instance that can be set during initialization
var Registry AWSClientRegistry