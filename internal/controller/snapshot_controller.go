package controller

import (
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SnapshotController handles snapshot routes.
type SnapshotController struct {
	db        *dynamodb.DynamoDB
	s3Client  *s3.S3
	bucketName string
}

func NewSnapshotController(db *dynamodb.DynamoDB, s3Client *s3.S3, bucketName string) *SnapshotController {
	return &SnapshotController{
		db:         db,
		s3Client:   s3Client,
		bucketName: bucketName,
	}
}

// UploadSnapshotHandler handles image uploads from devices
func (sc *SnapshotController) UploadSnapshotHandler(c *gin.Context) {
	spotID, timestamp, file := sc.validateSnapshotUpload(c)
	if spotID == "" {
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			slog.Warn("failed to close uploaded file", slog.Any("error", closeErr))
		}
	}()

	s3Key := generateSnapshotS3Key(spotID, file.Filename)
	if err := sc.uploadSnapshotToS3(src, s3Key, file.Header.Get("Content-Type"), spotID, timestamp); err != nil {
		slog.Warn("S3 upload error", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
		return
	}

	if err := sc.storeSnapshotMetadata(spotID, s3Key, timestamp); err != nil {
		slog.Warn("failed to store snapshot metadata", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store snapshot metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Snapshot uploaded successfully",
		"image_key": s3Key,
	})
}

// validateSnapshotUpload validates and extracts snapshot upload parameters.
func (sc *SnapshotController) validateSnapshotUpload(c *gin.Context) (string, time.Time, *multipart.FileHeader) {
	spotID := c.PostForm("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id is required"})
		return "", time.Time{}, nil
	}

	timestamp, err := parseTimestamp(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp format. Use ISO 8601/RFC3339"})
		return "", time.Time{}, nil
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return "", time.Time{}, nil
	}

	contentType := file.Header.Get("Content-Type")
	if !validateImageFile(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file must be an image"})
		return "", time.Time{}, nil
	}

	return spotID, timestamp, file
}

// uploadSnapshotToS3 uploads a snapshot file to S3.
func (sc *SnapshotController) uploadSnapshotToS3(src multipart.File, s3Key, contentType, spotID string, timestamp time.Time) error {
	_, err := sc.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(sc.bucketName),
		Key:         aws.String(s3Key),
		Body:        src,
		ContentType: aws.String(contentType),
		Metadata: map[string]*string{
			"SpotId":    aws.String(spotID),
			"Timestamp": aws.String(timestamp.Format(time.RFC3339)),
		},
	})
	return err
}

// generateSnapshotS3Key generates a unique S3 key for a snapshot.
func generateSnapshotS3Key(spotID, filename string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	return fmt.Sprintf("snapshots/%s/%s%s", spotID, uniqueID, ext)
}

// storeSnapshotMetadata stores snapshot metadata in DynamoDB.
func (sc *SnapshotController) storeSnapshotMetadata(spotID, s3Key string, timestamp time.Time) error {
	snapshot := SpotSnapshot{
		SpotID:     spotID,
		ImageKey:   s3Key,
		Timestamp:  timestamp,
		UploadedAt: time.Now(),
	}

	item, err := dynamodbattribute.MarshalMap(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	_, err = sc.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("SpotSnapshots"),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to store snapshot: %w", err)
	}

	return nil
}

// GetLatestSnapshotHandler retrieves the latest snapshot for a spot
func (sc *SnapshotController) GetLatestSnapshotHandler(c *gin.Context) {
	spotID := c.Query("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spot_id parameter is required"})
		return
	}

	// Query DynamoDB for the latest snapshot using Query instead of GetItem
	result, err := sc.db.Query(&dynamodb.QueryInput{
		TableName:              aws.String("SpotSnapshots"),
		KeyConditionExpression: aws.String("spot_id = :spotId"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":spotId": {S: aws.String(spotID)},
		},
		ScanIndexForward: aws.Bool(false), // Sort in descending order (newest first)
		Limit:            aws.Int64(1),    // Get only the most recent item
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
	req, _ := sc.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(sc.bucketName),
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
	Timestamp  time.Time `json:"timestamp"`
	UploadedAt time.Time `json:"uploaded_at"`
	SpotID     string    `json:"spot_id"`
	ImageKey   string    `json:"image_key"`
}
