package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestReportService_GetTodaysSurfReports(t *testing.T) {
	ctx := context.Background()
	expectedReports := []*model.SurfReport{
		{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now()},
		{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now().Add(-1 * time.Hour)},
	}

	reportRepo := &mockrepo.ReportRepo{
		GetBySpotFn: func(_ context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error) {
			if country != forecastTestCountry || region != forecastTestRegion || spot != forecastTestSpot {
				t.Errorf("unexpected location: %s/%s/%s", country, region, spot)
			}
			if limit != 1 {
				t.Errorf("expected limit 1, got %d", limit)
			}
			return expectedReports, nil
		},
	}

	service := &ReportService{reportRepo: reportRepo}

	reports, err := service.GetTodaysSurfReports(ctx, forecastTestCountry, forecastTestRegion, forecastTestSpot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 2 {
		t.Errorf("expected 2 reports, got %d", len(reports))
	}
}

func TestReportService_GetSpotSurfReports(t *testing.T) {
	tests := []struct {
		setupMock     func() repository.ReportRepository
		name          string
		country       string
		region        string
		spot          string
		limit         int
		expectedCount int
		wantErr       bool
	}{
		{
			name:    "successful query with limit",
			country: "Ireland",
			region:  "Donegal",
			spot:    "Bundoran",
			limit:   5,
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					GetBySpotFn: func(_ context.Context, _, _, _ string, limit int) ([]*model.SurfReport, error) {
						reports := make([]*model.SurfReport, limit)
						for i := 0; i < limit; i++ {
							reports[i] = &model.SurfReport{
								CountryRegionSpot: forecastTestSpotID,
								Timestamp:         time.Now().Add(-time.Duration(i) * time.Hour),
							}
						}
						return reports, nil
					},
				}
			},
			wantErr:       false,
			expectedCount: 5,
		},
		{
			name:    "repository error",
			country: "Ireland",
			region:  "Donegal",
			spot:    "Bundoran",
			limit:   5,
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					GetBySpotFn: func(_ context.Context, _, _, _ string, _ int) ([]*model.SurfReport, error) {
						return nil, errors.New("database error")
					},
				}
			},
			wantErr:       true,
			expectedCount: 0,
		},
		{
			name:    "empty results",
			country: "Ireland",
			region:  "Donegal",
			spot:    "Bundoran",
			limit:   5,
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					GetBySpotFn: func(_ context.Context, _, _, _ string, _ int) ([]*model.SurfReport, error) {
						return []*model.SurfReport{}, nil
					},
				}
			},
			wantErr:       false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{reportRepo: tt.setupMock()}

			reports, err := service.GetSpotSurfReports(ctx, tt.country, tt.region, tt.spot, tt.limit)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(reports) != tt.expectedCount {
					t.Errorf("expected %d reports, got %d", tt.expectedCount, len(reports))
				}
			}
		})
	}
}

func TestReportService_convertReportsToMaps(t *testing.T) {
	tests := []struct {
		name    string
		reports []*model.SurfReport
		wantErr bool
	}{
		{
			name: "successful conversion",
			reports: []*model.SurfReport{
				{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now()},
				{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now()},
			},
			wantErr: false,
		},
		{
			name:    "empty reports",
			reports: []*model.SurfReport{},
			wantErr: false,
		},
		{
			name: "reports with nil entries",
			reports: []*model.SurfReport{
				{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now()},
				nil,
				{CountryRegionSpot: forecastTestSpotID, Timestamp: time.Now()},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ReportService{}

			maps, err := service.convertReportsToMaps(tt.reports)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				expectedCount := 0
				for _, r := range tt.reports {
					if r != nil {
						expectedCount++
					}
				}
				if len(maps) != expectedCount {
					t.Errorf("expected %d maps, got %d", expectedCount, len(maps))
				}
			}
		})
	}
}

func TestReportService_normalizeSpotReports(t *testing.T) {
	tests := []struct {
		want    func([]map[string]interface{}) bool
		name    string
		reports []map[string]interface{}
	}{
		{
			name: "removes user_email and sets defaults",
			reports: []map[string]interface{}{
				{
					"user_email":     "test@example.com",
					"country":        "Ireland",
					"region":         "Donegal",
					"spot":           "Bundoran",
					"surf_size":      "head-high",
					"wind_amount":    "light",
					"wind_direction": "offshore",
				},
			},
			want: func(reports []map[string]interface{}) bool {
				if len(reports) != 1 {
					return false
				}
				report := reports[0]
				if _, exists := report["user_email"]; exists {
					return false
				}
				if report["video_key"] == nil {
					return false
				}
				if report["media_type"] == nil {
					return false
				}
				if report["ios_validated"] == nil {
					return false
				}
				return true
			},
		},
		{
			name: "sets missing defaults",
			reports: []map[string]interface{}{
				{
					"country": "Ireland",
				},
			},
			want: func(reports []map[string]interface{}) bool {
				if len(reports) != 1 {
					return false
				}
				report := reports[0]
				if report["video_key"] == nil {
					return false
				}
				if report["media_type"] == nil {
					return false
				}
				if report["reporter"] == nil {
					return false
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ReportService{}
			service.normalizeSpotReports(tt.reports)

			if !tt.want(tt.reports) {
				t.Error("normalization did not produce expected result")
			}
		})
	}
}
