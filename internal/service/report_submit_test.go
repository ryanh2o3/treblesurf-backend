package service

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rekognition"
)

const testSpotIdentifier = "Ireland/Donegal/Bundoran"

// createTestWebSocketService creates a minimal WebSocketService for testing
func createTestWebSocketService(
	getSubscribersFn func(ctx context.Context, spotIdentifier string) ([]string, error),
) *WebSocketService {
	wsRepo := &mockrepo.WebSocketRepo{}
	subRepo := &mockrepo.SpotSubscriptionRepo{
		GetSubscribersBySpotFn: getSubscribersFn,
	}
	return NewWebSocketService(wsRepo, subRepo, []byte("test-secret"), "", "")
}

func TestReportService_SubmitSurfReport(t *testing.T) {
	// Save original env value
	originalEnv := os.Getenv("GO_ENV")
	defer func() {
		if err := os.Setenv("GO_ENV", originalEnv); err != nil {
			t.Fatalf("failed to restore GO_ENV: %v", err)
		}
	}()
	if err := os.Setenv("GO_ENV", constants.EnvDevelopment); err != nil {
		t.Fatalf("failed to set GO_ENV: %v", err)
	}

	tests := []struct {
		report    *model.ReportWithImage
		setupMock func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService)
		name      string
		userEmail string
		userName  string
		errMsg    string
		wantErr   bool
	}{
		{
			name: "successful submission with image",
			report: &model.ReportWithImage{
				Country:       "Ireland",
				Region:        "Donegal",
				Spot:          "Bundoran",
				SurfSize:      "head-high",
				WindAmount:    "light",
				WindDirection: "offshore",
				Consistency:   "consistent",
				Quality:       "good",
				Messiness:     "clean",
				ImageData:     base64.StdEncoding.EncodeToString([]byte("fake-image-data")),
				Date:          "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if report.CountryRegionSpot != forecastTestSpotID {
							t.Errorf("unexpected country_region_spot: %s", report.CountryRegionSpot)
						}
						return nil
					},
				}
				mediaRepo := &mockrepo.MediaRepo{
					UploadFn: func(_ context.Context, _ string, _ []byte, _ string) error {
						return nil
					},
				}
				rekognitionClient := &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return &rekognition.DetectLabelsOutput{
							Labels: []*rekognition.Label{
								{Name: aws.String("Sea"), Confidence: aws.Float64(95.0)},
							},
						}, nil
					},
				}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{"user1", "user2"}, nil
				})
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: false,
		},
		{
			name: "successful submission without image",
			report: &model.ReportWithImage{
				Country:       "Ireland",
				Region:        "Donegal",
				Spot:          "Bundoran",
				SurfSize:      "head-high",
				WindAmount:    "light",
				WindDirection: "offshore",
				ImageData:     "",
				Date:          "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, _ *model.SurfReport) error {
						return nil
					},
				}
				mediaRepo := &mockrepo.MediaRepo{}
				rekognitionClient := &mockRekognitionClient{}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{}, nil
				})
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: false,
		},
		{
			name: "user not found",
			report: &model.ReportWithImage{
				Country: "Ireland",
				Region:  "Donegal",
				Spot:    "Bundoran",
			},
			userEmail: "notfound@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{}
				mediaRepo := &mockrepo.MediaRepo{}
				rekognitionClient := &mockRekognitionClient{}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name: "invalid image data",
			report: &model.ReportWithImage{
				Country:   "Ireland",
				Region:    "Donegal",
				Spot:      "Bundoran",
				ImageData: "invalid-base64",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{}
				mediaRepo := &mockrepo.MediaRepo{}
				rekognitionClient := &mockRekognitionClient{}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: true,
		},
		{
			name: "repository error on store",
			report: &model.ReportWithImage{
				Country:   "Ireland",
				Region:    "Donegal",
				Spot:      "Bundoran",
				ImageData: "",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, _ *model.SurfReport) error {
						return errors.New("database error")
					},
				}
				mediaRepo := &mockrepo.MediaRepo{}
				rekognitionClient := &mockRekognitionClient{}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService, reportRepo, mediaRepo, rekognitionClient, wsService := tt.setupMock()
			service := &ReportService{
				userService:       userService,
				reportRepo:        reportRepo,
				mediaRepo:         mediaRepo,
				rekognitionClient: rekognitionClient,
				websocketService:  wsService,
			}

			err := service.SubmitSurfReport(ctx, tt.report, tt.userEmail, tt.userName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReportService_SubmitSurfReportWithS3Image(t *testing.T) {
	// Save original env value
	originalEnv := os.Getenv("GO_ENV")
	defer func() {
		if err := os.Setenv("GO_ENV", originalEnv); err != nil {
			t.Fatalf("failed to restore GO_ENV: %v", err)
		}
	}()
	if err := os.Setenv("GO_ENV", constants.EnvDevelopment); err != nil {
		t.Fatalf("failed to set GO_ENV: %v", err)
	}

	tests := []struct {
		report    *model.ReportWithS3Image
		setupMock func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService)
		name      string
		userEmail string
		userName  string
		errMsg    string
		wantErr   bool
	}{
		{
			name: "successful submission with S3 image",
			report: &model.ReportWithS3Image{
				Country:       "Ireland",
				Region:        "Donegal",
				Spot:          "Bundoran",
				SurfSize:      "head-high",
				WindAmount:    "light",
				WindDirection: "offshore",
				ImageKey:      "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
				Date:          "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, _ *model.SurfReport) error {
						return nil
					},
				}
				mediaRepo := &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return []byte("fake-image-data"), nil
					},
				}
				rekognitionClient := &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return &rekognition.DetectLabelsOutput{
							Labels: []*rekognition.Label{
								{Name: aws.String("Sea"), Confidence: aws.Float64(95.0)},
							},
						}, nil
					},
				}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{"user1"}, nil
				})
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: false,
		},
		{
			name: "S3 image not found",
			report: &model.ReportWithS3Image{
				Country:  "Ireland",
				Region:   "Donegal",
				Spot:     "Bundoran",
				ImageKey: "surf-reports/Ireland_Donegal_Bundoran/missing.jpg",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{}
				mediaRepo := &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return nil, repository.ErrNotFound
					},
				}
				rekognitionClient := &mockRekognitionClient{}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: true,
		},
		{
			name: "image validation fails",
			report: &model.ReportWithS3Image{
				Country:  "Ireland",
				Region:   "Donegal",
				Spot:     "Bundoran",
				ImageKey: "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, repository.MediaRepository, RekognitionAPI, *WebSocketService) {
				// Set to production mode for this test
				if err := os.Setenv("GO_ENV", "production"); err != nil {
					t.Fatalf("failed to set GO_ENV: %v", err)
				}
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{}
				mediaRepo := &mockrepo.MediaRepo{
					DownloadFn: func(_ context.Context, _ string) ([]byte, error) {
						return []byte("fake-image-data"), nil
					},
				}
				rekognitionClient := &mockRekognitionClient{
					DetectLabelsFn: func(_ *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
						return &rekognition.DetectLabelsOutput{
							Labels: []*rekognition.Label{
								{Name: aws.String("Person"), Confidence: aws.Float64(95.0)},
							},
						}, nil
					},
				}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, mediaRepo, rekognitionClient, wsService
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset env after test
			defer func() {
				if err := os.Setenv("GO_ENV", originalEnv); err != nil {
					t.Fatalf("failed to restore GO_ENV: %v", err)
				}
			}()
			ctx := context.Background()
			userService, reportRepo, mediaRepo, rekognitionClient, wsService := tt.setupMock()
			service := &ReportService{
				userService:       userService,
				reportRepo:        reportRepo,
				mediaRepo:         mediaRepo,
				rekognitionClient: rekognitionClient,
				websocketService:  wsService,
			}

			err := service.SubmitSurfReportWithS3Image(ctx, tt.report, tt.userEmail, tt.userName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReportService_SubmitSurfReportWithIOSValidation(t *testing.T) {
	tests := []struct {
		report    *model.ReportWithIOSValidation
		setupMock func() (*UserService, repository.ReportRepository, *WebSocketService)
		name      string
		userEmail string
		userName  string
		errMsg    string
		wantErr   bool
	}{
		{
			name: "successful submission with image",
			report: &model.ReportWithIOSValidation{
				Country:       "Ireland",
				Region:        "Donegal",
				Spot:          "Bundoran",
				SurfSize:      "head-high",
				WindAmount:    "light",
				WindDirection: "offshore",
				ImageKey:      "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
				IOSValidated:  true,
				Date:          "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if !report.IOSValidated {
							t.Error("expected IOSValidated to be true")
						}
						return nil
					},
				}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{"user1"}, nil
				})
				return userService, reportRepo, wsService
			},
			wantErr: false,
		},
		{
			name: "successful submission with video",
			report: &model.ReportWithIOSValidation{
				Country:      "Ireland",
				Region:       "Donegal",
				Spot:         "Bundoran",
				SurfSize:     "head-high",
				VideoKey:     "surf-reports/Ireland_Donegal_Bundoran/video.mp4",
				IOSValidated: true,
				Date:         "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if report.MediaType != "video" {
							t.Errorf("expected media type 'video', got %s", report.MediaType)
						}
						return nil
					},
				}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{}, nil
				})
				return userService, reportRepo, wsService
			},
			wantErr: false,
		},
		{
			name: "successful submission with both image and video",
			report: &model.ReportWithIOSValidation{
				Country:      "Ireland",
				Region:       "Donegal",
				Spot:         "Bundoran",
				SurfSize:     "head-high",
				ImageKey:     "surf-reports/Ireland_Donegal_Bundoran/image.jpg",
				VideoKey:     "surf-reports/Ireland_Donegal_Bundoran/video.mp4",
				IOSValidated: true,
				Date:         "2024-01-15 14:30:00",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if report.MediaType != "both" {
							t.Errorf("expected media type 'both', got %s", report.MediaType)
						}
						return nil
					},
				}
				wsService := createTestWebSocketService(func(_ context.Context, _ string) ([]string, error) {
					return []string{}, nil
				})
				return userService, reportRepo, wsService
			},
			wantErr: false,
		},
		{
			name: "user not found",
			report: &model.ReportWithIOSValidation{
				Country: "Ireland",
				Region:  "Donegal",
				Spot:    "Bundoran",
			},
			userEmail: "notfound@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, wsService
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name: "repository error on store",
			report: &model.ReportWithIOSValidation{
				Country: "Ireland",
				Region:  "Donegal",
				Spot:    "Bundoran",
			},
			userEmail: "test@example.com",
			userName:  "Test User",
			setupMock: func() (*UserService, repository.ReportRepository, *WebSocketService) {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				userService, _ := NewUserService(userRepo)
				reportRepo := &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, _ *model.SurfReport) error {
						return errors.New("database error")
					},
				}
				wsService := createTestWebSocketService(nil)
				return userService, reportRepo, wsService
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService, reportRepo, wsService := tt.setupMock()
			service := &ReportService{
				userService:      userService,
				reportRepo:       reportRepo,
				websocketService: wsService,
			}

			err := service.SubmitSurfReportWithIOSValidation(ctx, tt.report, tt.userEmail, tt.userName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReportService_getSpotSubscribers(t *testing.T) {
	tests := []struct {
		setupMock func() *WebSocketService
		name      string
		country   string
		region    string
		spot      string
		wantCount int
		wantErr   bool
	}{
		{
			name:    "successful subscriber retrieval",
			country: "Ireland",
			region:  "Donegal",
			spot:    "Bundoran",
			setupMock: func() *WebSocketService {
				return createTestWebSocketService(func(_ context.Context, spotIdentifier string) ([]string, error) {
					if spotIdentifier != testSpotIdentifier {
						t.Errorf("unexpected spot identifier: %s", spotIdentifier)
					}
					return []string{"user1", "user2"}, nil
				})
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:    "no websocket service",
			country: "Ireland",
			region:  "Donegal",
			spot:    "Bundoran",
			setupMock: func() *WebSocketService {
				return nil
			},
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			wsService := tt.setupMock()
			service := &ReportService{
				websocketService: wsService,
			}

			subscribers, err := service.getSpotSubscribers(ctx, tt.country, tt.region, tt.spot)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(subscribers) != tt.wantCount {
					t.Errorf("expected %d subscribers, got %d", tt.wantCount, len(subscribers))
				}
			}
		})
	}
}

