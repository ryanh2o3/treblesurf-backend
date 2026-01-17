package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

// generateUploadURLParams contains common parameters for generating upload URLs
type generateUploadURLParams struct {
	user       *model.User
	keyPrefix  string
	fileExt    string
	expiration time.Duration
}

// prepareUploadURLParams validates user and prepares common parameters for URL generation
func (s *ReportService) prepareUploadURLParams(
	ctx context.Context,
	country, region, spot, userEmail string,
	fileExt string,
) (*generateUploadURLParams, error) {
	// Get the user's UUID
	user, err := s.userService.GetByEmail(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.UUID == "" {
		return nil, fmt.Errorf("user does not have a UUID")
	}

	// Generate a predictable S3 key based on location and user UUID
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", country, region, spot)
	keyPrefix := fmt.Sprintf("surf-reports/%s", countryRegionSpot)

	return &generateUploadURLParams{
		user:       user,
		keyPrefix:  keyPrefix,
		fileExt:    fileExt,
		expiration: 15 * time.Minute,
	}, nil
}

func (s *ReportService) GenerateImageUploadURL(
	ctx context.Context,
	country, region, spot, userEmail string,
) (*model.PresignedUploadResponse, error) {
	params, err := s.prepareUploadURLParams(ctx, country, region, spot, userEmail, "jpg")
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	imageKey := fmt.Sprintf(
		"%s/%s_%s.%s",
		params.keyPrefix,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		params.user.UUID,
		params.fileExt,
	)

	// Generate presigned URL valid for 15 minutes
	presignedURL, err := s.mediaRepo.GenerateUploadURL(ctx, imageKey, params.expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := currentTime.Add(params.expiration)

	return &model.PresignedUploadResponse{
		UploadURL: presignedURL,
		ImageKey:  imageKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *ReportService) GenerateVideoUploadURL(
	ctx context.Context,
	country, region, spot, userEmail string,
) (*model.VideoUploadResponse, error) {
	params, err := s.prepareUploadURLParams(ctx, country, region, spot, userEmail, "mp4")
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	videoKey := fmt.Sprintf(
		"%s/%s_%s.%s",
		params.keyPrefix,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		params.user.UUID,
		params.fileExt,
	)

	// Generate presigned URL valid for 15 minutes
	presignedURL, err := s.mediaRepo.GenerateUploadURL(ctx, videoKey, params.expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	expiresAt := currentTime.Add(params.expiration)

	return &model.VideoUploadResponse{
		UploadURL: presignedURL,
		VideoKey:  videoKey,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *ReportService) GetReportImage(ctx context.Context, imageKey string) (imageData []byte, contentType string, err error) {
	imageData, err = s.mediaRepo.Download(ctx, imageKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %v", err)
	}

	// For now, assume JPEG content type
	// TODO: Implement proper content type detection
	contentType = "image/jpeg"

	return imageData, contentType, nil
}

func (s *ReportService) GetReportVideo(ctx context.Context, videoKey string) (videoData []byte, contentType string, err error) {
	videoData, err = s.mediaRepo.Download(ctx, videoKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read video data: %v", err)
	}

	// For now, assume MP4 content type
	// TODO: Implement proper content type detection
	contentType = "video/mp4"

	return videoData, contentType, nil
}

func (s *ReportService) GenerateVideoViewURL(ctx context.Context, videoKey, userEmail string) (*model.VideoViewURLResponse, error) {
	if videoKey == "" {
		return nil, fmt.Errorf("video key is required")
	}

	user, err := s.userService.GetByEmail(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.UUID == "" {
		return nil, fmt.Errorf("user does not have a UUID")
	}

	_, err = s.mediaRepo.Download(ctx, videoKey)
	if err != nil {
		return nil, fmt.Errorf("video not found or not accessible: %v", err)
	}

	// Video keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.mp4
	if !s.canUserAccessVideo(videoKey, user.UUID) {
		return nil, fmt.Errorf("access denied: you don't have permission to view this video")
	}

	expires := 1 * time.Hour
	viewURL, err := s.mediaRepo.GenerateViewURL(ctx, videoKey, expires)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned view URL: %v", err)
	}

	expiresAt := time.Now().Add(expires)

	return &model.VideoViewURLResponse{
		ViewURL:   viewURL,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *ReportService) validateImageWithRekognition(imageData []byte) (bool, error) {
	if os.Getenv("GO_ENV") == constants.EnvDevelopment {
		// In development, always return true to allow all images
		return true, nil
	}

	input := &rekognition.DetectLabelsInput{
		Image: &rekognition.Image{
			Bytes: imageData,
		},
		MinConfidence: aws.Float64(90.0),
	}

	result, err := s.rekognitionClient.DetectLabels(input)
	if err != nil {
		return false, model.NewImageValidationError(err, "image analysis failed")
	}

	validLabels := []string{"Sea", "Water", "Sea Waves", "Beach", "Coast"}
	var detectedLabels []string

	for _, label := range result.Labels {
		detectedLabels = append(detectedLabels, *label.Name)
		for _, validLabel := range validLabels {
			if strings.EqualFold(*label.Name, validLabel) {
				return true, nil
			}
		}
	}

	// Return a helpful error message with detected labels
	if len(detectedLabels) > 0 {
		return false, model.ErrImageNotSurfRelated
	}

	return false, model.ErrImageAnalysisFailed
}

func (s *ReportService) uploadImageToS3(ctx context.Context, imageData []byte, key string) (string, error) {
	err := s.mediaRepo.Upload(ctx, key, imageData, "image/jpeg")
	if err != nil {
		return "", model.NewImageValidationError(err, "failed to upload image to S3")
	}

	return key, nil
}

func (s *ReportService) ValidateImageKeyExists(ctx context.Context, imageKey string) (bool, error) {
	if imageKey == "" {
		return false, fmt.Errorf("image key is empty")
	}

	// Try to get the object metadata to check if it exists
	_, err := s.mediaRepo.Download(ctx, imageKey)
	if err != nil {
		return false, fmt.Errorf("image key %s does not exist or is not accessible: %v", imageKey, err)
	}

	return true, nil
}

func (s *ReportService) CleanupOrphanedImage(ctx context.Context, imageKey string) error {
	if imageKey == "" {
		return nil
	}

	slog.Info("cleaning up orphaned image", slog.String("key", imageKey))
	err := s.mediaRepo.Delete(ctx, imageKey)
	if err != nil {
		slog.Warn("failed to cleanup orphaned image", slog.String("key", imageKey), slog.Any("error", err))
		return fmt.Errorf("failed to cleanup orphaned image: %v", err)
	}

	slog.Info("successfully cleaned up orphaned image", slog.String("key", imageKey))
	return nil
}

func (s *ReportService) DeleteMediaFromS3(ctx context.Context, mediaKey string) error {
	if mediaKey == "" {
		return fmt.Errorf("media key is required")
	}

	slog.Info("deleting media", slog.String("key", mediaKey))
	err := s.mediaRepo.Delete(ctx, mediaKey)
	if err != nil {
		slog.Warn("failed to delete media", slog.String("key", mediaKey), slog.Any("error", err))
		return fmt.Errorf("failed to delete media from S3: %v", err)
	}

	slog.Info("successfully deleted media", slog.String("key", mediaKey))
	return nil
}

func (s *ReportService) canUserAccessVideo(videoKey, userUUID string) bool {
	// Video keys follow the pattern: surf-reports/Country_Region_Spot/Timestamp_UUID.mp4
	// We need to extract the UUID from the video key to verify ownership

	// Split the video key by "/" to get the parts
	parts := strings.Split(videoKey, "/")
	if len(parts) < 3 {
		slog.Warn("invalid video key format", slog.String("key", videoKey))
		return false
	}

	// Get the filename part (last part)
	filename := parts[len(parts)-1]

	// Remove the .mp4 extension
	if !strings.HasSuffix(filename, ".mp4") {
		slog.Warn("video key does not end with .mp4", slog.String("key", videoKey))
		return false
	}

	filenameWithoutExt := strings.TrimSuffix(filename, ".mp4")

	// Split by "_" to get timestamp and UUID
	fileParts := strings.Split(filenameWithoutExt, "_")
	if len(fileParts) < 2 {
		slog.Warn("invalid video key filename format", slog.String("filename", filename))
		return false
	}

	// The UUID should be the last part after splitting by "_"
	videoUUID := fileParts[len(fileParts)-1]

	// Check if the UUID matches the user's UUID
	if videoUUID != userUUID {
		slog.Warn("video UUID does not match user UUID", slog.String("video_uuid", videoUUID), slog.String("user_uuid", userUUID))
		return false
	}

	return true
}
