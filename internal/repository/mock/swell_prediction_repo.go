package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.SwellPredictionRepository = (*SwellPredictionRepo)(nil)

type SwellPredictionRepo struct {
	GetSpotPredictionsFn      func(ctx context.Context, spotID string, start time.Time, limit int) ([]model.SwellPrediction, error)
	GetListSpotsPredictionsFn func(ctx context.Context, spotIDs []string, start time.Time, limit int) ([][]model.SwellPrediction, error)
	GetRegionPredictionsFn    func(
		ctx context.Context,
		country, region string,
		start time.Time,
		perSpotLimit int,
	) ([]model.SwellPrediction, error)
	GetSpotPredictionRangeFn func(ctx context.Context, spotID string, start, end time.Time) ([]model.SwellPrediction, error)
	GetRecentPredictionsFn   func(ctx context.Context, cutoff time.Time, perSpotLimit int) ([]model.SwellPrediction, error)
	GetClosestPredictionFn   func(ctx context.Context, spotID string, now time.Time) (*model.SwellPrediction, error)
}

func (m *SwellPredictionRepo) GetSpotPredictions(
	ctx context.Context,
	spotID string,
	start time.Time,
	limit int,
) ([]model.SwellPrediction, error) {
	if m.GetSpotPredictionsFn != nil {
		return m.GetSpotPredictionsFn(ctx, spotID, start, limit)
	}
	return []model.SwellPrediction{}, nil
}

func (m *SwellPredictionRepo) GetListSpotsPredictions(
	ctx context.Context,
	spotIDs []string,
	start time.Time,
	limit int,
) ([][]model.SwellPrediction, error) {
	if m.GetListSpotsPredictionsFn != nil {
		return m.GetListSpotsPredictionsFn(ctx, spotIDs, start, limit)
	}
	return [][]model.SwellPrediction{}, nil
}

func (m *SwellPredictionRepo) GetRegionPredictions(
	ctx context.Context,
	country, region string,
	start time.Time,
	perSpotLimit int,
) ([]model.SwellPrediction, error) {
	if m.GetRegionPredictionsFn != nil {
		return m.GetRegionPredictionsFn(ctx, country, region, start, perSpotLimit)
	}
	return []model.SwellPrediction{}, nil
}

func (m *SwellPredictionRepo) GetSpotPredictionRange(
	ctx context.Context,
	spotID string,
	start, end time.Time,
) ([]model.SwellPrediction, error) {
	if m.GetSpotPredictionRangeFn != nil {
		return m.GetSpotPredictionRangeFn(ctx, spotID, start, end)
	}
	return []model.SwellPrediction{}, nil
}

func (m *SwellPredictionRepo) GetRecentPredictions(
	ctx context.Context,
	cutoff time.Time,
	perSpotLimit int,
) ([]model.SwellPrediction, error) {
	if m.GetRecentPredictionsFn != nil {
		return m.GetRecentPredictionsFn(ctx, cutoff, perSpotLimit)
	}
	return []model.SwellPrediction{}, nil
}

func (m *SwellPredictionRepo) GetClosestPrediction(
	ctx context.Context,
	spotID string,
	now time.Time,
) (*model.SwellPrediction, error) {
	if m.GetClosestPredictionFn != nil {
		return m.GetClosestPredictionFn(ctx, spotID, now)
	}
	return nil, repository.ErrNotFound
}
