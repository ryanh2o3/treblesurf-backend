package service

import (
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/validation"

	"github.com/aws/aws-sdk-go/service/rekognition"
)

type ReportService struct {
	mediaRepo         repository.MediaRepository
	reportRepo        repository.ReportRepository
	buoyRepo          repository.BuoyRepository
	locationRepo      repository.LocationRepository
	forecastDataRepo  repository.ForecastDataRepository
	rekognitionClient RekognitionAPI
	userService       *UserService
	websocketService  *WebSocketService
}

type RekognitionAPI interface {
	DetectLabels(input *rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error)
}

func NewReportService(
	mediaRepo repository.MediaRepository,
	reportRepo repository.ReportRepository,
	buoyRepo repository.BuoyRepository,
	locationRepo repository.LocationRepository,
	forecastDataRepo repository.ForecastDataRepository,
	rekognitionClient RekognitionAPI,
	userService *UserService,
	websocketService *WebSocketService,
) *ReportService {
	return &ReportService{
		mediaRepo:         mediaRepo,
		reportRepo:        reportRepo,
		buoyRepo:          buoyRepo,
		locationRepo:      locationRepo,
		forecastDataRepo:  forecastDataRepo,
		rekognitionClient: rekognitionClient,
		userService:       userService,
		websocketService:  websocketService,
	}
}

func (s *ReportService) IsValidSurfSize(swellSize string) bool {
	return validation.IsValidSurfSize(swellSize)
}

func (s *ReportService) IsValidWindAmount(windAmount string) bool {
	return validation.IsValidWindAmount(windAmount)
}

func (s *ReportService) IsValidWindDirection(windDirection string) bool {
	return validation.IsValidWindDirection(windDirection)
}

func (s *ReportService) IsValidSurfConditions(surfConditions string) bool {
	return validation.IsValidSurfConditions(surfConditions)
}

func (s *ReportService) IsValidSurfDifficulty(surfDifficulty string) bool {
	return validation.IsValidSurfDifficulty(surfDifficulty)
}

func (s *ReportService) IsValidMessiness(messiness string) bool {
	return validation.IsValidMessiness(messiness)
}
