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

type similarityMocks func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository)

type matchingConditionsMocks func() (
	repository.ReportRepository,
	repository.BuoyRepository,
	repository.LocationRepository,
	repository.ForecastDataRepository,
)

func TestReportService_GetSurfReportsWithSimilarBuoyData(t *testing.T) {
	tests := []struct {
		setupMock     similarityMocks
		regionName    string
		spotName      string
		name          string
		buoyName      string
		countryName   string
		period        float64
		waveDirection float64
		daysBack      int
		maxResults    int
		waveHeight    float64
		wantCount     int
		wantErr       bool
	}{
		{
			name:          "successful match with spot filter",
			waveHeight:    2.0,
			waveDirection: 180.0,
			period:        12.0,
			buoyName:      "test-buoy",
			countryName:   "Ireland",
			regionName:    "Donegal",
			spotName:      "Bundoran",
			daysBack:      30,
			maxResults:    10,
			setupMock: func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository) {
				reportRepo := &mockrepo.ReportRepo{
					GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
						return []*model.SurfReport{
							{
								CountryRegionSpot: "Ireland_Donegal_Bundoran",
								Timestamp:         time.Now().Add(-2 * time.Hour),
								Time:              time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
							},
						}, nil
					},
				}
				buoyRepo := &mockrepo.BuoyRepo{
					GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
						return map[string]*model.BuoyLocation{
							"test-buoy": {
								Name:      "test-buoy",
								Latitude:  54.5,
								Longitude: -8.5,
							},
						}, nil
					},
					GetBatchDataRangesFn: func(_ context.Context, _ []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
						return map[string][]*model.BuoyData{
							"test-buoy": {
								{
									BuoyName:      "test-buoy",
									Timestamp:     time.Now().Add(-2 * time.Hour),
									WaveHeight:    2.1,
									WaveDirection: 180.5,
									MaxPeriod:     12.2,
								},
							},
						}, nil
					},
				}
				locationRepo := &mockrepo.LocationRepo{
					GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
						return &model.LocationInfo{
							Latitude:  54.5,
							Longitude: -8.5,
						}, nil
					},
				}
				return reportRepo, buoyRepo, locationRepo
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:          "no matching reports",
			waveHeight:    2.0,
			waveDirection: 180.0,
			period:        12.0,
			buoyName:      "test-buoy",
			countryName:   "Ireland",
			regionName:    "Donegal",
			spotName:      "Bundoran",
			daysBack:      30,
			maxResults:    10,
			setupMock: func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository) {
				reportRepo := &mockrepo.ReportRepo{
					GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
						return []*model.SurfReport{}, nil
					},
				}
				buoyRepo := &mockrepo.BuoyRepo{
					GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
						return map[string]*model.BuoyLocation{
							"test-buoy": {
								Name:      "test-buoy",
								Latitude:  54.5,
								Longitude: -8.5,
							},
						}, nil
					},
				}
				locationRepo := &mockrepo.LocationRepo{
					GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
						return &model.LocationInfo{
							Latitude:  54.5,
							Longitude: -8.5,
						}, nil
					},
				}
				return reportRepo, buoyRepo, locationRepo
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:          "buoy not found",
			waveHeight:    2.0,
			waveDirection: 180.0,
			period:        12.0,
			buoyName:      "missing-buoy",
			countryName:   "Ireland",
			regionName:    "Donegal",
			spotName:      "Bundoran",
			daysBack:      30,
			maxResults:    10,
			setupMock: func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository) {
				reportRepo := &mockrepo.ReportRepo{
					GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
						// Return at least one report so the function continues to check the buoy
						return []*model.SurfReport{
							{
								CountryRegionSpot: "Ireland_Donegal_Bundoran",
								Timestamp:         time.Now().Add(-2 * time.Hour),
								Time:              time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
							},
						}, nil
					},
				}
				buoyRepo := &mockrepo.BuoyRepo{
					GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
						return map[string]*model.BuoyLocation{}, nil
					},
				}
				locationRepo := &mockrepo.LocationRepo{}
				return reportRepo, buoyRepo, locationRepo
			},
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:          "repository error",
			waveHeight:    2.0,
			waveDirection: 180.0,
			period:        12.0,
			buoyName:      "test-buoy",
			countryName:   "Ireland",
			regionName:    "Donegal",
			spotName:      "Bundoran",
			daysBack:      30,
			maxResults:    10,
			setupMock: func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository) {
				reportRepo := &mockrepo.ReportRepo{
					GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
						return nil, errors.New("database error")
					},
				}
				buoyRepo := &mockrepo.BuoyRepo{
					GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
						return map[string]*model.BuoyLocation{
							"test-buoy": {
								Name:      "test-buoy",
								Latitude:  54.5,
								Longitude: -8.5,
							},
						}, nil
					},
				}
				locationRepo := &mockrepo.LocationRepo{}
				return reportRepo, buoyRepo, locationRepo
			},
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:          "scan all reports when no spot filter",
			waveHeight:    2.0,
			waveDirection: 180.0,
			period:        12.0,
			buoyName:      "test-buoy",
			countryName:   "",
			regionName:    "",
			spotName:      "",
			daysBack:      30,
			maxResults:    10,
			setupMock: func() (repository.ReportRepository, repository.BuoyRepository, repository.LocationRepository) {
				reportRepo := &mockrepo.ReportRepo{
					ScanSinceFn: func(_ context.Context, _ time.Time, _ int) ([]*model.SurfReport, error) {
						return []*model.SurfReport{
							{
								CountryRegionSpot: "Ireland_Donegal_Bundoran",
								Timestamp:         time.Now().Add(-2 * time.Hour),
								Time:              time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
							},
						}, nil
					},
				}
				buoyRepo := &mockrepo.BuoyRepo{
					GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
						return map[string]*model.BuoyLocation{
							"test-buoy": {
								Name:      "test-buoy",
								Latitude:  54.5,
								Longitude: -8.5,
							},
						}, nil
					},
					GetBatchDataRangesFn: func(_ context.Context, _ []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
						return map[string][]*model.BuoyData{
							"test-buoy": {
								{
									BuoyName:      "test-buoy",
									Timestamp:     time.Now().Add(-2 * time.Hour),
									WaveHeight:    2.1,
									WaveDirection: 180.5,
									MaxPeriod:     12.2,
								},
							},
						}, nil
					},
				}
				locationRepo := &mockrepo.LocationRepo{
					GetLocationInfoFn: func(_ context.Context, country, region, spot string) (*model.LocationInfo, error) {
						parts := []string{country, region, spot}
						if parts[0] == forecastTestCountry && parts[1] == forecastTestRegion && parts[2] == forecastTestSpot {
							return &model.LocationInfo{
								Latitude:  54.5,
								Longitude: -8.5,
							}, nil
						}
						return nil, errors.New("location not found")
					},
				}
				return reportRepo, buoyRepo, locationRepo
			},
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reportRepo, buoyRepo, locationRepo := tt.setupMock()
			service := &ReportService{
				reportRepo:   reportRepo,
				buoyRepo:     buoyRepo,
				locationRepo: locationRepo,
			}

			reports, err := service.GetSurfReportsWithSimilarBuoyData(
				ctx,
				tt.waveHeight,
				tt.waveDirection,
				tt.period,
				tt.buoyName,
				tt.countryName,
				tt.regionName,
				tt.spotName,
				tt.daysBack,
				tt.maxResults,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(reports) != tt.wantCount {
					t.Errorf("expected %d reports, got %d", tt.wantCount, len(reports))
				}
				// Verify similarity scores are present
				if len(reports) > 0 {
					for _, report := range reports {
						if _, ok := report["similarity"]; !ok {
							t.Error("expected similarity score in report")
						}
					}
				}
			}
		})
	}
}

