package service

import (
	"context"
	"testing"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

const (
	locationTestCountry = "Ireland"
	locationTestRegion  = "Donegal"
	locationTestSpot    = "Bundoran"
	locationTestBaseURL = "https://cdn.example.com"
)

func TestNewLocationService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewLocationService(nil, locationTestBaseURL)
	if err == nil {
		t.Fatalf("expected error for nil location repository")
	}
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

	service, err := NewLocationService(repo, locationTestBaseURL)
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

	service, err := NewLocationService(repo, locationTestBaseURL)
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
	want0 := locationTestBaseURL + "/spot-images/Ireland_Donegal_Bundoran.jpg"
	if got[0].ImageURL != want0 {
		t.Fatalf("expected ImageURL %q, got %q", want0, got[0].ImageURL)
	}
	want1 := locationTestBaseURL + "/spot-images/Ireland_Donegal_Rossnowlagh.jpg"
	if got[1].ImageURL != want1 {
		t.Fatalf("expected ImageURL %q, got %q", want1, got[1].ImageURL)
	}
}

func TestLocationService_GetSpots_NoBaseURL_NoImageURL(t *testing.T) {
	ctx := context.Background()
	expected := []*model.LocationInfo{
		{CountryRegionSpot: locationTestCountry + "/" + locationTestRegion + "/" + locationTestSpot},
	}
	repo := &mockrepo.LocationRepo{
		GetSpotsFn: func(_ context.Context, _, _ string) ([]*model.LocationInfo, error) {
			return expected, nil
		},
	}

	service, err := NewLocationService(repo, "")
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetSpots(ctx, locationTestCountry, locationTestRegion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].ImageURL != "" {
		t.Fatalf("expected empty ImageURL without base URL, got %q", got[0].ImageURL)
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

	service, err := NewLocationService(repo, locationTestBaseURL)
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
	wantURL := locationTestBaseURL + "/spot-images/Ireland_Donegal_Bundoran.jpg"
	if got.ImageURL != wantURL {
		t.Fatalf("expected ImageURL %q, got %q", wantURL, got.ImageURL)
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

	service, err := NewLocationService(repo, locationTestBaseURL)
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
			return 0, 0, repository.ErrNotFound
		},
	}

	service, err := NewLocationService(repo, locationTestBaseURL)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	_, err = service.GetCoordinates(ctx, locationTestCountry, locationTestRegion, locationTestSpot)
	if err == nil {
		t.Fatalf("expected error for not found location")
	}
}

func TestLocationService_attachSpotImageURL_EdgeCases(t *testing.T) {
	t.Run("handles nil location", func(_ *testing.T) {
		service, _ := NewLocationService(&mockrepo.LocationRepo{}, locationTestBaseURL)
		service.attachSpotImageURL(nil, "Ireland", "Donegal")
	})

	t.Run("handles empty CountryRegionSpot", func(_ *testing.T) {
		service, _ := NewLocationService(&mockrepo.LocationRepo{}, locationTestBaseURL)
		location := &model.LocationInfo{CountryRegionSpot: ""}
		service.attachSpotImageURL(location, "Ireland", "Donegal")
		if location.ImageURL != "" {
			t.Fatalf("expected empty ImageURL")
		}
	})

	t.Run("handles invalid CountryRegionSpot format", func(_ *testing.T) {
		service, _ := NewLocationService(&mockrepo.LocationRepo{}, locationTestBaseURL)
		location := &model.LocationInfo{CountryRegionSpot: "invalid"}
		service.attachSpotImageURL(location, "Ireland", "Donegal")
		if location.ImageURL != "" {
			t.Fatalf("expected empty ImageURL")
		}
	})
}
