package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

type LocationService struct {
	locations repository.LocationRepository
	media     repository.MediaRepository
}

func NewLocationService(
	locations repository.LocationRepository,
	media repository.MediaRepository,
) *LocationService {
	return &LocationService{
		locations: locations,
		media:     media,
	}
}

func (s *LocationService) GetRegions(countryName string) ([]string, error) {
	return s.GetRegionsWithContext(context.Background(), countryName)
}

func (s *LocationService) GetRegionsWithContext(ctx context.Context, countryName string) ([]string, error) {
	return s.locations.GetRegions(ctx, countryName)
}

func (s *LocationService) GetSpots(countryName, regionName string) ([]model.LocationInfo, error) {
	return s.GetSpotsWithContext(context.Background(), countryName, regionName)
}

func (s *LocationService) GetSpotsWithContext(ctx context.Context, countryName, regionName string) ([]model.LocationInfo, error) {
	spots, err := s.locations.GetSpots(ctx, countryName, regionName)
	if err != nil {
		return nil, err
	}

	locations := make([]model.LocationInfo, 0, len(spots))
	for _, spot := range spots {
		if spot == nil {
			continue
		}
		location := *spot
		s.populateImage(ctx, &location, countryName, regionName)
		locations = append(locations, location)
	}

	return locations, nil
}

func (s *LocationService) GetLocationInfo(countryName, regionName, spotName string) (*model.LocationInfo, error) {
	return s.GetLocationInfoWithContext(context.Background(), countryName, regionName, spotName)
}

func (s *LocationService) GetLocationInfoWithContext(
	ctx context.Context,
	countryName, regionName, spotName string,
) (*model.LocationInfo, error) {
	location, err := s.locations.GetLocationInfo(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}

	s.populateImage(ctx, location, countryName, regionName)
	return location, nil
}

func (s *LocationService) GetCoordinates(countryName, regionName, spotName string) ([]float64, error) {
	return s.GetCoordinatesWithContext(context.Background(), countryName, regionName, spotName)
}

func (s *LocationService) GetCoordinatesWithContext(
	ctx context.Context,
	countryName, regionName, spotName string,
) ([]float64, error) {
	lat, lon, err := s.locations.GetCoordinates(ctx, countryName, regionName, spotName)
	if err != nil {
		return nil, err
	}
	return []float64{lat, lon}, nil
}

func (s *LocationService) populateImage(
	ctx context.Context,
	location *model.LocationInfo,
	countryName, regionName string,
) {
	if location == nil || location.CountryRegionSpot == "" {
		return
	}

	parts := strings.Split(location.CountryRegionSpot, "/")
	if len(parts) < 3 {
		return
	}
	spotName := parts[2]

	imageKey := fmt.Sprintf(
		"spot-images/%s_%s_%s.jpg",
		strings.ReplaceAll(countryName, " ", ""),
		strings.ReplaceAll(regionName, " ", ""),
		strings.ReplaceAll(spotName, " ", ""),
	)

	imageData, err := s.media.Download(ctx, imageKey)
	if err != nil {
		fmt.Printf("Failed to fetch image for %s: %v\n", imageKey, err)
		return
	}

	location.ImageString = base64.StdEncoding.EncodeToString(imageData)
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
