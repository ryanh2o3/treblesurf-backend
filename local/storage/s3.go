package storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// LocalS3Storage handles file storage for local development
type LocalS3Storage struct {
    basePath string
}

// NewLocalS3Storage creates a new local S3 storage handler
func NewLocalS3Storage() *LocalS3Storage {
    return &LocalS3Storage{
        basePath: "./local/data/s3",
    }
}

// PutObject stores a file locally to simulate S3
func (s *LocalS3Storage) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
    bucketName := aws.StringValue(input.Bucket)
    key := aws.StringValue(input.Key)
    
    // Create directory if it doesn't exist
    dirPath := filepath.Join(s.basePath, bucketName, filepath.Dir(key))
    if err := os.MkdirAll(dirPath, 0755); err != nil {
        return nil, err
    }
    
    // Write file
    filePath := filepath.Join(s.basePath, bucketName, key)
    
    // Read the body
    data, err := io.ReadAll(input.Body)
    if err != nil {
        return nil, err
    }
    
    // Write to file
    if err := os.WriteFile(filePath, data, 0644); err != nil {
        return nil, err
    }
    
    // Write metadata if present
    if len(input.Metadata) > 0 {
        metadataPath := filePath + ".metadata"
        var metadataContent bytes.Buffer
        
        for k, v := range input.Metadata {
            if v != nil {
                metadataContent.WriteString(fmt.Sprintf("%s=%s\n", k, *v))
            }
        }
        
        if err := os.WriteFile(metadataPath, metadataContent.Bytes(), 0644); err != nil {
            log.Printf("Warning: Failed to write metadata: %v", err)
        }
    }
    
    log.Printf("S3: Saved %s to %s", key, filePath)
    
    // Return empty response
    return &s3.PutObjectOutput{}, nil
}

// GetObject retrieves a file from local storage
func (s *LocalS3Storage) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
    bucketName := aws.StringValue(input.Bucket)
    key := aws.StringValue(input.Key)
    
    filePath := filepath.Join(s.basePath, bucketName, key)
    
    // Check if file exists
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return nil, fmt.Errorf("file not found: %s", filePath)
    }
    
    // Read file content
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    
    // Read metadata if it exists
    metadata := make(map[string]*string)
    metadataPath := filePath + ".metadata"
    if _, err := os.Stat(metadataPath); err == nil {
        metadataContent, err := os.ReadFile(metadataPath)
        if err == nil {
            lines := bytes.Split(metadataContent, []byte("\n"))
            for _, line := range lines {
                parts := bytes.SplitN(line, []byte("="), 2)
                if len(parts) == 2 {
                    key := string(parts[0])
                    value := string(parts[1])
                    metadata[key] = &value
                }
            }
        }
    }
    
    // Create response
    return &s3.GetObjectOutput{
        Body:     aws.ReadSeekCloser(bytes.NewReader(data)),
        Metadata: metadata,
    }, nil
}

// DeleteObject removes a file from local storage
func (s *LocalS3Storage) DeleteObject(input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
    bucketName := aws.StringValue(input.Bucket)
    key := aws.StringValue(input.Key)
    
    filePath := filepath.Join(s.basePath, bucketName, key)
    
    // Delete the file
    if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
        return nil, err
    }
    
    // Also delete metadata if it exists
    metadataPath := filePath + ".metadata"
    if _, err := os.Stat(metadataPath); err == nil {
        os.Remove(metadataPath)
    }
    
    return &s3.DeleteObjectOutput{}, nil
}