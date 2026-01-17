package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

type SnapshotService struct {
	snapshots repository.SnapshotRepository
}

func NewSnapshotService(snapshots repository.SnapshotRepository) *SnapshotService {
	return &SnapshotService{snapshots: snapshots}
}

func (s *SnapshotService) StoreSnapshot(
	ctx context.Context,
	spotID string,
	imageKey string,
	timestamp time.Time,
) (*model.SpotSnapshot, error) {
	if spotID == "" {
		return nil, fmt.Errorf("spot id is required")
	}
	if imageKey == "" {
		return nil, fmt.Errorf("image key is required")
	}

	snapshot := &model.SpotSnapshot{
		SpotID:     spotID,
		ImageKey:   imageKey,
		Timestamp:  timestamp,
		UploadedAt: time.Now(),
	}

	if err := s.snapshots.Save(ctx, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func (s *SnapshotService) GetLatestSnapshot(ctx context.Context, spotID string) (*model.SpotSnapshot, error) {
	if spotID == "" {
		return nil, fmt.Errorf("spot id is required")
	}

	return s.snapshots.GetLatestBySpot(ctx, spotID)
}
