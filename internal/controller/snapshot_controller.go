package controller

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// This controller uses the shared dependencies from dependencies.go

// UploadSnapshotHandler handles image uploads from devices
func UploadSnapshotHandler(c *gin.Context) {
	// Get the spot ID from form data
	spotID := c.PostForm("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id is required"})
		return
	}

	// Parse timestamp if provided, otherwise use current time
	timestampStr := c.PostForm("timestamp")
	var timestamp time.Time
	var err error
	if timestampStr != "" {
		// Try multiple timestamp formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05.999999", // Python's isoformat()
			"2006-01-02T15:04:05",         // isoformat without microseconds
			"2006-01-02 15:04:05",
		}

		var parseError error
		for _, format := range formats {
			timestamp, parseError = time.Parse(format, timestampStr)
			if parseError == nil {
				break
			}
		}

		if parseError != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format. Use ISO 8601/RFC3339"})
			return
		}
	} else {
		timestamp = time.Now()
	}

	// Get the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Validate file is an image
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file must be an image"})
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close uploaded file: %v", closeErr)
		}
	}()

	// Generate a unique filename
	ext := filepath.Ext(file.Filename)
	uniqueID := uuid.New().String()
	s3Key := fmt.Sprintf("snapshots/%s/%s%s", spotID, uniqueID, ext)

	// Upload to S3
	_, err = S3Client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String("treblesurf-images"),
		Key:    aws.String(s3Key),
		Body:   src,
		ContentType: aws.String(file.Header.Get("Content-Type")),
		Metadata: map[string]*string{
			"SpotId":    aws.String(spotID),
			"Timestamp": aws.String(timestamp.Format(time.RFC3339)),
		},
	})

	if err != nil {
		log.Printf("S3 upload error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		return
	}

	// Store metadata in DynamoDB
	snapshot := SpotSnapshot{
		SpotID:     spotID,
		ImageKey:   s3Key,
		Timestamp:  timestamp,
		UploadedAt: time.Now(),
	}

	item, err := dynamodbattribute.MarshalMap(snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process snapshot metadata"})
		return
	}

	// Use UpdateItem to ensure we're always storing the latest snapshot
	_, err = DB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("SpotSnapshots"),
		Item:      item,
	})

	if err != nil {
		fmt.Print("Failed to store snapshot:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store snapshot metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Snapshot uploaded successfully",
		"image_key": s3Key,
	})
}

// GetLatestSnapshotHandler retrieves the latest snapshot for a spot
func GetLatestSnapshotHandler(c *gin.Context) {
	spotID := c.Query("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id parameter is required"})
		return
	}

	// Query DynamoDB for the latest snapshot using Query instead of GetItem
	result, err := DB.Query(&dynamodb.QueryInput{
		TableName: aws.String("SpotSnapshots"),
		KeyConditionExpression: aws.String("spot_id = :spotId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spotId": {S: aws.String(spotID)},
		},
		ScanIndexForward: aws.Bool(false), // Sort in descending order (newest first)
		Limit:            aws.Int64(1),     // Get only the most recent item
	})

	if err != nil {
		fmt.Print("Failed to retrieve:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve snapshot data"})
		return
	}

	// Check if snapshot exists
	if len(result.Items) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No snapshots available for this spot"})
		return
	}

	// Extract image key from the first (most recent) item
	var snapshot SpotSnapshot
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse snapshot data"})
		return
	}

	// Generate presigned URL for the image
	req, _ := S3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("treblesurf-images"),
		Key:    aws.String(snapshot.ImageKey),
	})

	presignedURL, err := req.Presign(15 * time.Minute) // URL valid for 15 minutes
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image_url": presignedURL,
		"timestamp": snapshot.Timestamp.Format(time.RFC3339),
		"image_key": snapshot.ImageKey,
	})
}

// SpotSnapshot represents a spot snapshot
type SpotSnapshot struct {
	SpotID     string    `json:"spot_id"`
	ImageKey   string    `json:"image_key"`
	Timestamp  time.Time `json:"timestamp"`
	UploadedAt time.Time `json:"uploaded_at"`
}
