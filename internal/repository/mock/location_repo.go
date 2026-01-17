package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.LocationRepository = (*LocationRepo)(nil)

type LocationRepo struct {
	GetRegionsFn     func(ctx context.Context, country string) ([]string, error)
	GetSpotsFn       func(ctx context.Context, country, region string) ([]*model.LocationInfo, error)
	GetLocationInfoFn func(ctx context.Context, country, region, spot string) (*model.LocationInfo, error)
	GetCoordinatesFn func(ctx context.Context, country, region, spot string) (lat, lon float64, err error)
}

func (m *LocationRepo) GetRegions(ctx context.Context, country string) ([]string, error) {
	if m.GetRegionsFn != nil {
		return m.GetRegionsFn(ctx, country)
	}
	return nil, nil
}

func (m *LocationRepo) GetSpots(ctx context.Context, country, region string) ([]*model.LocationInfo, error) {
	if m.GetSpotsFn != nil {
		return m.GetSpotsFn(ctx, country, region)
	}
	return nil, nil
}

func (m *LocationRepo) GetLocationInfo(ctx context.Context, country, region, spot string) (*model.LocationInfo, error) {
	if m.GetLocationInfoFn != nil {
		return m.GetLocationInfoFn(ctx, country, region, spot)
	}
	return nil, model.ErrLocationNotFound
}

func (m *LocationRepo) GetCoordinates(ctx context.Context, country, region, spot string) (float64, float64, error) {
	if m.GetCoordinatesFn != nil {
		return m.GetCoordinatesFn(ctx, country, region, spot)
	}
	return 0, 0, model.ErrLocationNotFound
}
