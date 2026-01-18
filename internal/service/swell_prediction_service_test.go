package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

const (
	testSwellSpotID     = "Ireland#Donegal#Bundoran"
	swellTestCountry    = "Ireland"
	swellTestRegion     = "Donegal"
	swellTestSpot       = "Bundoran"
)

func TestNewSwellPredictionService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewSwellPredictionService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestSwellPredictionService_GetSpotSwellPrediction(t *testing.T) {
	ctx := context.Background()
	expected := []model.SwellPrediction{
		{SpotID: testSwellSpotID, Data: map[string]interface{}{"wave_height": 2.5}},
		{SpotID: testSwellSpotID, Data: map[string]interface{}{"wave_height": 3.0}},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionsFn: func(_ context.Context, spotID string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			if spotID != testSwellSpotID {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpotSwellPrediction(ctx, swellTestSpot, swellTestRegion, swellTestCountry)
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
		GetListSpotsPredictionsFn: func(_ context.Context, spotIDs []string, _ time.Time, _ int) ([][]model.SwellPrediction, error) {
			if len(spotIDs) != 2 {
				t.Fatalf("expected 2 spot IDs, got %d", len(spotIDs))
			}
			return [][]model.SwellPrediction{
				{{SpotID: spotIDs[0], Data: map[string]interface{}{}}},
				{{SpotID: spotIDs[1], Data: map[string]interface{}{}}},
			}, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	spots := []string{"Bundoran", "Rossnowlagh"}
	got, err := service.GetListSpotsSwellPrediction(ctx, spots, swellTestRegion, swellTestCountry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(spots) {
		t.Fatalf("expected %d results, got %d", len(spots), len(got))
	}
}

func TestSwellPredictionService_GetRegionSwellPrediction(t *testing.T) {
	ctx := context.Background()
	expected := []model.SwellPrediction{
		{SpotID: testSwellSpotID, Data: map[string]interface{}{}},
		{SpotID: "Ireland#Donegal#Rossnowlagh", Data: map[string]interface{}{}},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetRegionPredictionsFn: func(_ context.Context, country, region string, _ time.Time, _ int) ([]model.SwellPrediction, error) {
			if country != swellTestCountry || region != swellTestRegion {
				t.Fatalf("unexpected args: %s %s", country, region)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegionSwellPrediction(ctx, swellTestRegion, swellTestCountry)
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
	expected := []model.SwellPrediction{
		{
			SpotID: testSwellSpotID,
			Data:   map[string]interface{}{"timestamp": start},
		},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionRangeFn: func(_ context.Context, spotID string, _, _ time.Time) ([]model.SwellPrediction, error) {
			if spotID != testSwellSpotID {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpotSwellPredictionRange(ctx, swellTestSpot, swellTestRegion, swellTestCountry, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d predictions, got %d", len(expected), len(got))
	}
}

func TestSwellPredictionService_GetRecentSwellPredictions(t *testing.T) {
	ctx := context.Background()
	expected := []model.SwellPrediction{
		{SpotID: testSwellSpotID, Data: map[string]interface{}{}},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetRecentPredictionsFn: func(_ context.Context, _ time.Time, _ int) ([]model.SwellPrediction, error) {
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
	expected := &model.SwellPrediction{
		SpotID: testSwellSpotID,
		Data:   map[string]interface{}{"wave_height": 2.5},
	}
	repo := &mockrepo.SwellPredictionRepo{
		GetClosestPredictionFn: func(_ context.Context, spotID string, _ time.Time) (*model.SwellPrediction, error) {
			if spotID != testSwellSpotID {
				t.Fatalf("unexpected spot ID: %s", spotID)
			}
			return expected, nil
		},
	}

	service, err := NewSwellPredictionService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetClosestAIPredictionForSpot(ctx, swellTestSpot, swellTestRegion, swellTestCountry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SpotID != expected.SpotID {
		t.Fatalf("expected spot_id %v, got %v", expected.SpotID, got.SpotID)
	}
}
