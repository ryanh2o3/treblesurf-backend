package service

import (
	"fmt"

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
	userLookup        UserByEmail
	spotBroadcaster   SpotNotificationBroadcaster
	reportPush        *NotificationService
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
	userLookup UserByEmail,
	spotBroadcaster SpotNotificationBroadcaster,
) (*ReportService, error) {
	switch {
	case mediaRepo == nil:
		return nil, fmt.Errorf("media repository is required")
	case reportRepo == nil:
		return nil, fmt.Errorf("report repository is required")
	case buoyRepo == nil:
		return nil, fmt.Errorf("buoy repository is required")
	case locationRepo == nil:
		return nil, fmt.Errorf("location repository is required")
	case forecastDataRepo == nil:
		return nil, fmt.Errorf("forecast data repository is required")
	case rekognitionClient == nil:
		return nil, fmt.Errorf("rekognition client is required")
	case userLookup == nil:
		return nil, fmt.Errorf("user lookup is required")
	}
	return &ReportService{
		mediaRepo:         mediaRepo,
		reportRepo:        reportRepo,
		buoyRepo:          buoyRepo,
		locationRepo:      locationRepo,
		forecastDataRepo:  forecastDataRepo,
		rekognitionClient: rekognitionClient,
		userLookup:        userLookup,
		spotBroadcaster:   spotBroadcaster,
	}, nil
}

func (s *ReportService) WithReportPush(notify *NotificationService) *ReportService {
	if s == nil {
		return nil
	}
	s.reportPush = notify
	return s
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
