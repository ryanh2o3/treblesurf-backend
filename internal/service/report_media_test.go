package service

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

const testUploadURL = "https://s3.example.com/upload-url"

// mockRekognitionClient is a mock implementation of RekognitionAPI
type mockRekognitionClient struct {
	DetectLabelsFn func(input *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error)
}

func (m *mockRekognitionClient) DetectLabels(input *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
	if m.DetectLabelsFn != nil {
		return m.DetectLabelsFn(input)
	}
	return &rekognition.DetectLabelsOutput{}, nil
}

func TestReportService_GenerateImageUploadURL(t *testing.T) {
	tests := []struct {
		setupMock func() (*UserService, repository.MediaRepository)
		name      string
		country   string
		region    string
		spot      string
		userEmail string
		wantErr   bool
	}{
		{
			name:      "successful URL generation",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					GenerateUploadURLFn: func(_ context.Context, _ string, expires time.Duration) (string, error) {
						if expires != 15*time.Minute {
							t.Errorf("expected expiration 15 minutes, got %v", expires)
						}
						return testUploadURL, nil
					},
				}
				return userService, mediaRepo
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "notfound@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{}
				return userService, mediaRepo
			},
			wantErr: true,
		},
		{
			name:      "user without UUID",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "nouuid@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: ""}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{}
				return userService, mediaRepo
			},
			wantErr: true,
		},
		{
			name:      "S3 URL generation error",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					GenerateUploadURLFn: func(_ context.Context, _ string, _ time.Duration) (string, error) {
						return "", errors.New("S3 error")
					},
				}
				return userService, mediaRepo
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService, mediaRepo := tt.setupMock()
			service := &ReportService{
				userLookup: userService,
				mediaRepo:  mediaRepo,
			}

			result, err := service.GenerateImageUploadURL(ctx, tt.country, tt.region, tt.spot, tt.userEmail)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if result != nil {
					t.Error("expected nil result on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected result but got nil")
				}
				if result.UploadURL == "" {
					t.Error("expected upload URL to be set")
				}
				if result.ImageKey == "" {
					t.Error("expected image key to be set")
				}
				if result.ExpiresAt == "" {
					t.Error("expected expires at to be set")
				}
			}
		})
	}
}

func TestReportService_GenerateVideoUploadURL(t *testing.T) {
	tests := []struct {
		setupMock func() (*UserService, repository.MediaRepository)
		name      string
		country   string
		region    string
		spot      string
		userEmail string
		wantErr   bool
	}{
		{
			name:      "successful URL generation",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					GenerateUploadURLFn: func(_ context.Context, _ string, _ time.Duration) (string, error) {
						return testUploadURL, nil
					},
				}
				return userService, mediaRepo
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			country:   "Ireland",
			region:    "Donegal",
			spot:      "Bundoran",
			userEmail: "notfound@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{}
				return userService, mediaRepo
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService, mediaRepo := tt.setupMock()
			service := &ReportService{
				userLookup: userService,
				mediaRepo:  mediaRepo,
			}

			result, err := service.GenerateVideoUploadURL(ctx, tt.country, tt.region, tt.spot, tt.userEmail)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected result but got nil")
				}
				if result.UploadURL == "" {
					t.Error("expected upload URL to be set")
				}
				if result.VideoKey == "" {
					t.Error("expected video key to be set")
				}
			}
		})
	}
}

func TestReportService_GetReportImage(t *testing.T) {
	tests := []struct {
		setupMock func() repository.MediaRepository
		name      string
		imageKey  string
		wantErr   bool
	}{
		{
			name:     "successful image retrieval",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return []byte("fake-image-data"), nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:     "image not found",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/missing.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			wantErr: true,
		},
		{
			name:     "download error",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return nil, errors.New("S3 error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{mediaRepo: tt.setupMock()}

			imageData, contentType, err := service.GetReportImage(ctx, tt.imageKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if imageData != nil {
					t.Error("expected nil image data on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if imageData == nil {
					t.Fatal("expected image data but got nil")
				}
				if contentType == "" {
					t.Error("expected content type to be set")
				}
			}
		})
	}
}

func TestReportService_GetReportVideo(t *testing.T) {
	tests := []struct {
		setupMock func() repository.MediaRepository
		name      string
		videoKey  string
		wantErr   bool
	}{
		{
			name:     "successful video retrieval",
			videoKey: "surf-reports/Ireland_Donegal_Bundoran/video.mp4",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return []byte("fake-video-data"), nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:     "video not found",
			videoKey: "surf-reports/Ireland_Donegal_Bundoran/missing.mp4",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{mediaRepo: tt.setupMock()}

			videoData, contentType, err := service.GetReportVideo(ctx, tt.videoKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if videoData == nil {
					t.Fatal("expected video data but got nil")
				}
				if contentType == "" {
					t.Error("expected content type to be set")
				}
			}
		})
	}
}

