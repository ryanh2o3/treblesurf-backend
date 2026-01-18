package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestReportService_getUserAndValidate(t *testing.T) {
	tests := []struct {
		name      string
		userEmail string
		setupMock func() *UserService
		wantErr   bool
		errMsg    string
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
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return nil, repository.ErrNotFound
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			errMsg:  "user not found",
		},
		{
			name:      "user nil",
			userEmail: "nil@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return nil, nil
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			errMsg:  "user not found",
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
			errMsg:  "user does not have a UUID",
		},
		{
			name:      "repository error",
			userEmail: "error@example.com",
			setupMock: func() *UserService {
				userRepo := &mockrepo.UserRepo{
					GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
						return nil, errors.New("database error")
					},
				}
				service, _ := NewUserService(userRepo)
				return service
			},
			wantErr: true,
			errMsg:  "failed to get user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			userService := tt.setupMock()
			reportService := &ReportService{userService: userService}

			user, err := reportService.getUserAndValidate(ctx, tt.userEmail)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
				if user != nil {
					t.Error("expected nil user on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
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
		name     string
		dateStr  string
		wantTime func() time.Time
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
			wantTime: func() time.Time { return time.Now() },
		},
		{
			name:     "invalid format uses current time",
			dateStr:  "invalid-date",
			wantTime: func() time.Time { return time.Now() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseReportDate(tt.dateStr)
			want := tt.wantTime()

			if tt.dateStr == "" || tt.dateStr == "invalid-date" {
				// For current time cases, just check it's recent (within 1 second)
				diff := got.Sub(want)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Second {
					t.Errorf("expected time close to now, got %v", got)
				}
			} else {
				if !got.Equal(want) {
					t.Errorf("parseReportDate(%q) = %v, want %v", tt.dateStr, got, want)
				}
			}
		})
	}
}

func TestAddReportFieldsToReport(t *testing.T) {
	tests := []struct {
		name         string
		report       *model.SurfReport
		surfSize     string
		windAmount   string
		windDirection string
		consistency  string
		quality      string
		messiness    string
		want         *model.SurfReport
	}{
		{
			name:         "all fields set",
			report:       &model.SurfReport{},
			surfSize:     "head-high",
			windAmount:   "light",
			windDirection: "offshore",
			consistency:  "consistent",
			quality:      "good",
			messiness:    "clean",
			want: &model.SurfReport{
				SurfSize:      "head-high",
				WindAmount:     "light",
				WindDirection:  "offshore",
				Consistency:   "consistent",
				Quality:       "good",
				Messiness:     "clean",
			},
		},
		{
			name:         "empty fields not set",
			report:       &model.SurfReport{SurfSize: "existing"},
			surfSize:     "",
			windAmount:   "",
			windDirection: "",
			consistency:  "",
			quality:      "",
			messiness:    "",
			want: &model.SurfReport{
				SurfSize: "existing",
			},
		},
		{
			name:         "partial fields",
			report:       &model.SurfReport{},
			surfSize:     "overhead",
			windAmount:   "",
			windDirection: "onshore",
			consistency:  "",
			quality:      "excellent",
			messiness:    "",
			want: &model.SurfReport{
				SurfSize:     "overhead",
				WindDirection: "onshore",
				Quality:      "excellent",
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
	ctx := context.Background()
	service := &ReportService{}
	currentTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	report := service.createBaseReport(
		"Ireland_Donegal_Bundoran",
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
	if report.CountryRegionSpot != "Ireland_Donegal_Bundoran" {
		t.Errorf("CountryRegionSpot = %q, want %q", report.CountryRegionSpot, "Ireland_Donegal_Bundoran")
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
	_ = ctx // Suppress unused variable warning
}

func TestReportService_storeReport(t *testing.T) {
	tests := []struct {
		name      string
		report    *model.SurfReport
		setupMock func() repository.ReportRepository
		wantErr   bool
	}{
		{
			name:   "successful store",
			report: &model.SurfReport{CountryRegionSpot: "Ireland_Donegal_Bundoran"},
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
						if report.CountryRegionSpot != "Ireland_Donegal_Bundoran" {
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
			report: &model.SurfReport{CountryRegionSpot: "Ireland_Donegal_Bundoran"},
			setupMock: func() repository.ReportRepository {
				return &mockrepo.ReportRepo{
					CreateFn: func(_ context.Context, report *model.SurfReport) error {
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