func TestReportService_GetSurfReportsWithMatchingConditions(t *testing.T) {
	buildMatchingConditionsSuccess := func() (
		repository.ReportRepository,
		repository.BuoyRepository,
		repository.LocationRepository,
		repository.ForecastDataRepository,
	) {
		reportRepo := &mockrepo.ReportRepo{
			GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
				return []*model.SurfReport{
					{
						CountryRegionSpot: "Ireland_Donegal_Bundoran",
						Timestamp:         time.Now().Add(-2 * time.Hour),
						Time:              time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					},
				}, nil
			},
		}
		buoyRepo := &mockrepo.BuoyRepo{
			GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
				return map[string]*model.BuoyLocation{
					"buoy1": {
						Name:      "buoy1",
						Latitude:  54.5,
						Longitude: -8.5,
					},
					"buoy2": {
						Name:      "buoy2",
						Latitude:  54.6,
						Longitude: -8.6,
					},
				}, nil
			},
			GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
				return &model.BuoyData{
					BuoyName:      buoyName,
					WaveHeight:    2.0,
					WaveDirection: 180.0,
					MaxPeriod:     12.0,
				}, nil
			},
			GetBatchDataRangesFn: func(_ context.Context, _ []repository.BuoyDataRequest) (map[string][]*model.BuoyData, error) {
				return map[string][]*model.BuoyData{
					"buoy1": {
						{
							BuoyName:      "buoy1",
							Timestamp:     time.Now().Add(-2 * time.Hour),
							WaveHeight:    2.1,
							WaveDirection: 180.5,
							MaxPeriod:     12.2,
						},
					},
				}, nil
			},
		}
		locationRepo := &mockrepo.LocationRepo{
			GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
				return &model.LocationInfo{
					Latitude:  54.5,
					Longitude: -8.5,
				}, nil
			},
		}
		forecastRepo := &mockrepo.ForecastRepo{
			QuerySinceFn: func(_ context.Context, spotID string, _ time.Time, _ int) ([]*model.ForecastDataPoint, error) {
				return []*model.ForecastDataPoint{
					{
						SpotID:            spotID,
						ForecastTimestamp: time.Now(),
						Data: map[string]interface{}{
							"windSpeed":     10.0,
							"windDirection": 270.0,
						},
					},
				}, nil
			},
			QueryBetweenFn: func(_ context.Context, spotID string, _, _ time.Time, _ int) ([]*model.ForecastDataPoint, error) {
				return []*model.ForecastDataPoint{
					{
						SpotID:            spotID,
						ForecastTimestamp: time.Now().Add(-2 * time.Hour),
						Data: map[string]interface{}{
							"windSpeed":     10.5,
							"windDirection": 270.5,
						},
					},
				}, nil
			},
		}
		return reportRepo, buoyRepo, locationRepo, forecastRepo
	}

	buildMatchingConditionsNoReports := func() (
		repository.ReportRepository,
		repository.BuoyRepository,
		repository.LocationRepository,
		repository.ForecastDataRepository,
	) {
		reportRepo := &mockrepo.ReportRepo{
			GetBySpotAndTimeRangeFn: func(_ context.Context, _, _, _ string, _, _ time.Time) ([]*model.SurfReport, error) {
				return []*model.SurfReport{}, nil
			},
		}
		buoyRepo := &mockrepo.BuoyRepo{
			GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
				return map[string]*model.BuoyLocation{
					"buoy1": {
						Name:      "buoy1",
						Latitude:  54.5,
						Longitude: -8.5,
					},
				}, nil
			},
			GetLiveDataFn: func(_ context.Context, buoyName string) (*model.BuoyData, error) {
				return &model.BuoyData{
					BuoyName:      buoyName,
					WaveHeight:    2.0,
					WaveDirection: 180.0,
					MaxPeriod:     12.0,
				}, nil
			},
		}
		locationRepo := &mockrepo.LocationRepo{
			GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
				return &model.LocationInfo{
					Latitude:  54.5,
					Longitude: -8.5,
				}, nil
			},
		}
		return reportRepo, buoyRepo, locationRepo, &mockrepo.ForecastRepo{}
	}

	buildMatchingConditionsLocationNotFound := func() (
		repository.ReportRepository,
		repository.BuoyRepository,
		repository.LocationRepository,
		repository.ForecastDataRepository,
	) {
		reportRepo := &mockrepo.ReportRepo{}
		buoyRepo := &mockrepo.BuoyRepo{}
		locationRepo := &mockrepo.LocationRepo{
			GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
				return nil, errors.New("location not found")
			},
		}
		return reportRepo, buoyRepo, locationRepo, &mockrepo.ForecastRepo{}
	}

	buildMatchingConditionsNoBuoys := func() (
		repository.ReportRepository,
		repository.BuoyRepository,
		repository.LocationRepository,
		repository.ForecastDataRepository,
	) {
		reportRepo := &mockrepo.ReportRepo{}
		buoyRepo := &mockrepo.BuoyRepo{
			GetLocationsFn: func(_ context.Context) (map[string]*model.BuoyLocation, error) {
				return map[string]*model.BuoyLocation{}, nil
			},
		}
		locationRepo := &mockrepo.LocationRepo{
			GetLocationInfoFn: func(_ context.Context, _, _, _ string) (*model.LocationInfo, error) {
				return &model.LocationInfo{
					Latitude:  54.5,
					Longitude: -8.5,
				}, nil
			},
		}
		return reportRepo, buoyRepo, locationRepo, &mockrepo.ForecastRepo{}
	}

	tests := []struct {
		setupMock   matchingConditionsMocks
		name        string
		countryName string
		regionName  string
		spotName    string
		daysBack    int
		maxResults  int
		wantCount   int
		wantErr     bool
	}{
		{
			name:        "successful match",
			countryName: "Ireland",
			regionName:  "Donegal",
			spotName:    "Bundoran",
			daysBack:    30,
			maxResults:  10,
			setupMock: buildMatchingConditionsSuccess,
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:        "no matching reports",
			countryName: "Ireland",
			regionName:  "Donegal",
			spotName:    "Bundoran",
			daysBack:    30,
			maxResults:  10,
			setupMock: buildMatchingConditionsNoReports,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:        "location not found",
			countryName: "Ireland",
			regionName:  "Donegal",
			spotName:    "Bundoran",
			daysBack:    30,
			maxResults:  10,
			setupMock: buildMatchingConditionsLocationNotFound,
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:        "no buoys found",
			countryName: "Ireland",
			regionName:  "Donegal",
			spotName:    "Bundoran",
			daysBack:    30,
			maxResults:  10,
			setupMock: buildMatchingConditionsNoBuoys,
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			reportRepo, buoyRepo, locationRepo, forecastRepo := tt.setupMock()
			service := &ReportService{
				reportRepo:       reportRepo,
				buoyRepo:         buoyRepo,
				locationRepo:     locationRepo,
				forecastDataRepo: forecastRepo,
			}

			reports, err := service.GetSurfReportsWithMatchingConditions(
				ctx,
				tt.countryName,
				tt.regionName,
				tt.spotName,
				tt.daysBack,
				tt.maxResults,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(reports) != tt.wantCount {
					t.Errorf("expected %d reports, got %d", tt.wantCount, len(reports))
				}
				// Verify similarity scores are present
				if len(reports) > 0 {
					for _, report := range reports {
						if _, ok := report["combined_similarity"]; !ok {
							t.Error("expected combined_similarity score in report")
						}
						if _, ok := report["buoy_similarity"]; !ok {
							t.Error("expected buoy_similarity score in report")
						}
						if _, ok := report["wind_similarity"]; !ok {
							t.Error("expected wind_similarity score in report")
						}
					}
				}
			}
		})
	}
}

