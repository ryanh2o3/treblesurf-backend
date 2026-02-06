package controller

import (
	"log/slog"
	"net/http"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesisvideo"
	"github.com/aws/aws-sdk-go/service/kinesisvideoarchivedmedia"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gin-gonic/gin"
)

// StreamController handles streaming routes.
type StreamController struct {
	streams *service.StreamService
}

func NewStreamController(streams *service.StreamService) *StreamController {
	return &StreamController{streams: streams}
}

// GetStreamingCredentials generates temporary AWS credentials for streaming
func (sc *StreamController) GetStreamingCredentials(c *gin.Context) {
	apiKey, exists := c.Get("apiKey")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
		return
	}

	key, ok := apiKey.(*model.APIKey)
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
		requestLogger(c).Warn("failed to assume streaming role", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate streaming credentials"})
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
	_, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

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
		requestLogger(c).Warn("failed to get streaming data endpoint", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stream"})
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
		requestLogger(c).Warn("failed to get HLS stream URL", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load stream"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hlsUrl": *hlsOutput.HLSStreamingSessionURL,
	})
}

// RequestStreamHandler handles stream requests
func (sc *StreamController) RequestStreamHandler(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
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

	streamRequest, err := sc.streams.RequestStream(c.Request.Context(), request.SpotID, emailStr)
	if err != nil {
		requestLogger(c).Warn("failed to save stream request", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save stream request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Stream requested successfully",
		"expires_at": time.Unix(streamRequest.Expiration, 0).Format(time.RFC3339),
	})
}

// CheckStreamRequestHandler checks if a stream has been requested
func (sc *StreamController) CheckStreamRequestHandler(c *gin.Context) {
	spotID := c.Query("spot_id")
	if spotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing spot_id parameter"})
		return
	}

	// Check if a stream request exists and is still valid.
	streamRequested, err := sc.streams.IsStreamRequested(c.Request.Context(), spotID)
	if err != nil {
		requestLogger(c).Warn("failed to check stream request status", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check stream request status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stream_requested": streamRequested,
	})
}

 
