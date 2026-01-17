// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/repository"
)

// SwellPredictionService provides business logic for swell prediction operations.
type SwellPredictionService struct {
	predictions repository.SwellPredictionRepository
}

// NewSwellPredictionService creates a new SwellPredictionService with the given repository.
// Returns an error if the repository is nil.
func NewSwellPredictionService(predictions repository.SwellPredictionRepository) (*SwellPredictionService, error) {
	if predictions == nil {
		return nil, fmt.Errorf("swell prediction repository is required")
	}
	return &SwellPredictionService{
		predictions: predictions,
	}, nil
}


func (s *SwellPredictionService) GetSpotSwellPrediction(
	ctx context.Context,
	spotName, regionName, countryName string,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetSpotPredictions(ctx, spotID, currentHour, 25)
}

func (s *SwellPredictionService) GetListSpotsSwellPrediction(
	ctx context.Context,
	spots []string, regionName, countryName string,
) ([][]map[string]interface{}, error) {
	spotIDs := make([]string, 0, len(spots))
	for _, spot := range spots {
		spotIDs = append(spotIDs, fmt.Sprintf("%s#%s#%s", countryName, regionName, spot))
	}

	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetListSpotsPredictions(ctx, spotIDs, currentHour, 25)
}

func (s *SwellPredictionService) GetRegionSwellPrediction(
	ctx context.Context,
	regionName, countryName string,
) ([]map[string]interface{}, error) {
	// Get current time rounded to the hour (UTC)
	now := time.Now().UTC()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	return s.predictions.GetRegionPredictions(ctx, countryName, regionName, currentHour, 3)
}

func (s *SwellPredictionService) GetSpotSwellPredictionRange(
	ctx context.Context,
	spotName, regionName, countryName string,
	startTime, endTime time.Time,
) ([]map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	return s.predictions.GetSpotPredictionRange(ctx, spotID, startTime, endTime)
}

func (s *SwellPredictionService) GetRecentSwellPredictions(ctx context.Context, hoursBack int) ([]map[string]interface{}, error) {
	cutoffTime := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	return s.predictions.GetRecentPredictions(ctx, cutoffTime, 3)
}

func (s *SwellPredictionService) GetClosestAIPredictionForSpot(
	ctx context.Context,
	spotName, regionName, countryName string,
) (map[string]interface{}, error) {
	spotID := fmt.Sprintf("%s#%s#%s", countryName, regionName, spotName)
	now := time.Now().UTC()

	return s.predictions.GetClosestPrediction(ctx, spotID, now)
}