package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.SnapshotRepository = (*SnapshotRepo)(nil)

type SnapshotRepo struct {
	SaveFn            func(ctx context.Context, snapshot *model.SpotSnapshot) error
	GetLatestBySpotFn func(ctx context.Context, spotID string) (*model.SpotSnapshot, error)
}

func (m *SnapshotRepo) Save(ctx context.Context, snapshot *model.SpotSnapshot) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, snapshot)
	}
	return nil
}

func (m *SnapshotRepo) GetLatestBySpot(ctx context.Context, spotID string) (*model.SpotSnapshot, error) {
	if m.GetLatestBySpotFn != nil {
		return m.GetLatestBySpotFn(ctx, spotID)
	}
	return nil, repository.ErrNotFound
}
