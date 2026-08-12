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

const (
	invalidDateStr = "invalid-date"
	testSpotID     = "Ireland_Donegal_Bundoran"
)

func TestReportService_getUserAndValidate(t *testing.T) {
	tests := []struct {
		setupMock func() *UserService
		name      string
		userEmail string
		wantErr   bool
		wantIs    error
	}{
		{
			name:      "valid user with UUID",
			userEmail: "test@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: "test-uuid-123"}, nil
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: false,
		},
		{
			name:      "user not found",
			userEmail: "notfound@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			wantIs:  ErrReportUserNotFound,
		},
		{
			name:      "user nil",
			userEmail: "nil@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, nil
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			wantIs:  ErrReportUserNotFound,
		},
		{
			name:      "user without UUID",
			userEmail: "nouuid@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return &model.User{Email: email, UUID: ""}, nil
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			wantIs:  ErrReportUserMissingUUID,
		},
		{
			name:      "repository error",
			userEmail: "error@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
						return nil, errors.New("database error")
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			wantIs:  ErrReportUserLookupFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService := tt.setupMock()
			reportService := &ReportService{userLookup: userService}

			user, err := reportService.getUserAndValidate(ctx, tt.userEmail)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
					t.Errorf("errors.Is: want %v, got %v", tt.wantIs, err)
				}
				if user != nil {
					t.Error("expected nil user on error")
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else {
				if user == nil {
					t.Fatal("expected user but got nil")
				}
				if user.UUID == "" {
					t.Error("expected user to have UUID")
				}
			}
		})
	}
}

func TestParseReportDate(t *testing.T) {
	tests := []struct {
		wantTime func() time.Time
		name     string
		dateStr  string
	}{
		{
			name:    "valid date string",
			dateStr: "2024-01-15 14:30:00",
			wantTime: func() time.Time {
				t, _ := time.Parse("2006-01-02 15:04:05", "2024-01-15 14:30:00")
				return t
			},
		},
		{
			name:     "empty string uses current time",
			dateStr:  "",
			wantTime: time.Now,
		},
		{
			name:     "invalid format uses current time",
			dateStr:  invalidDateStr,
			wantTime: time.Now,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReportDate(tt.dateStr)
			want := tt.wantTime()

			if tt.dateStr == "" || tt.dateStr == invalidDateStr {
				// For current time cases, just check it's recent (within 1 second)
				diff := got.Sub(want)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Second {
					t.Errorf("expected time close to now, got %v", got)
				}
			} else if !got.Equal(want) {
				t.Errorf("parseReportDate(%q) = %v, want %v", tt.dateStr, got, want)
			}
		})
	}
}

func TestAddReportFieldsToReport(t *testing.T) {
	tests := []struct {
		report        *model.SurfReport
		want          *model.SurfReport
		name          string
		surfSize      string
		windAmount    string
		windDirection string
		consistency   string
		quality       string
		messiness     string
	}{
		{
			name:          "all fields set",
			report:        &model.SurfReport{},
			surfSize:      "head-high",
			windAmount:    "light",
			windDirection: "offshore",
			consistency:   "consistent",
			quality:       "good",
			messiness:     "clean",
			want: &model.SurfReport{
				SurfSize:      "head-high",
				WindAmount:    "light",
				WindDirection: "offshore",
				Consistency:   "consistent",
				Quality:       "good",
				Messiness:     "clean",
			},
		},
		{
			name:          "empty fields not set",
			report:        &model.SurfReport{SurfSize: "existing"},
			surfSize:      "",
			windAmount:    "",
			windDirection: "",
			consistency:   "",
			quality:       "",
			messiness:     "",
			want: &model.SurfReport{
				SurfSize: "existing",
			},
		},
		{
			name:          "partial fields",
			report:        &model.SurfReport{},
			surfSize:      "overhead",
			windAmount:    "",
			windDirection: "onshore",
			consistency:   "",
			quality:       "excellent",
			messiness:     "",
			want: &model.SurfReport{
				SurfSize:      "overhead",
				WindDirection: "onshore",
				Quality:       "excellent",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addReportFieldsToReport(
				tt.report,
				tt.surfSize,
				tt.windAmount,
				tt.windDirection,
				tt.consistency,
				tt.quality,
				tt.messiness,
			)

			if tt.report.SurfSize != tt.want.SurfSize {
				t.Errorf("SurfSize = %q, want %q", tt.report.SurfSize, tt.want.SurfSize)
			}
			if tt.report.WindAmount != tt.want.WindAmount {
				t.Errorf("WindAmount = %q, want %q", tt.report.WindAmount, tt.want.WindAmount)
			}
			if tt.report.WindDirection != tt.want.WindDirection {
				t.Errorf("WindDirection = %q, want %q", tt.report.WindDirection, tt.want.WindDirection)
			}
			if tt.report.Consistency != tt.want.Consistency {
				t.Errorf("Consistency = %q, want %q", tt.report.Consistency, tt.want.Consistency)
			}
			if tt.report.Quality != tt.want.Quality {
				t.Errorf("Quality = %q, want %q", tt.report.Quality, tt.want.Quality)
			}
			if tt.report.Messiness != tt.want.Messiness {
				t.Errorf("Messiness = %q, want %q", tt.report.Messiness, tt.want.Messiness)
			}
		})
	}
}

