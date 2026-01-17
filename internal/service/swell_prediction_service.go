package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/repository"
)

type SwellPredictionService struct {
	predictions repository.SwellPredictionRepository
}

func NewSwellPredictionService(predictions repository.SwellPredictionRepository) *SwellPredictionService {
	return &SwellPredictionService{
		predictions: predictions,
	}
}


func (s *SwellPredictionService) GetSpotSwellPrediction(
	spotName, regionName, countryName string,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetSpotPredictions(context.Background(), spotID, currentHour, 25)
}

func (s *SwellPredictionService) GetListSpotsSwellPrediction(
	spots []string, regionName, countryName string,
) ([][]map[string]interface{}, error) {
	spotIDs := make([]string, 0, len(spots))
	for _, spot := range spots {
		spotIDs = append(spotIDs, fmt.Sprintf("%s#%s#%s", countryName, regionName, spot))
	}

	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetListSpotsPredictions(context.Background(), spotIDs, currentHour, 25)
}

func (s *SwellPredictionService) GetRegionSwellPrediction(
	regionName, countryName string,
) ([]map[string]interface{}, error) {
	// Get current time rounded to the hour (UTC)
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetRegionPredictions(context.Background(), countryName, regionName, currentHour, 3)
}

func (s *SwellPredictionService) GetSpotSwellPredictionRange(
	spotName, regionName, countryName string,
	startTime, endTime time.Time,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	return s.predictions.GetSpotPredictionRange(context.Background(), spotID, startTime, endTime)
}

func (s *SwellPredictionService) GetRecentSwellPredictions(hoursBack int) ([]map[string]interface{}, error) {
	cutoffTime := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	return s.predictions.GetRecentPredictions(context.Background(), cutoffTime, 3)
}

func (s *SwellPredictionService) GetClosestAIPredictionForSpot(
	spotName, regionName, countryName string,
) (map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	now := time.Now().UTC()

	return s.predictions.GetClosestPrediction(context.Background(), spotID, now)
}