func TestReportService_calculateBuoyConditionSimilarity(t *testing.T) {
	service := &ReportService{}

	tests := []struct {
		buoyData    map[string]interface{}
		name        string
		predHeight  float64
		predDir     float64
		predPeriod  float64
		wantSimilar bool
	}{
		{
			name:       "exact match",
			predHeight: 2.0,
			predDir:    180.0,
			predPeriod: 12.0,
			buoyData: map[string]interface{}{
				"WaveHeight":        2.0,
				"MeanWaveDirection": 180.0,
				"MaxPeriod":         12.0,
			},
			wantSimilar: true,
		},
		{
			name:       "close match",
			predHeight: 2.0,
			predDir:    180.0,
			predPeriod: 12.0,
			buoyData: map[string]interface{}{
				"WaveHeight":        2.1,
				"MeanWaveDirection": 180.5,
				"MaxPeriod":         12.2,
			},
			wantSimilar: true,
		},
		{
			name:       "different height",
			predHeight: 2.0,
			predDir:    180.0,
			predPeriod: 12.0,
			buoyData: map[string]interface{}{
				"WaveHeight":        5.0,
				"MeanWaveDirection": 180.0,
				"MaxPeriod":         12.0,
			},
			wantSimilar: false,
		},
		{
			name:       "different direction",
			predHeight: 2.0,
			predDir:    180.0,
			predPeriod: 12.0,
			buoyData: map[string]interface{}{
				"WaveHeight":        2.0,
				"MeanWaveDirection": 270.0,
				"MaxPeriod":         12.0,
			},
			wantSimilar: false,
		},
		{
			name:       "string values",
			predHeight: 2.0,
			predDir:    180.0,
			predPeriod: 12.0,
			buoyData: map[string]interface{}{
				"WaveHeight":        "2.1",
				"MeanWaveDirection": "180.5",
				"MaxPeriod":         "12.2",
			},
			wantSimilar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := service.calculateBuoyConditionSimilarity(
				tt.predHeight,
				tt.predDir,
				tt.predPeriod,
				tt.buoyData,
			)

			if tt.wantSimilar && similarity <= 0.7 {
				t.Errorf("expected similarity > 0.7, got %f", similarity)
			}
			if !tt.wantSimilar && similarity > 0.7 {
				t.Errorf("expected similarity <= 0.7, got %f", similarity)
			}
		})
	}
}

