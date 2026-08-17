package service

import (
	"testing"

	mockrepo "treblesurf-backend/internal/repository/mock"

	"github.com/aws/aws-sdk-go/service/rekognition"
)

type stubRekognition struct{}

func (stubRekognition) DetectLabels(*rekognition.DetectLabelsInput) (*rekognition.DetectLabelsOutput, error) {
	return &rekognition.DetectLabelsOutput{}, nil
}

func TestNewReportService(t *testing.T) {
	userSvc, err := NewUserService(&mockrepo.UserRepo{})
	if err != nil {
		t.Fatalf("NewUserService: %v", err)
	}
	wsSvc, err := NewWebSocketService(
		&mockrepo.WebSocketRepo{},
		&mockrepo.SpotSubscriptionRepo{},
		[]byte("secret"),
		"",
		"",
	)
	if err != nil {
		t.Fatalf("NewWebSocketService: %v", err)
	}
	svc, err := NewReportService(
		&mockrepo.MediaRepo{},
		&mockrepo.ReportRepo{},
		&mockrepo.BuoyRepo{},
		&mockrepo.LocationRepo{},
		&mockrepo.ForecastRepo{},
		stubRekognition{},
		userSvc,
		wsSvc,
	)
	if err != nil {
		t.Fatalf("NewReportService: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestReportService_IsValidSurfSize(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name      string
		swellSize string
		want      bool
	}{
		{"valid size", "knee-waist", true},
		{"valid size", "head-high", true},
		{"invalid size", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidSurfSize(tt.swellSize)
			if got != tt.want {
				t.Errorf("IsValidSurfSize(%q) = %v, want %v", tt.swellSize, got, tt.want)
			}
		})
	}
}

func TestReportService_IsValidWindAmount(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name       string
		windAmount string
		want       bool
	}{
		{"valid amount", "light", true},
		{"valid amount", "moderate", true},
		{"invalid amount", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidWindAmount(tt.windAmount)
			if got != tt.want {
				t.Errorf("IsValidWindAmount(%q) = %v, want %v", tt.windAmount, got, tt.want)
			}
		})
	}
}

func TestReportService_IsValidWindDirection(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name          string
		windDirection string
		want          bool
	}{
		{"valid direction", "offshore", true},
		{"valid direction", "onshore", true},
		{"invalid direction", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidWindDirection(tt.windDirection)
			if got != tt.want {
				t.Errorf("IsValidWindDirection(%q) = %v, want %v", tt.windDirection, got, tt.want)
			}
		})
	}
}

func TestReportService_IsValidSurfConditions(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name           string
		surfConditions string
		want           bool
	}{
		{"valid condition", "good", true},
		{"valid condition", "excellent", true},
		{"invalid condition", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidSurfConditions(tt.surfConditions)
			if got != tt.want {
				t.Errorf("IsValidSurfConditions(%q) = %v, want %v", tt.surfConditions, got, tt.want)
			}
		})
	}
}

func TestReportService_IsValidSurfDifficulty(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name           string
		surfDifficulty string
		want           bool
	}{
		{"valid difficulty", "consistent", true},
		{"valid difficulty", "setty", true},
		{"invalid difficulty", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidSurfDifficulty(tt.surfDifficulty)
			if got != tt.want {
				t.Errorf("IsValidSurfDifficulty(%q) = %v, want %v", tt.surfDifficulty, got, tt.want)
			}
		})
	}
}

func TestReportService_IsValidMessiness(t *testing.T) {
	service := &ReportService{}
	tests := []struct {
		name      string
		messiness string
		want      bool
	}{
		{"valid messiness", "clean", true},
		{"valid messiness", "messy", true},
		{"invalid messiness", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsValidMessiness(tt.messiness)
			if got != tt.want {
				t.Errorf("IsValidMessiness(%q) = %v, want %v", tt.messiness, got, tt.want)
			}
		})
	}
}
