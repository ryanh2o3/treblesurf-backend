// Package service provides business logic services for the application.
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// LocationService provides business logic for location operations.
type LocationService struct {
	locations repository.LocationRepository
	media     repository.MediaRepository
}

// NewLocationService creates a new LocationService with the given repositories.
// Returns an error if any required repository is nil.
func NewLocationService(
	locations repository.LocationRepository,
	media repository.MediaRepository,
) (*LocationService, error) {
	if locations == nil {
		return nil, fmt.Errorf("location repository is required")
	}
	if media == nil {
		return nil, fmt.Errorf("media repository is required")
	}
	return &LocationService{
		locations: locations,
		media:     media,
	}, nil
}

func (s *LocationService) GetRegions(ctx context.Context, countryName string) ([]string, error) {
	return s.locations.GetRegions(ctx, countryName)
}

func (s *LocationService) GetSpots(ctx context.Context, countryName, regionName string) ([]model.LocationInfo, error) {
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

func (s *LocationService) GetLocationInfo(
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

func (s *LocationService) GetCoordinates(
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
		slog.Debug("failed to fetch image", slog.String("key", imageKey), slog.Any("error", err))
		return
	}

	location.ImageString = base64.StdEncoding.EncodeToString(imageData)
}