func TestReportService_broadcastToUsers(t *testing.T) {
	tests := []struct {
		message     interface{}
		setupMock   func() *WebSocketService
		name        string
		subscribers []string
		wantErr     bool
	}{
		{
			name:        "successful broadcast",
			subscribers: []string{"user1", "user2"},
			message:     map[string]interface{}{"action": "new_report"},
			setupMock: func() *WebSocketService {
				return createTestWebSocketService(nil)
			},
			wantErr: false,
		},
		{
			name:        "empty subscribers - no broadcast",
			subscribers: []string{},
			message:     map[string]interface{}{"action": "new_report"},
			setupMock: func() *WebSocketService {
				return createTestWebSocketService(nil)
			},
			wantErr: false,
		},
		{
			name:        "no websocket service - no error",
			subscribers: []string{"user1"},
			message:     map[string]interface{}{"action": "new_report"},
			setupMock: func() *WebSocketService {
				return nil
			},
			wantErr: false,
		},
		{
			name:        "broadcast error - logged but not returned",
			subscribers: []string{"user1"},
			message:     map[string]interface{}{"action": "new_report"},
			setupMock: func() *WebSocketService {
				return createTestWebSocketService(nil)
			},
			wantErr: false, // Errors are logged but not returned
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			ctx := context.Background()
			wsService := tt.setupMock()
			service := &ReportService{
				websocketService: wsService,
			}

			service.broadcastToUsers(ctx, tt.subscribers, tt.message)
			// broadcastToUsers doesn't return errors, it just logs them
		})
	}
}