func TestReportService_calculateDistance(t *testing.T) {
	service := &ReportService{}

	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		wantDist  float64 // Approximate distance in meters
		tolerance float64
	}{
		{
			name:      "same location",
			lat1:      54.5,
			lon1:      -8.5,
			lat2:      54.5,
			lon2:      -8.5,
			wantDist:  0.0,
			tolerance: 1.0,
		},
		{
			name:      "short distance",
			lat1:      54.5,
			lon1:      -8.5,
			lat2:      54.501,
			lon2:      -8.5,
			wantDist:  111.0, // Approximately 111 meters per degree latitude
			tolerance: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist := service.calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			diff := dist - tt.wantDist
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("calculateDistance() = %f, want %f (tolerance %f)", dist, tt.wantDist, tt.tolerance)
			}
		})
	}
}

func TestReportService_calculateBearing(t *testing.T) {
	service := &ReportService{}

	tests := []struct {
		name        string
		lat1        float64
		lon1        float64
		lat2        float64
		lon2        float64
		wantBearing float64 // Expected bearing in degrees
		tolerance   float64
	}{
		{
			name:        "north",
			lat1:        54.5,
			lon1:        -8.5,
			lat2:        54.6,
			lon2:        -8.5,
			wantBearing: 0.0, // North
			tolerance:   5.0,
		},
		{
			name:        "east",
			lat1:        54.5,
			lon1:        -8.5,
			lat2:        54.5,
			lon2:        -8.4,
			wantBearing: 90.0, // East
			tolerance:   5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bearing := service.calculateBearing(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			diff := bearing - tt.wantBearing
			if diff < 0 {
				diff = -diff
			}
			// Handle wraparound
			if diff > 180 {
				diff = 360 - diff
			}
			if diff > tt.tolerance {
				t.Errorf("calculateBearing() = %f, want %f (tolerance %f)", bearing, tt.wantBearing, tt.tolerance)
			}
		})
	}
}

