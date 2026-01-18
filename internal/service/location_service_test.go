package service

import (
	"context"
	"testing"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

const (
	locationTestCountry = "Ireland"
	locationTestRegion  = "Donegal"
	locationTestSpot    = "Bundoran"
)

func TestNewLocationService_NilRepositories_ReturnsError(t *testing.T) {
	t.Run("nil location repository", func(t *testing.T) {
		_, err := NewLocationService(nil, &mockrepo.MediaRepo{})
		if err == nil {
			t.Fatalf("expected error for nil location repository")
		}
	})

	t.Run("nil media repository", func(t *testing.T) {
		_, err := NewLocationService(&mockrepo.LocationRepo{}, nil)
		if err == nil {
			t.Fatalf("expected error for nil media repository")
		}
	})
}

func TestLocationService_GetRegions(t *testing.T) {
	ctx := context.Background()
	expected := []string{"Donegal", "Sligo", "Clare"}
	repo := &mockrepo.LocationRepo{
		GetRegionsFn: func(_ context.Context, country string) ([]string, error) {
			if country != locationTestCountry {
				t.Fatalf("unexpected country: %s", country)
			}
			return expected, nil
		},
	}

	service, err := NewLocationService(repo, &mockrepo.MediaRepo{})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetRegions(ctx, locationTestCountry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d regions, got %d", len(expected), len(got))
	}
}

func TestLocationService_GetSpots(t *testing.T) {
	ctx := context.Background()
	expected := []*model.LocationInfo{
		{CountryRegionSpot: locationTestCountry + "/" + locationTestRegion + "/" + locationTestSpot},
		{CountryRegionSpot: locationTestCountry + "/" + locationTestRegion + "/Rossnowlagh"},
	}
	repo := &mockrepo.LocationRepo{
		GetSpotsFn: func(_ context.Context, country, region string) ([]*model.LocationInfo, error) {
			if country != locationTestCountry || region != locationTestRegion {
				t.Fatalf("unexpected args: %s %s", country, region)
			}
			return expected, nil
		},
	}

	service, err := NewLocationService(repo, &mockrepo.MediaRepo{})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpots(ctx, locationTestCountry, locationTestRegion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("expected %d spots, got %d", len(expected), len(got))
	}
}

func TestLocationService_GetLocationInfo(t *testing.T) {
	ctx := context.Background()
	expected := &model.LocationInfo{
		CountryRegionSpot: locationTestCountry + "/" + locationTestRegion + "/" + locationTestSpot,
		Latitude:          54.5,
		Longitude:         -8.3,
	}
	repo := &mockrepo.LocationRepo{
		GetLocationInfoFn: func(_ context.Context, country, region, spot string) (*model.LocationInfo, error) {
			if country != locationTestCountry || region != locationTestRegion || spot != locationTestSpot {
				t.Fatalf("unexpected args: %s %s %s", country, region, spot)
			}
			return expected, nil
		},
	}

	service, err := NewLocationService(repo, &mockrepo.MediaRepo{})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetLocationInfo(ctx, locationTestCountry, locationTestRegion, locationTestSpot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.CountryRegionSpot != expected.CountryRegionSpot {
		t.Fatalf("expected %s, got %s", expected.CountryRegionSpot, got.CountryRegionSpot)
	}
}

func TestLocationService_GetCoordinates(t *testing.T) {
	ctx := context.Background()
	expectedLat := 54.5
	expectedLon := -8.3
	repo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, _, _, _ string) (float64, float64, error) {
			return expectedLat, expectedLon, nil
		},
	}

	service, err := NewLocationService(repo, &mockrepo.MediaRepo{})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetCoordinates(ctx, locationTestCountry, locationTestRegion, locationTestSpot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(got))
	}
	if got[0] != expectedLat || got[1] != expectedLon {
		t.Fatalf("expected [%f, %f], got %v", expectedLat, expectedLon, got)
	}
}

func TestLocationService_GetCoordinates_Error(t *testing.T) {
	ctx := context.Background()
	repo := &mockrepo.LocationRepo{
		GetCoordinatesFn: func(_ context.Context, _, _, _ string) (float64, float64, error) {
			return 0, 0, model.ErrLocationNotFound
		},
	}

	service, err := NewLocationService(repo, &mockrepo.MediaRepo{})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	_, err = service.GetCoordinates(ctx, locationTestCountry, locationTestRegion, locationTestSpot)
	if err == nil {
		t.Fatalf("expected error for not found location")
	}
}

func TestLocationService_PopulateImage(t *testing.T) {
	t.Run("handles nil location", func(t *testing.T) {
		service := &LocationService{}
		service.populateImage(context.Background(), nil, "Ireland", "Donegal")
		// Should not panic
	})

	t.Run("handles empty CountryRegionSpot", func(t *testing.T) {
		service := &LocationService{}
		location := &model.LocationInfo{CountryRegionSpot: ""}
		service.populateImage(context.Background(), location, "Ireland", "Donegal")
		// Should not panic
	})

	t.Run("handles invalid CountryRegionSpot format", func(t *testing.T) {
		service := &LocationService{}
		location := &model.LocationInfo{CountryRegionSpot: "invalid"}
		service.populateImage(context.Background(), location, "Ireland", "Donegal")
		// Should not panic (parts < 3)
	})
}
