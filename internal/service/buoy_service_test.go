package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

const testBuoyName = "M4"

func TestNewBuoyService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewBuoyService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestBuoyService_GetLiveData(t *testing.T) {
	t.Run("returns data for valid buoy", func(t *testing.T) {
		ctx := context.Background()
		expected := &model.BuoyData{
			BuoyName:   testBuoyName,
			WaveHeight: 2.5,
			WavePeriod: 12.0,
		}
		repo := &mockrepo.BuoyRepo{
			GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
				if buoyName != testBuoyName {
					t.Fatalf("unexpected buoy name: %s", buoyName)
				}
				return expected, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetLiveData(ctx, testBuoyName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	})

	t.Run("returns error for empty buoy name", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		_, err := service.GetLiveData(context.Background(), "")
		if err == nil {
			t.Fatalf("expected error for empty buoy name")
		}
	})
}

func TestBuoyService_GetDataRange(t *testing.T) {
	t.Run("returns data within range", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now().Add(-24 * time.Hour)
		end := time.Now()
		expected := []*model.BuoyData{
			{BuoyName: testBuoyName, WaveHeight: 2.0},
			{BuoyName: testBuoyName, WaveHeight: 2.5},
		}
		repo := &mockrepo.BuoyRepo{
			GetDataRangeFn: func(_ context.Context, buoyName string, _, _ time.Time) ([]*model.BuoyData, error) {
				if buoyName != testBuoyName {
					t.Fatalf("unexpected buoy name: %s", buoyName)
				}
				return expected, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetDataRange(ctx, testBuoyName, start, end)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(expected) {
			t.Fatalf("expected %d items, got %d", len(expected), len(got))
		}
	})

	t.Run("returns error for end before start", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		start := time.Now()
		end := time.Now().Add(-24 * time.Hour) // End before start

		_, err := service.GetDataRange(context.Background(), testBuoyName, start, end)
		if err == nil {
			t.Fatalf("expected error for end time before start time")
		}
	})
}

func TestBuoyService_GetLocations(t *testing.T) {
	ctx := context.Background()
	expected := map[string]*model.BuoyLocation{
		testBuoyName: {Name: testBuoyName, Region: "Atlantic", Latitude: 51.0, Longitude: -10.0},
		"M5": {Name: "M5", Region: "Celtic", Latitude: 51.5, Longitude: -9.5},
	}
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return expected, nil
		},
	}

	service, err := NewBuoyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetLocations(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d locations, got %d", len(expected), len(got))
	}
}

func TestBuoyService_GetRegionBuoys(t *testing.T) {
	ctx := context.Background()
	locations := map[string]*model.BuoyLocation{
		testBuoyName: {Name: testBuoyName, Region: "Atlantic", Latitude: 51.0, Longitude: -10.0},
		"M5":         {Name: "M5", Region: "Atlantic", Latitude: 51.5, Longitude: -9.5},
		"Blackstone": {Name: "Blackstone", Region: "Celtic", Latitude: 52.0, Longitude: -8.0},
	}
	repo := &mockrepo.BuoyRepo{
		GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
			return locations, nil
		},
	}

	service, err := NewBuoyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegionBuoys(ctx, "Atlantic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 buoys for Atlantic, got %d", len(got))
	}
}

func TestBuoyService_GetLast24HoursData(t *testing.T) {
	ctx := context.Background()
	expected := []*model.BuoyData{
		{BuoyName: testBuoyName, WaveHeight: 2.0},
		{BuoyName: testBuoyName, WaveHeight: 2.5},
	}
	repo := &mockrepo.BuoyRepo{
		GetDataRangeFn: func(_ context.Context, _ string, start, end time.Time) ([]*model.BuoyData, error) {
			// Verify time range is approximately 24 hours
			duration := end.Sub(start)
			if duration < 23*time.Hour || duration > 25*time.Hour {
				t.Fatalf("expected ~24 hour range, got %v", duration)
			}
			return expected, nil
		},
	}

	service, err := NewBuoyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetLast24HoursData(ctx, testBuoyName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(got))
	}
}

func TestBuoyService_DefaultBuoys(t *testing.T) {
	repo := &mockrepo.BuoyRepo{}
	service, err := NewBuoyService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	buoys := service.DefaultBuoys()
	if len(buoys) == 0 {
		t.Fatalf("expected default buoys to be returned")
	}
	// Should include M4
	found := false
	for _, b := range buoys {
		if b == testBuoyName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s to be in default buoys", testBuoyName)
	}
}

