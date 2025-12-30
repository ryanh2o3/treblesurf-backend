package storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// S3Storage defines the interface for S3 storage operations.
type S3Storage interface {
	GetObject(bucket, key string) ([]byte, error)
	PutObject(bucket, key string, data []byte, contentType string) error
	DeleteObject(bucket, key string) error
	GeneratePresignedUploadURL(bucket, key string, expires time.Duration) (string, error)
	GeneratePresignedViewURL(bucket, key string, expires time.Duration) (string, error)
}

// S3Client provides S3 storage operations using the AWS SDK.
type S3Client struct {
	client *s3.S3
}

// NewS3Storage creates a new S3 storage client for the specified AWS region.
func NewS3Storage(region string) (*S3Client, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	
	client := s3.New(sess)
	return &S3Client{client: client}, nil
}

// GetObject retrieves an object from S3 and returns its contents as a byte slice.
func (s *S3Client) GetObject(bucket, key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
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

// PutObject uploads data to S3 with the specified bucket, key, and content type.
func (s *S3Client) PutObject(bucket, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(&s3.PutObjectInput{
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

// DeleteObject removes an object from S3.
func (s *S3Client) DeleteObject(bucket, key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %v", err)
	}
	return nil
}

// GeneratePresignedUploadURL generates a presigned URL for uploading to S3
func (s *S3Client) GeneratePresignedUploadURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := s.client.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return presignedURL, nil
}

// GeneratePresignedViewURL generates a presigned URL for viewing/downloading from S3
func (s *S3Client) GeneratePresignedViewURL(bucket, key string, expires time.Duration) (string, error) {
	req, _ := s.client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	presignedURL, err := req.Presign(expires)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned view URL: %v", err)
	}

	return presignedURL, nil
}

// GetS3Client returns the underlying S3 client
func (s *S3Client) GetS3Client() *s3.S3 {
	return s.client
}
