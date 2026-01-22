// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// BuoyService provides business logic for buoy data operations.
type BuoyService struct {
	buoys repository.BuoyRepository
}

// NewBuoyService creates a new BuoyService with the given repository.
// Returns an error if the repository is nil.
func NewBuoyService(buoys repository.BuoyRepository) (*BuoyService, error) {
	if buoys == nil {
		return nil, fmt.Errorf("buoy repository is required")
	}
	return &BuoyService{buoys: buoys}, nil
}

// GetLiveData retrieves the most recent data for a specific buoy.
func (s *BuoyService) GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error) {
	if buoyName == "" {
		return nil, fmt.Errorf("buoy name is required")
	}
	return s.buoys.GetLiveData(ctx, buoyName)
}

// GetDataAtTime retrieves buoy data at a specific time.
func (s *BuoyService) GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error) {
	if buoyName == "" {
		return nil, fmt.Errorf("buoy name is required")
	}
	return s.buoys.GetDataAtTime(ctx, buoyName, t)
}

// GetDataRange retrieves buoy data within a time range.
func (s *BuoyService) GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error) {
	if buoyName == "" {
		return nil, fmt.Errorf("buoy name is required")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}
	return s.buoys.GetDataRange(ctx, buoyName, start, end)
}

// GetLast24HoursData retrieves buoy data from the last 24 hours.
func (s *BuoyService) GetLast24HoursData(ctx context.Context, buoyName string) ([]*model.BuoyData, error) {
	if buoyName == "" {
		return nil, fmt.Errorf("buoy name is required")
	}
	endTime := time.Now().UTC()
	startTime := endTime.Add(-24 * time.Hour)
	return s.buoys.GetDataRange(ctx, buoyName, startTime, endTime)
}

// GetMultipleBuoysData retrieves live data for multiple buoys.
func (s *BuoyService) GetMultipleBuoysData(ctx context.Context, buoyNames []string) ([]*model.BuoyData, error) {
	if len(buoyNames) == 0 {
		return nil, nil
	}

	results := make([]*model.BuoyData, 0, len(buoyNames))
	for _, name := range buoyNames {
		data, err := s.buoys.GetLiveData(ctx, name)
		if err != nil {
			continue // Skip buoys with errors
		}
		if data != nil {
			results = append(results, data)
		}
	}
	return results, nil
}

// GetLocations retrieves all buoy location information.
func (s *BuoyService) GetLocations(ctx context.Context) (map[string]*model.BuoyLocation, error) {
	return s.buoys.GetLocations(ctx)
}

// GetLocationByName retrieves location information for a specific buoy.
func (s *BuoyService) GetLocationByName(ctx context.Context, buoyName string) (*model.BuoyLocation, error) {
	if buoyName == "" {
		return nil, fmt.Errorf("buoy name is required")
	}

	locations, err := s.buoys.GetLocations(ctx)
	if err != nil {
		return nil, err
	}

	if location, ok := locations[buoyName]; ok {
		return location, nil
	}
	return nil, model.ErrBuoyDataNotFound
}

// GetRegionBuoys retrieves all buoys for a specific region.
func (s *BuoyService) GetRegionBuoys(ctx context.Context, regionName string) ([]model.Buoy, error) {
	locations, err := s.buoys.GetLocations(ctx)
	if err != nil {
		return nil, err
	}

	buoys := make([]model.Buoy, 0)
	for name, location := range locations {
		if location == nil {
			continue
		}
		if regionName != "" && location.Region != regionName {
			continue
		}
		buoys = append(buoys, model.Buoy{
			RegionBuoy: fmt.Sprintf("%s_%s", location.Region, name),
			Name:       name,
			Latitude:   location.Latitude,
			Longitude:  location.Longitude,
		})
	}

	return buoys, nil
}

// GetBatchDataRanges retrieves data for multiple buoys in batch.
func (s *BuoyService) GetBatchDataRanges(
	ctx context.Context,
	requests []repository.BuoyDataRequest,
) (map[string][]*model.BuoyData, error) {
	if len(requests) == 0 {
		return make(map[string][]*model.BuoyData), nil
	}
	return s.buoys.GetBatchDataRanges(ctx, requests)
}

// DefaultBuoys returns the list of default buoys to display.
func (s *BuoyService) DefaultBuoys() []string {
	return []string{"M4", "Blackstones", "West Hebrides", "M2", "M3", "M5", "M6"}
}