func TestReportService_calculateWindSimilarity(t *testing.T) {
	service := &ReportService{}

	tests := []struct {
		name                string
		currentSpeed        float64
		currentDirection    float64
		historicalSpeed     float64
		historicalDirection float64
		wantSimilar         bool // Similarity > 0.5
	}{
		{
			name:                "exact match",
			currentSpeed:        10.0,
			currentDirection:    270.0,
			historicalSpeed:     10.0,
			historicalDirection: 270.0,
			wantSimilar:         true,
		},
		{
			name:                "close match",
			currentSpeed:        10.0,
			currentDirection:    270.0,
			historicalSpeed:     10.5,
			historicalDirection: 270.5,
			wantSimilar:         true,
		},
		{
			name:                "different speed",
			currentSpeed:        10.0,
			currentDirection:    270.0,
			historicalSpeed:     20.0,
			historicalDirection: 270.0,
			wantSimilar:         false,
		},
		{
			name:                "different direction",
			currentSpeed:        10.0,
			currentDirection:    270.0,
			historicalSpeed:     10.0,
			historicalDirection: 300.0,
			wantSimilar:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := service.calculateWindSimilarity(
				tt.currentSpeed,
				tt.currentDirection,
				tt.historicalSpeed,
				tt.historicalDirection,
			)

			if tt.wantSimilar && similarity <= 0.5 {
				t.Errorf("expected similarity > 0.5, got %f", similarity)
			}
			if !tt.wantSimilar && similarity > 0.5 {
				t.Errorf("expected similarity <= 0.5, got %f", similarity)
			}
		})
	}
}

