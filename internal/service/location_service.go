// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// LocationService provides business logic for location operations.
type LocationService struct {
	locations         repository.LocationRepository
	spotImagesBaseURL string
}

// NewLocationService creates a new LocationService with the given repositories.
// spotImagesBaseURL is the origin for JPEG keys under spot-images/ (no trailing slash), e.g. a CloudFront URL or S3 virtual-hosted base.
// Returns an error if the location repository is nil.
func NewLocationService(
	locations repository.LocationRepository,
	spotImagesBaseURL string,
) (*LocationService, error) {
	if locations == nil {
		return nil, fmt.Errorf("location repository is required")
	}
	return &LocationService{
		locations:         locations,
		spotImagesBaseURL: strings.TrimSuffix(strings.TrimSpace(spotImagesBaseURL), "/"),
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
		location.ImageString = ""
		s.attachSpotImageURL(&location, countryName, regionName)
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

	location.ImageString = ""
	s.attachSpotImageURL(location, countryName, regionName)
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

func (s *LocationService) attachSpotImageURL(
	location *model.LocationInfo,
	countryName, regionName string,
) {
	if location == nil || s.spotImagesBaseURL == "" || location.CountryRegionSpot == "" {
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

	location.ImageURL = s.spotImagesBaseURL + "/" + imageKey
}
