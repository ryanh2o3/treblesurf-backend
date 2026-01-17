// Package httphandler provides local storage wrappers for development environment.
package httphandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
)

// localDynamoDBWrapper wraps the DynamoDB client for local development.
type localDynamoDBWrapper struct {
	client *dynamodb.DynamoDB
}

// Scan performs a scan operation on DynamoDB.
func (l *localDynamoDBWrapper) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	return l.client.Scan(input)
}

// Query performs a query operation on DynamoDB.
func (l *localDynamoDBWrapper) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	return l.client.Query(input)
}

// GetItem retrieves a single item from DynamoDB.
func (l *localDynamoDBWrapper) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return l.client.GetItem(input)
}

// PutItem stores an item in DynamoDB with timeout protection.
func (l *localDynamoDBWrapper) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, _ := l.client.PutItemRequest(input)
	req.SetContext(ctx)

	if err := req.Send(); err != nil {
		return nil, fmt.Errorf("putting item: %w", err)
	}

	output, ok := req.Data.(*dynamodb.PutItemOutput)
	if !ok {
		return nil, fmt.Errorf("unexpected response type from PutItem")
	}

	return output, nil
}

// UpdateItem modifies an item in DynamoDB.
func (l *localDynamoDBWrapper) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return l.client.UpdateItem(input)
}

// DeleteItem removes an item from DynamoDB.
func (l *localDynamoDBWrapper) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	return l.client.DeleteItem(input)
}

// localS3Wrapper wraps the S3 client for local development.
type localS3Wrapper struct {
	client *s3.S3
}

// GetObject retrieves an object from S3.
func (l *localS3Wrapper) GetObject(bucket, key string) ([]byte, error) {
	result, err := l.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("getting object from S3: %w", err)
	}
	defer func() {
		if closeErr := result.Body.Close(); closeErr != nil {
			slog.Warn("failed to close S3 response body", slog.Any("error", closeErr))
		}
	}()

	data, readErr := io.ReadAll(result.Body)
	if readErr != nil {
		return nil, fmt.Errorf("reading S3 object body: %w", readErr)
	}
	return data, nil
}

// PutObject stores an object in S3.
func (l *localS3Wrapper) PutObject(bucket, key string, data []byte, contentType string) error {
	_, err := l.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("putting object to S3: %w", err)
	}
	return nil
}

// DeleteObject removes an object from S3.
func (l *localS3Wrapper) DeleteObject(bucket, key string) error {
	_, err := l.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting object from S3: %w", err)
	}
	return nil
}

// GeneratePresignedUploadURL creates a presigned URL for uploading to S3.
func (l *localS3Wrapper) GeneratePresignedUploadURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := l.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("generating presigned upload URL: %w", err)
	}

	return presignedURL, nil
}

// GeneratePresignedViewURL creates a presigned URL for viewing an S3 object.
func (l *localS3Wrapper) GeneratePresignedViewURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := l.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("generating presigned view URL: %w", err)
	}

	return presignedURL, nil
}