func TestBuoyDataCache_getDataAtTime(t *testing.T) {
	tests := []struct {
		targetTime time.Time
		cacheData  map[string][]*model.BuoyData
		name       string
		buoyName   string
		wantData   bool
	}{
		{
			name: "exact match",
			cacheData: map[string][]*model.BuoyData{
				"test-buoy": {
					{
						BuoyName:  "test-buoy",
						Timestamp: time.Now().Add(-2 * time.Hour),
					},
				},
			},
			buoyName:   "test-buoy",
			targetTime: time.Now().Add(-2 * time.Hour),
			wantData:   true,
		},
		{
			name: "within 6 hours",
			cacheData: map[string][]*model.BuoyData{
				"test-buoy": {
					{
						BuoyName:  "test-buoy",
						Timestamp: time.Now().Add(-2 * time.Hour),
					},
				},
			},
			buoyName:   "test-buoy",
			targetTime: time.Now().Add(-4 * time.Hour),
			wantData:   true,
		},
		{
			name: "outside 6 hours",
			cacheData: map[string][]*model.BuoyData{
				"test-buoy": {
					{
						BuoyName:  "test-buoy",
						Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), // Fixed time
					},
				},
			},
			buoyName:   "test-buoy",
			targetTime: time.Date(2024, 1, 1, 7, 0, 0, 0, time.UTC), // 7 hours later, diff is 7h which is > 6h
			wantData:   false,
		},
		{
			name: "buoy not in cache",
			cacheData: map[string][]*model.BuoyData{
				"other-buoy": {
					{
						BuoyName:  "other-buoy",
						Timestamp: time.Now().Add(-2 * time.Hour),
					},
				},
			},
			buoyName:   "test-buoy",
			targetTime: time.Now().Add(-2 * time.Hour),
			wantData:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newBuoyDataCache(tt.cacheData)
			data := cache.getDataAtTime(tt.buoyName, tt.targetTime)

			if tt.wantData && data == nil {
				t.Error("expected data but got nil")
			}
			if !tt.wantData && data != nil {
				t.Error("expected nil but got data")
			}
		})
	}
}
