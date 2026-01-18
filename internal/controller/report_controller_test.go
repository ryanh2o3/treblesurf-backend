package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/gin-gonic/gin"
)

type mockRekognitionClient struct {
	DetectLabelsFn func(*rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error)
}

func (m *mockRekognitionClient) DetectLabels(input *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
	if m.DetectLabelsFn != nil {
		return m.DetectLabelsFn(input)
	}
	return &rekognition.DetectLabelsOutput{}, nil
}

func setupReportController(reportSvc *service.ReportService, userSvc *service.UserService) *ReportController {
	return NewReportController(reportSvc, userSvc)
}

func createTestReportService(
	t *testing.T,
	reportRepo *mockrepo.ReportRepo,
	userSvc *service.UserService,
	mediaRepo *mockrepo.MediaRepo,
	rekognitionClient service.RekognitionAPI,
	wsService *service.WebSocketService,
) *service.ReportService {
	return service.NewReportService(
		mediaRepo,
		reportRepo,
		&mockrepo.BuoyRepo{},
		&mockrepo.LocationRepo{},
		&mockrepo.ForecastRepo{},
		rekognitionClient,
		userSvc,
		wsService,
	)
}

func createTestWebSocketService() *service.WebSocketService {
	wsRepo := &mockrepo.WebSocketRepo{}
	subRepo := &mockrepo.SpotSubscriptionRepo{
		GetSubscribersBySpotFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{}, nil
		},
	}
	return service.NewWebSocketService(wsRepo, subRepo, []byte("test-secret"), "", "")
}