func TestReportService_createBaseReport(t *testing.T) {
	service := &ReportService{}
	currentTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	report := service.createBaseReport(
		testSpotID,
		"2024-01-15T14:30:00Z_test-uuid",
		"test@example.com",
		"Test User",
		"test-uuid",
		currentTime,
		"image",
		false,
	)

	if report == nil {
		t.Fatal("expected report but got nil")
	}
	if report.CountryRegionSpot != testSpotID {
		t.Errorf("CountryRegionSpot = %q, want %q", report.CountryRegionSpot, testSpotID)
	}
	if report.UserEmail != "test@example.com" {
		t.Errorf("UserEmail = %q, want %q", report.UserEmail, "test@example.com")
	}
	if report.Reporter != "Test User" {
		t.Errorf("Reporter = %q, want %q", report.Reporter, "Test User")
	}
	if report.ReportedBy != "test-uuid" {
		t.Errorf("ReportedBy = %q, want %q", report.ReportedBy, "test-uuid")
	}
	if report.MediaType != "image" {
		t.Errorf("MediaType = %q, want %q", report.MediaType, "image")
	}
	if report.IOSValidated {
		t.Error("expected IOSValidated to be false")
	}
	if !report.Timestamp.Equal(currentTime.UTC()) {
		t.Errorf("Timestamp = %v, want %v", report.Timestamp, currentTime.UTC())
	}
}

func TestReportService_storeReport(t *testing.T) {
	tests := []struct {
		report    *model.SurfReport
		setupMock func() repository.ReportRepository
		name      string
		wantErr   bool
	}{
		{
			name:   "successful store",
			report: &model.SurfReport{CountryRegionSpot: testSpotID},
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if report.CountryRegionSpot != testSpotID {
							return errors.New("unexpected report")
						}
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name:   "nil report",
			report: nil,
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{}
			},
			wantErr: true,
		},
		{
			name:   "repository error",
			report: &model.SurfReport{CountryRegionSpot: testSpotID},
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, _ *model.SurfReport) error {
						return errors.New("database error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			service := &ReportService{reportRepo: tt.setupMock()}

			err := service.storeReport(ctx, tt.report)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPrimarySourceForSpotID(t *testing.T) {
	tests := []struct {
		spotID string
		want   string
	}{
		{"ireland#connacht#easkey", sourceComposedIreland},
		{"Ireland#Donegal#Bundoran", sourceComposedIreland},
		{"usa#ca#spot", "stormglass"},
		{"short", "stormglass"},
		{"", "stormglass"},
	}
	for _, tt := range tests {
		t.Run(tt.spotID, func(t *testing.T) {
			got := primarySourceForSpotID(tt.spotID)
			if got != tt.want {
				t.Fatalf("primarySourceForSpotID(%q) = %q, want %q", tt.spotID, got, tt.want)
			}
		})
	}
}

func TestSelectPrimaryForecast(t *testing.T) {
	spotID := "ireland#donegal#bundoran"
	points := []*model.ForecastDataPoint{
		{Source: "stormglass", Data: map[string]interface{}{"wave_height": 2.0}},
		{Source: sourceComposedIreland, Data: map[string]interface{}{"wave_height": 2.5}},
		{Source: "weatherkit", Data: map[string]interface{}{"wave_height": 1.8}},
	}
	got := selectPrimaryForecast(spotID, points)
	if got == nil {
		t.Fatal("expected non-nil forecast")
	}
	if got.Source != sourceComposedIreland {
		t.Fatalf("expected primary source %s for Ireland, got %s", sourceComposedIreland, got.Source)
	}
	// When primary is not present, returns first
	pointsUSA := []*model.ForecastDataPoint{
		{Source: "stormglass", Data: map[string]interface{}{}},
		{Source: "weatherkit", Data: map[string]interface{}{}},
	}
	gotUSA := selectPrimaryForecast("usa#ca#spot", pointsUSA)
	if gotUSA == nil || gotUSA.Source != "stormglass" {
		t.Fatalf("expected first point when primary not in list, got %v", gotUSA)
	}
	// Empty list
	if selectPrimaryForecast(spotID, nil) != nil {
		t.Fatal("expected nil for nil slice")
	}
	if selectPrimaryForecast(spotID, []*model.ForecastDataPoint{}) != nil {
		t.Fatal("expected nil for empty slice")
	}
}
