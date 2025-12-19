package storage

import (
	"bytes"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

// MockRekognition provides mock implementations for image recognition
type MockRekognition struct{}

// DetectLabels always returns a successful result for local development
func (m *MockRekognition) DetectLabels(input *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
    // Always return a "surf-related" label in development mode
	if input == nil {
        return nil, awserr.New("InvalidParameterException", "Input cannot be nil", errors.New("nil input"))
    }
    
    // Check if Image is provided
    if input.Image == nil {
        return nil, awserr.New("InvalidParameterException", "Image is required", errors.New("missing image"))
    }
    
    // Check that either Bytes or S3Object is provided in Image
    if input.Image.Bytes == nil && input.Image.S3Object == nil {
        return nil, awserr.New("InvalidParameterException", "Either Image.Bytes or Image.S3Object must be provided", errors.New("invalid image format"))
    }
    confidence := float64(99.5)
    
    labels := []*rekognition.Label{
        {
            Name:       aws.String("Sea"),
            Confidence: &confidence,
        },
        {
            Name:       aws.String("Water"),
            Confidence: &confidence,
        },
        {
            Name:       aws.String("Beach"),
            Confidence: &confidence,
        },
    }
    
    // Add a wave label if the image has specific patterns (simplified check)
    if input.Image.Bytes != nil && (bytes.Contains(input.Image.Bytes, []byte("wave")) || bytes.Contains(input.Image.Bytes, []byte("surf"))) {
        labels = append(labels, &rekognition.Label{
            Name:       aws.String("Sea Waves"),
            Confidence: &confidence,
        })
    }
    
    log.Println("Mock Rekognition: Detected labels", labels)
    
    return &rekognition.DetectLabelsOutput{
        Labels: labels,
    }, nil
}

// MockKinesis provides mock implementations for Kinesis video stream
type MockKinesis struct{}

// GetHLSStreamingSessionURL returns a fake URL for local development
func (m *MockKinesis) GetHLSStreamingSessionURL(input interface{}) (interface{}, error) {
    // Return a mock streaming URL
    return struct {
        HLSStreamingSessionURL *string
    }{
        HLSStreamingSessionURL: aws.String("http://localhost:8080/mock-stream/index.m3u8"),
    }, nil
}

// MockSTS provides mock STS functionality
type MockSTS struct{}

// AssumeRole returns mock credentials for local development
func (m *MockSTS) AssumeRole(input interface{}) (interface{}, error) {
    // Return mock credentials
    expiration := time.Now().Add(1 * time.Hour)
    
    return struct {
        Credentials struct {
            AccessKeyId     *string
            SecretAccessKey *string
            SessionToken    *string
            Expiration      *time.Time
        }
    }{
        Credentials: struct {
            AccessKeyId     *string
            SecretAccessKey *string
            SessionToken    *string
            Expiration      *time.Time
        }{
            AccessKeyId:     aws.String("MOCK-ACCESS-KEY"),
            SecretAccessKey: aws.String("mock-secret-key"),
            SessionToken:    aws.String("mock-session-token"),
            Expiration:      &expiration,
        },
    }, nil
}