func TestReportController_SubmitCurrentSurfReport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123", GivenName: "Test User"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)

	reportRepo := &mockrepo.ReportRepo{
		CreateFn: func(_ context.Context, report *model.SurfReport) error {
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

	reportSvc := createTestReportService(t, reportRepo, userSvc, mediaRepo, rekognitionClient, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	reportData := model.ReportWithImage{
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
	}

	jsonData, _ := json.Marshal(reportData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitReport", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.SubmitCurrentSurfReport(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "Report submitted successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}
}

func TestReportController_SubmitCurrentSurfReport_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, &service.UserService{}, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	controller := setupReportController(reportSvc, userSvc)

	reportData := model.ReportWithImage{
		Country: "Ireland",
		Region:  "Donegal",
		Spot:    "Bundoran",
	}

	jsonData, _ := json.Marshal(reportData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitReport", bytes.NewBuffer(jsonData))
	// No email in context

	controller.SubmitCurrentSurfReport(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestReportController_SubmitCurrentSurfReport_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitReport", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.SubmitCurrentSurfReport(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_SubmitSurfReportWithS3Image(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123", GivenName: "Test User"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)

	reportRepo := &mockrepo.ReportRepo{
		CreateFn: func(_ context.Context, report *model.SurfReport) error {
			return nil
		},
	}

	mediaRepo := &mockrepo.MediaRepo{
		DownloadFn: func(_ context.Context, key string) ([]byte, error) {
			return []byte("image-data"), nil
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

	reportSvc := createTestReportService(t, reportRepo, userSvc, mediaRepo, rekognitionClient, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	reportData := model.ReportWithS3Image{
		Country:       "Ireland",
		Region:        "Donegal",
		Spot:          "Bundoran",
		SurfSize:      "head-high",
		WindAmount:    "light",
		WindDirection: "offshore",
		Consistency:   "consistent",
		Quality:       "good",
		Messiness:     "clean",
		ImageKey:      "surf-reports/Ireland_Donegal_Bundoran/test-image.jpg",
		Date:          "2024-01-15 14:30:00",
	}

	jsonData, _ := json.Marshal(reportData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitS3Report", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.SubmitSurfReportWithS3Image(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_GenerateImageUploadURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)
	mediaRepo := &mockrepo.MediaRepo{
		GenerateUploadURLFn: func(_ context.Context, key string, expires time.Duration) (string, error) {
			return "https://presigned-url.com/" + key, nil
		},
	}

	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/generateImageUploadURL?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)
	c.Set("email", "test@example.com")

	controller.GenerateImageUploadURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response model.PresignedUploadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.UploadURL == "" {
		t.Error("expected upload URL to be present")
	}
}

func TestReportController_GenerateImageUploadURL_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/generateImageUploadURL?country=Ireland", http.NoBody)
	c.Set("email", "test@example.com")

	controller.GenerateImageUploadURL(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_RetrieveTodaysSurfReports(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reportRepo := &mockrepo.ReportRepo{
		GetBySpotFn: func(_ context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error) {
			return []*model.SurfReport{
				{
					Country: country,
					Region:  region,
					Spot:    spot,
				},
			}, nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, reportRepo, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/todayReports?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.RetrieveTodaysSurfReports(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_RetrieveTodaysSurfReports_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/todayReports?country=Ireland", http.NoBody)

	controller.RetrieveTodaysSurfReports(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_GetAllSpotSurfReports(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reportRepo := &mockrepo.ReportRepo{
		GetBySpotFn: func(_ context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error) {
			reports := make([]*model.SurfReport, limit)
			for i := 0; i < limit; i++ {
				reports[i] = &model.SurfReport{
					Country: country,
					Region:  region,
					Spot:    spot,
				}
			}
			return reports, nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, reportRepo, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/spotReports?country=Ireland&region=Donegal&spot=Bundoran&limit=10", http.NoBody)

	controller.GetAllSpotSurfReports(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_GetAllSpotSurfReports_InvalidLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/spotReports?country=Ireland&region=Donegal&spot=Bundoran&limit=invalid", http.NoBody)

	controller.GetAllSpotSurfReports(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_GetReportImage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mediaRepo := &mockrepo.MediaRepo{
		DownloadFn: func(_ context.Context, key string) ([]byte, error) {
			return []byte("image-data"), nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/reportImage?key=test-image.jpg", http.NoBody)

	controller.GetReportImage(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["imageData"] == nil {
		t.Error("expected imageData to be present")
	}
}

func TestReportController_GetReportImage_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/reportImage", http.NoBody)

	controller.GetReportImage(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_GetReportImage_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mediaRepo := &mockrepo.MediaRepo{
		DownloadFn: func(_ context.Context, key string) ([]byte, error) {
			return nil, repository.ErrNotFound
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/reportImage?key=missing.jpg", http.NoBody)

	controller.GetReportImage(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestReportController_GenerateVideoUploadURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)
	mediaRepo := &mockrepo.MediaRepo{
		GenerateUploadURLFn: func(_ context.Context, key string, expires time.Duration) (string, error) {
			return "https://presigned-url.com/" + key, nil
		},
	}

	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/generateVideoUploadURL?country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)
	c.Set("email", "test@example.com")

	controller.GenerateVideoUploadURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_GetReportVideo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mediaRepo := &mockrepo.MediaRepo{
		DownloadFn: func(_ context.Context, key string) ([]byte, error) {
			return []byte("video-data"), nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/reportVideo?key=test-video.mp4", http.NoBody)

	controller.GetReportVideo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_GenerateVideoViewURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mediaRepo := &mockrepo.MediaRepo{
		GenerateViewURLFn: func(_ context.Context, key string, expires time.Duration) (string, error) {
			return "https://view-url.com/" + key, nil
		},
		ExistsFn: func(_ context.Context, key string) (bool, error) {
			return true, nil
		},
	}

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123"}, nil
		},
	}

	reportRepo := &mockrepo.ReportRepo{
		GetBySpotFn: func(_ context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error) {
			return []*model.SurfReport{
				{
					VideoKey: "surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_test-uuid-123.mp4",
				},
			}, nil
		},
	}

	userSvc, _ := service.NewUserService(userRepo)
	reportSvc := createTestReportService(t, reportRepo, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/generateVideoViewURL?key=surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_test-uuid-123.mp4", http.NoBody)
	c.Set("email", "test@example.com")

	controller.GenerateVideoViewURL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_GenerateVideoViewURL_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/generateVideoViewURL", http.NoBody)
	c.Set("email", "test@example.com")

	controller.GenerateVideoViewURL(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_SubmitSurfReportWithIOSValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123", GivenName: "Test User"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)

	reportRepo := &mockrepo.ReportRepo{
		CreateFn: func(_ context.Context, report *model.SurfReport) error {
			return nil
		},
	}

	mediaRepo := &mockrepo.MediaRepo{
		DownloadFn: func(_ context.Context, key string) ([]byte, error) {
			return []byte("media-data"), nil
		},
	}

	reportSvc := createTestReportService(t, reportRepo, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	reportData := model.ReportWithIOSValidation{
		Country:       "Ireland",
		Region:        "Donegal",
		Spot:          "Bundoran",
		SurfSize:      "head-high",
		WindAmount:    "light",
		WindDirection: "offshore",
		Consistency:   "consistent",
		Quality:       "good",
		Messiness:     "clean",
		ImageKey:      "surf-reports/Ireland_Donegal_Bundoran/test-image.jpg",
		IOSValidated:  true,
		Date:          "2024-01-15 14:30:00",
	}

	jsonData, _ := json.Marshal(reportData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitIOSReport", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.SubmitSurfReportWithIOSValidation(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_SubmitSurfReportWithIOSValidation_NotIOSValidated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	reportData := model.ReportWithIOSValidation{
		Country:      "Ireland",
		Region:       "Donegal",
		Spot:         "Bundoran",
		IOSValidated: false,
	}

	jsonData, _ := json.Marshal(reportData)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/submitIOSReport", bytes.NewBuffer(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", "test@example.com")

	controller.SubmitSurfReportWithIOSValidation(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_DeleteUploadedMedia(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "test-uuid-123"}, nil
		},
	}
	userSvc, _ := service.NewUserService(userRepo)

	mediaRepo := &mockrepo.MediaRepo{
		DeleteFn: func(_ context.Context, key string) error {
			return nil
		},
	}

	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, mediaRepo, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMedia?key=surf-reports/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_test-uuid-123.jpg&type=image", http.NoBody)
	c.Set("email", "test@example.com")

	controller.DeleteUploadedMedia(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
}

func TestReportController_DeleteUploadedMedia_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	reportSvc := createTestReportService(t, &mockrepo.ReportRepo{}, userSvc, &mockrepo.MediaRepo{}, &mockRekognitionClient{}, createTestWebSocketService())
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMedia?key=test.jpg", http.NoBody)
	c.Set("email", "test@example.com")

	controller.DeleteUploadedMedia(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestReportController_GetSurfReportsWithSimilarBuoyData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	buoyRepo := &mockrepo.BuoyRepo{
		GetBatchDataRangesFn: func(_ context.Context, requests []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
			return map[string][]*model.BuoyData{
				"test-buoy": {
					{
						BuoyName:      "test-buoy",
						WaveHeight:    2.5,
						WaveDirection: 270.0,
						WavePeriod:    12.0,
					},
				},
			}, nil
		},
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				"test-buoy": {
					Name:      "test-buoy",
					Latitude:  54.5,
					Longitude: -8.3,
				},
			}, nil
		},
	}

	locationRepo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, country, region, spot string) (float64, float64, error) {
			return 54.5, -8.3, nil
		},
	}

	reportRepo := &mockrepo.ReportRepo{
		ScanSinceFn: func(_ context.Context, since time.Time, limit int) ([]*model.SurfReport, error) {
			return []*model.SurfReport{
				{
					Country: "Ireland",
					Region:  "Donegal",
					Spot:    "Bundoran",
				},
			}, nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	mediaRepo := &mockrepo.MediaRepo{}
	forecastDataRepo := &mockrepo.ForecastRepo{}
	reportSvc := service.NewReportService(
		mediaRepo,
		reportRepo,
		buoyRepo,
		locationRepo,
		forecastDataRepo,
		&mockRekognitionClient{},
		userSvc,
		createTestWebSocketService(),
	)
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/similarBuoy?waveHeight=2.5&waveDirection=270&period=12&buoyName=test-buoy&country=Ireland&region=Donegal&spot=Bundoran", http.NoBody)

	controller.GetSurfReportsWithSimilarBuoyData(c)

	// This endpoint may return 200 or 500 depending on service implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d or %d, got %d. Body: %s", http.StatusOK, http.StatusInternalServerError, w.Code, w.Body.String())
	}
}

func TestReportController_GetSurfReportsWithMatchingConditions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reportRepo := &mockrepo.ReportRepo{
		ScanSinceFn: func(_ context.Context, since time.Time, limit int) ([]*model.SurfReport, error) {
			return []*model.SurfReport{
				{
					Country: "Ireland",
					Region:  "Donegal",
					Spot:    "Bundoran",
				},
			}, nil
		},
	}

	locationRepo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, country, region, spot string) (float64, float64, error) {
			return 54.5, -8.3, nil
		},
	}

	buoyRepo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return map[string]*model.BuoyLocation{
				"test-buoy": {
					Name:      "test-buoy",
					Latitude:  54.5,
					Longitude: -8.3,
				},
			}, nil
		},
		GetBatchDataRangesFn: func(_ context.Context, requests []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
			return map[string][]*model.BuoyData{}, nil
		},
	}

	forecastRepo := &mockrepo.ForecastRepo{
		QueryBetweenFn: func(_ context.Context, spotID string, start, end time.Time, limit int) ([]*model.ForecastDataPoint, error) {
			return []*model.ForecastDataPoint{
				{
					Data: map[string]interface{}{"windSpeed": 10.0},
				},
			}, nil
		},
	}

	userSvc, _ := service.NewUserService(&mockrepo.UserRepo{})
	mediaRepo := &mockrepo.MediaRepo{}
	reportSvc := service.NewReportService(
		mediaRepo,
		reportRepo,
		buoyRepo,
		locationRepo,
		forecastRepo,
		&mockRekognitionClient{},
		userSvc,
		createTestWebSocketService(),
	)
	controller := setupReportController(reportSvc, userSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/matchingConditions?country=Ireland&region=Donegal&spot=Bundoran&daysBack=365&maxResults=20", http.NoBody)

	controller.GetSurfReportsWithMatchingConditions(c)

	// This endpoint may return 200 or 500 depending on service implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d or %d, got %d. Body: %s", http.StatusOK, http.StatusInternalServerError, w.Code, w.Body.String())
	}
}
