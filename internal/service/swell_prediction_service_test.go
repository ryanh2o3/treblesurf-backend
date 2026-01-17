package service

import (
	"context"
	"testing"
	"time"

	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestNewSwellPredictionService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewSwellPredictionService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestSwellPredictionService_GetSpotSwellPrediction(t *testing.T) {
	ctx := context.Background()
	expected := []map[string]interface{}{
		{"spot_id": "Ireland#Donegal#Bundoran", "wave_height": 2.5},
		{"spot_id": "Ireland#Donegal#Bundoran", "wave_height": 3.0},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionsFn: func(_ context.Context, spotID string, start time.Time, limit int) ([]map[string]interface{}, error) {
			if spotID != "Ireland#Donegal#Bundoran" {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpotSwellPrediction(ctx, "Bundoran", "Donegal", "Ireland")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d predictions, got %d", len(expected), len(got))
	}
}

func TestSwellPredictionService_GetListSpotsSwellPrediction(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.SwellPredictionRepo{
		GetListSpotsPredictionsFn: func(_ context.Context, spotIDs []string, start time.Time, limit int) ([][]map[string]interface{}, error) {
			if len(spotIDs) != 2 {
				t.Fatalf("expected 2 spot IDs, got %d", len(spotIDs))
			}
			return [][]map[string]interface{}{
				{{"spot_id": spotIDs[0]}},
				{{"spot_id": spotIDs[1]}},
			}, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	spots := []string{"Bundoran", "Rossnowlagh"}
	got, err := service.GetListSpotsSwellPrediction(ctx, spots, "Donegal", "Ireland")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(spots) {
		t.Fatalf("expected %d results, got %d", len(spots), len(got))
	}
}

func TestSwellPredictionService_GetRegionSwellPrediction(t *testing.T) {
	ctx := context.Background()
	expected := []map[string]interface{}{
		{"spot_id": "Ireland#Donegal#Bundoran"},
		{"spot_id": "Ireland#Donegal#Rossnowlagh"},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetRegionPredictionsFn: func(_ context.Context, country, region string, start time.Time, perSpotLimit int) ([]map[string]interface{}, error) {
			if country != "Ireland" || region != "Donegal" {
				t.Fatalf("unexpected args: %s %s", country, region)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegionSwellPrediction(ctx, "Donegal", "Ireland")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d predictions, got %d", len(expected), len(got))
	}
}

func TestSwellPredictionService_GetSpotSwellPredictionRange(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	end := start.Add(24 * time.Hour)
	expected := []map[string]interface{}{
		{"spot_id": "Ireland#Donegal#Bundoran", "timestamp": start},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionRangeFn: func(_ context.Context, spotID string, s, e time.Time) ([]map[string]interface{}, error) {
			if spotID != "Ireland#Donegal#Bundoran" {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpotSwellPredictionRange(ctx, "Bundoran", "Donegal", "Ireland", start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d predictions, got %d", len(expected), len(got))
	}
}

func TestSwellPredictionService_GetRecentSwellPredictions(t *testing.T) {
	ctx := context.Background()
	expected := []map[string]interface{}{
		{"spot_id": "Ireland#Donegal#Bundoran"},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetRecentPredictionsFn: func(_ context.Context, cutoff time.Time, perSpotLimit int) ([]map[string]interface{}, error) {
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRecentSwellPredictions(ctx, 24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d predictions, got %d", len(expected), len(got))
	}
}

func TestSwellPredictionService_GetClosestAIPredictionForSpot(t *testing.T) {
	ctx := context.Background()
	expected := map[string]interface{}{
		"spot_id":     "Ireland#Donegal#Bundoran",
		"wave_height": 2.5,
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetClosestPredictionFn: func(_ context.Context, spotID string, now time.Time) (map[string]interface{}, error) {
			if spotID != "Ireland#Donegal#Bundoran" {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetClosestAIPredictionForSpot(ctx, "Bundoran", "Donegal", "Ireland")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["spot_id"] != expected["spot_id"] {
		t.Fatalf("expected spot_id %v, got %v", expected["spot_id"], got["spot_id"])
	}
}
