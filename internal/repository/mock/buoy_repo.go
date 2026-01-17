package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.BuoyRepository = (*BuoyRepo)(nil)

type BuoyRepo struct {
	GetLiveDataFn  func(ctx context.Context, buoyName string) (*model.BuoyData, error)
	GetDataAtTimeFn func(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error)
	GetDataRangeFn func(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error)
	GetLocationsFn func(ctx context.Context) (map[string]*model.BuoyLocation, error)
}

func (m *BuoyRepo) GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error) {
	if m.GetLiveDataFn != nil {
		return m.GetLiveDataFn(ctx, buoyName)
	}
	return nil, nil
}

func (m *BuoyRepo) GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error) {
	if m.GetDataAtTimeFn != nil {
		return m.GetDataAtTimeFn(ctx, buoyName, t)
	}
	return nil, nil
}

func (m *BuoyRepo) GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error) {
	if m.GetDataRangeFn != nil {
		return m.GetDataRangeFn(ctx, buoyName, start, end)
	}
	return nil, nil
}

func (m *BuoyRepo) GetLocations(ctx context.Context) (map[string]*model.BuoyLocation, error) {
	if m.GetLocationsFn != nil {
		return m.GetLocationsFn(ctx)
	}
	return map[string]*model.BuoyLocation{}, nil
}
