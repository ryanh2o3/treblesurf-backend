package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

const testSpotID = "Ireland_Donegal_Bundoran"

func TestNewSnapshotService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewSnapshotService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestSnapshotService_StoreSnapshot(t *testing.T) {
	t.Run("stores valid snapshot", func(t *testing.T) {
		ctx := context.Background()
		var saved *model.SpotSnapshot
		repo := &mockrepo.SnapshotRepo{
			SaveFn: func(_ context.Context, snapshot *model.SpotSnapshot) error {
				saved = snapshot
				return nil
			},
		}

		service, err := NewSnapshotService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		timestamp := time.Now()
		snapshot, err := service.StoreSnapshot(ctx, testSpotID, "snapshots/test.jpg", timestamp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if snapshot == nil {
			t.Fatalf("expected snapshot to be returned")
		}
		if saved == nil {
			t.Fatalf("expected snapshot to be saved")
		}
		if snapshot.SpotID != testSpotID {
			t.Fatalf("unexpected spot ID: %s", snapshot.SpotID)
		}
		if snapshot.ImageKey != "snapshots/test.jpg" {
			t.Fatalf("unexpected image key: %s", snapshot.ImageKey)
		}
	})

	t.Run("returns error for empty spot ID", func(t *testing.T) {
		repo := &mockrepo.SnapshotRepo{}
		service, _ := NewSnapshotService(repo)

		_, err := service.StoreSnapshot(context.Background(), "", "key.jpg", time.Now())
		if err == nil {
			t.Fatalf("expected error for empty spot ID")
		}
	})

	t.Run("returns error for empty image key", func(t *testing.T) {
		repo := &mockrepo.SnapshotRepo{}
		service, _ := NewSnapshotService(repo)

		_, err := service.StoreSnapshot(context.Background(), "spot-id", "", time.Now())
		if err == nil {
			t.Fatalf("expected error for empty image key")
		}
	})
}

func TestSnapshotService_GetLatestSnapshot(t *testing.T) {
	t.Run("returns latest snapshot", func(t *testing.T) {
		ctx := context.Background()
		expected := &model.SpotSnapshot{
			SpotID:   testSpotID,
			ImageKey: "snapshots/latest.jpg",
		}
		repo := &mockrepo.SnapshotRepo{
			GetLatestBySpotFn: func(_ context.Context, spotID string) (*model.SpotSnapshot, error) {
				if spotID != testSpotID {
					t.Fatalf("unexpected spot ID: %s", spotID)
				}
				return expected, nil
			},
		}

		service, err := NewSnapshotService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		snapshot, err := service.GetLatestSnapshot(ctx, testSpotID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if snapshot != expected {
			t.Fatalf("expected %+v, got %+v", expected, snapshot)
		}
	})

	t.Run("returns error for empty spot ID", func(t *testing.T) {
		repo := &mockrepo.SnapshotRepo{}
		service, _ := NewSnapshotService(repo)

		_, err := service.GetLatestSnapshot(context.Background(), "")
		if err == nil {
			t.Fatalf("expected error for empty spot ID")
		}
	})
}