func TestReportService_GenerateVideoViewURL(t *testing.T) {
	tests := []struct {
		setupMock func() (*UserService, repository.MediaRepository)
		name      string
		videoKey  string
		userEmail string
		wantErr   bool
	}{
		{
			name:      "successful view URL generation",
			videoKey:  "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_test-uuid-123.mp4",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return true, nil
					},
					GenerateViewURLFn: func(_ context.Context, _ string, expires time.Duration) (string, error) {
						if expires != 1*time.Hour {
							t.Errorf("expected expiration 1 hour, got %v", expires)
						}
						return "https://s3.example.com/view-url", nil
					},
				}
				return userService, mediaRepo
			},
			wantErr: false,
		},
		{
			name:      "video not found",
			videoKey:  "surf-reports/Ireland_Donegal_Bundoran/missing.mp4",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return false, nil
					},
				}
				return userService, mediaRepo
			},
			wantErr: true,
		},
		{
			name:      "access denied - wrong user",
			videoKey:  "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_other-uuid.mp4",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return true, nil
					},
				}
				return userService, mediaRepo
			},
			wantErr: true,
		},
		{
			name:      "empty video key",
			videoKey:  "",
			userEmail: "test@example.com",
			setupMock: func() (*UserService, repository.MediaRepository) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				mediaRepo := &mockrepo.MediaRepo{}
				return userService, mediaRepo
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService, mediaRepo := tt.setupMock()
			service := &ReportService{
				userLookup: userService,
				mediaRepo:  mediaRepo,
			}

			result, err := service.GenerateVideoViewURL(ctx, tt.videoKey, tt.userEmail)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if result != nil {
					t.Error("expected nil result on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected result but got nil")
				}
				if result.ViewURL == "" {
					t.Error("expected view URL to be set")
				}
				if result.ExpiresAt == "" {
					t.Error("expected expires at to be set")
				}
			}
		})
	}
}

func TestReportService_validateImageWithRekognition(t *testing.T) {
	// Save original env value
	originalEnv := os.Getenv("GO_ENV")
	defer os.Setenv("GO_ENV", originalEnv)

	tests := []struct {
		setupMock   func() RekognitionAPI
		name        string
		env         string
		errContains string
		imageData   []byte
		wantValid   bool
		wantErr     bool
	}{
		{
			name:      "development mode - always valid",
			imageData: []byte("fake-image-data"),
			env:       constants.EnvDevelopment,
			setupMock: func() RekognitionAPI {
				return &mockRekognitionClient{}
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "production mode - valid surf image",
			imageData: []byte("fake-image-data"),
			env:       "production",
			setupMock: func() RekognitionAPI {
				return &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return &rekognition.DetectLabelsOutput{
							Labels: []*rekognition.Label{
								{Name: aws.String("Sea"), Confidence: aws.Float64(95.0)},
								{Name: aws.String("Water"), Confidence: aws.Float64(92.0)},
							},
						}, nil
					},
				}
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "production mode - invalid image (not surf-related)",
			imageData: []byte("fake-image-data"),
			env:       "production",
			setupMock: func() RekognitionAPI {
				return &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return &rekognition.DetectLabelsOutput{
							Labels: []*rekognition.Label{
								{Name: aws.String("Person"), Confidence: aws.Float64(95.0)},
								{Name: aws.String("Indoor"), Confidence: aws.Float64(92.0)},
							},
						}, nil
					},
				}
			},
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "production mode - rekognition error",
			imageData: []byte("fake-image-data"),
			env:       "production",
			setupMock: func() RekognitionAPI {
				return &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return nil, errors.New("rekognition service error")
					},
				}
			},
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GO_ENV", tt.env)
			service := &ReportService{
				rekognitionClient: tt.setupMock(),
			}

			valid, err := service.validateImageWithRekognition(tt.imageData)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" {
					if err.Error() != "" && !contains(err.Error(), tt.errContains) {
						t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
					}
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if valid != tt.wantValid {
				t.Errorf("validateImageWithRekognition() valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestReportService_DeleteMediaFromS3(t *testing.T) {
	tests := []struct {
		setupMock func() repository.MediaRepository
		name      string
		mediaKey  string
		wantErr   bool
	}{
		{
			name:     "successful deletion",
			mediaKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DeleteFn: func(_ context.Context, _ string) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:     "empty media key",
			mediaKey: "",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{}
			},
			wantErr: true,
		},
		{
			name:     "deletion error",
			mediaKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DeleteFn: func(_ context.Context, _ string) error {
						return errors.New("S3 deletion error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{mediaRepo: tt.setupMock()}

			err := service.DeleteMediaFromS3(ctx, tt.mediaKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestReportService_ValidateImageKeyExists(t *testing.T) {
	tests := []struct {
		setupMock func() repository.MediaRepository
		name      string
		imageKey  string
		want      bool
		wantErr   bool
	}{
		{
			name:     "image exists",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return true, nil
					},
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:     "image does not exist",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/missing.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return false, nil
					},
				}
			},
			want:    false,
			wantErr: false, // ValidateImageKeyExists returns exists=false, nil when Exists returns false, nil
		},
		{
			name:     "empty image key",
			imageKey: "",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{}
			},
			want:    false,
			wantErr: true,
		},
		{
			name:     "repository error",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					ExistsFn: func(_ context.Context, _ string) (bool, error) {
						return false, errors.New("S3 error")
					},
				}
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{mediaRepo: tt.setupMock()}

			exists, err := service.ValidateImageKeyExists(ctx, tt.imageKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if exists != tt.want {
					t.Errorf("ValidateImageKeyExists() exists = %v, want %v", exists, tt.want)
				}
			}
		})
	}
}

func TestReportService_CleanupOrphanedImage(t *testing.T) {
	tests := []struct {
		setupMock func() repository.MediaRepository
		name      string
		imageKey  string
		wantErr   bool
	}{
		{
			name:     "successful cleanup",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/orphaned.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DeleteFn: func(_ context.Context, _ string) error {
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:     "empty key - no error",
			imageKey: "",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{}
			},
			wantErr: false,
		},
		{
			name:     "deletion error",
			imageKey: "surf-reports/Ireland_Donegal_Bundoran/orphaned.jpg",
			setupMock: func() repository.MediaRepository {
				return &mockrepo.MediaRepo{
					DeleteFn: func(_ context.Context, _ string) error {
						return errors.New("S3 deletion error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{mediaRepo: tt.setupMock()}

			err := service.CleanupOrphanedImage(ctx, tt.imageKey)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
