package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/kinesisvideo"
	"github.com/aws/aws-sdk-go/service/kinesisvideoarchivedmedia"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gin-gonic/gin"
)

// StreamController handles streaming routes.
type StreamController struct {
	db *dynamodb.DynamoDB
}

func NewStreamController(db *dynamodb.DynamoDB) *StreamController {
	return &StreamController{db: db}
}

// GetStreamingCredentials generates temporary AWS credentials for streaming
func (sc *StreamController) GetStreamingCredentials(c *gin.Context) {
	apiKey, exists := c.Get("apiKey")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		return
	}

	key, ok := apiKey.(*APIKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key in context"})
		return
	}

	// Create an STS client
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	}))

	// Request temporary credentials with proper permissions
	stsClient := sts.New(sess)
	result, err := stsClient.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String("arn:aws:iam::759663378274:role/TreblesurfPiStreamingRole"),
		RoleSessionName: aws.String("device-stream-" + key.KeyID),
		DurationSeconds: aws.Int64(3600), // 1 hour
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessKey":    *result.Credentials.AccessKeyId,
		"secretKey":    *result.Credentials.SecretAccessKey,
		"sessionToken": *result.Credentials.SessionToken,
		"expiration":   result.Credentials.Expiration.Format(time.RFC3339),
	})
}

// GetStreamPlaybackURL generates a signed URL for viewing the stream
func (sc *StreamController) GetStreamPlaybackURL(c *gin.Context) {
	// Only authenticated users can access this endpoint
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	fmt.Print(email)

	// You can add additional authorization checks here
	// For example, check if the user has permission to view this camera

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	}))

	kvsClient := kinesisvideo.New(sess)

	getDataEndpointOutput, err := kvsClient.GetDataEndpoint(&kinesisvideo.GetDataEndpointInput{
		StreamName: aws.String("treblesurf-webcam"),
		APIName:    aws.String("GET_HLS_STREAMING_SESSION_URL"),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	archiveClient := kinesisvideoarchivedmedia.New(sess, &aws.Config{
		Endpoint: getDataEndpointOutput.DataEndpoint,
	})

	hlsOutput, err := archiveClient.GetHLSStreamingSessionURL(&kinesisvideoarchivedmedia.GetHLSStreamingSessionURLInput{
		StreamName:   aws.String("treblesurf-webcam"),
		PlaybackMode: aws.String("LIVE"), // Use "ON_DEMAND" for recorded content
		HLSFragmentSelector: &kinesisvideoarchivedmedia.HLSFragmentSelector{
			FragmentSelectorType: aws.String("SERVER_TIMESTAMP"),
		},
		ContainerFormat:   aws.String("FRAGMENTED_MP4"),
		DiscontinuityMode: aws.String("ALWAYS"),
		Expires:           aws.Int64(3600), // URL valid for 1 hour
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hlsUrl": *hlsOutput.HLSStreamingSessionURL,
	})
}

// RequestStreamHandler handles stream requests
func (sc *StreamController) RequestStreamHandler(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var request struct {
		SpotID string `json:"spot_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format. Spot ID is required."})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}

	now := time.Now()
	expiration := now.Add(5 * time.Minute).Unix()

	streamRequest := StreamRequest{
		SpotID:      request.SpotID,
		RequestedBy: emailStr,
		RequestedAt: now,
		Expiration:  expiration,
	}

	item, err := dynamodbattribute.MarshalMap(streamRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	_, err = sc.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("StreamRequests"),
		Item:      item,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save stream request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Stream requested successfully",
		"expires_at": now.Add(5 * time.Minute).Format(time.RFC3339),
	})
}

// CheckStreamRequestHandler checks if a stream has been requested
func (sc *StreamController) CheckStreamRequestHandler(c *gin.Context) {
	spotID := c.Query("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing spot_id parameter"})
		return
	}

	// Query DynamoDB for this spot ID
	result, err := sc.db.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("StreamRequests"),
		Key: map[string]*dynamodb.AttributeValue{
			"spot_id": {S: aws.String(spotID)},
		},
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stream request status"})
		return
	}

	streamRequested := len(result.Item) > 0

	if streamRequested {
		var request StreamRequest
		if err := dynamodbattribute.UnmarshalMap(result.Item, &request); err == nil {
			if time.Now().Unix() > request.Expiration {
				streamRequested = false
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"stream_requested": streamRequested,
	})
}

// StreamRequest represents a stream request
type StreamRequest struct {
	RequestedAt time.Time `json:"requested_at"`
	SpotID      string    `json:"spot_id"`
	RequestedBy string    `json:"requested_by"`
	Expiration  int64     `json:"expiration"`
}

// APIKey represents an API key
type APIKey struct {
	KeyID string `json:"key_id"`
	// Add other fields as needed
}