func TestBuoyService_GetDataAtTime(t *testing.T) {
	t.Run("returns data at specific time", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		expected := &model.BuoyData{
			BuoyName:   testBuoyName,
			WaveHeight: 2.5,
			Timestamp:  now,
		}
		repo := &mockrepo.BuoyRepo{
			GetDataAtTimeFn: func(_ context.Context, buoyName string, _ time.Time) (*model.BuoyData, error) {
				if buoyName != testBuoyName {
					t.Fatalf("unexpected buoy name: %s", buoyName)
				}
				return expected, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetDataAtTime(ctx, testBuoyName, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	})

	t.Run("returns error for empty buoy name", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		_, err := service.GetDataAtTime(context.Background(), "", time.Now())
		if err == nil {
			t.Fatalf("expected error for empty buoy name")
		}
	})
}

func TestBuoyService_GetMultipleBuoysData(t *testing.T) {
	t.Run("returns data for multiple buoys", func(t *testing.T) {
		ctx := context.Background()
		buoyNames := []string{testBuoyName, "M5", "Blackstones"}
		repo := &mockrepo.BuoyRepo{
			GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
				return &model.BuoyData{
					BuoyName:   buoyName,
					WaveHeight: 2.5,
					Timestamp:  time.Now(),
				}, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetMultipleBuoysData(ctx, buoyNames)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(buoyNames) {
			t.Fatalf("expected %d items, got %d", len(buoyNames), len(got))
		}
	})

	t.Run("returns empty for empty buoy names", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		got, err := service.GetMultipleBuoysData(context.Background(), []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty or nil, got %d items", len(got))
		}
	})

	t.Run("skips buoys with errors", func(t *testing.T) {
		ctx := context.Background()
		buoyNames := []string{testBuoyName, "M5"}
		callCount := 0
		repo := &mockrepo.BuoyRepo{
			GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
				callCount++
				if buoyName == testBuoyName {
					return &model.BuoyData{BuoyName: testBuoyName, WaveHeight: 2.5}, nil
				}
				// M5 returns error - should be skipped
				return nil, model.ErrBuoyDataNotFound
			},
		}

		service, _ := NewBuoyService(repo)

		got, err := service.GetMultipleBuoysData(ctx, buoyNames)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 item (%s), got %d", testBuoyName, len(got))
		}
		if got[0].BuoyName != testBuoyName {
			t.Fatalf("expected %s, got %s", testBuoyName, got[0].BuoyName)
		}
	})
}

func TestBuoyService_GetLocationByName(t *testing.T) {
	t.Run("returns location for existing buoy", func(t *testing.T) {
		ctx := context.Background()
		expected := &model.BuoyLocation{
			Name:      testBuoyName,
			Region:    "Atlantic",
			Country:   "Ireland",
			Latitude:  51.0,
			Longitude: -10.0,
		}
		locations := map[string]*model.BuoyLocation{
			testBuoyName: expected,
		}
		repo := &mockrepo.BuoyRepo{
			GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
				return locations, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetLocationByName(ctx, testBuoyName)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != expected {
			t.Fatalf("expected %+v, got %+v", expected, got)
		}
	})

	t.Run("returns error for non-existent buoy", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.BuoyRepo{
			GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
				return map[string]*model.BuoyLocation{}, nil
			},
		}

		service, _ := NewBuoyService(repo)

		_, err := service.GetLocationByName(ctx, "NonExistent")
		if err == nil {
			t.Fatalf("expected error for non-existent buoy")
		}
		if err != model.ErrBuoyDataNotFound {
			t.Fatalf("expected ErrBuoyDataNotFound, got %v", err)
		}
	})

	t.Run("returns error for empty buoy name", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		_, err := service.GetLocationByName(context.Background(), "")
		if err == nil {
			t.Fatalf("expected error for empty buoy name")
		}
	})
}

func TestBuoyService_GetBatchDataRanges(t *testing.T) {
	t.Run("returns data for batch requests", func(t *testing.T) {
		ctx := context.Background()
		requests := []repository.BuoyDataRequest{
			{BuoyName: testBuoyName, Start: time.Now().Add(-24 * time.Hour), End: time.Now()},
			{BuoyName: "M5", Start: time.Now().Add(-24 * time.Hour), End: time.Now()},
		}
		expected := map[string][]*model.BuoyData{
			testBuoyName: {{BuoyName: testBuoyName, WaveHeight: 2.5}},
			"M5": {{BuoyName: "M5", WaveHeight: 3.0}},
		}
		repo := &mockrepo.BuoyRepo{
			GetBatchDataRangesFn: func(_ context.Context, reqs []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
				if len(reqs) != len(requests) {
					t.Fatalf("expected %d requests, got %d", len(requests), len(reqs))
				}
				return expected, nil
			},
		}

		service, err := NewBuoyService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		got, err := service.GetBatchDataRanges(ctx, requests)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(expected) {
			t.Fatalf("expected %d buoys, got %d", len(expected), len(got))
		}
	})

	t.Run("returns empty map for empty requests", func(t *testing.T) {
		repo := &mockrepo.BuoyRepo{}
		service, _ := NewBuoyService(repo)

		got, err := service.GetBatchDataRanges(context.Background(), []repository.BuoyDataRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty map, got %d items", len(got))
		}
	})
}
