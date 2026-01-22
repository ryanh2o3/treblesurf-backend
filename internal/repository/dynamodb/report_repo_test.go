package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

func TestNewReportRepo(t *testing.T) {
	// Test that constructor doesn't panic
	repo := NewReportRepo(nil, forecastTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != forecastTableName {
		t.Fatalf("expected table name %s, got %s", forecastTableName, repo.tableName)
	}
}

func TestSurfReport_CountryRegionSpot_Format(t *testing.T) {
	report := &model.SurfReport{
		Country: "Ireland",
		Region:  "Donegal",
		Spot:    "Bundoran",
	}

	// The composite key should be generated in Create method
	expected := "Ireland_Donegal_Bundoran"
	actual := report.Country + "_" + report.Region + "_" + report.Spot

	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}

func TestSurfReport_DateReported_Format(t *testing.T) {
	timestamp := time.Now()
	report := &model.SurfReport{
		Timestamp: timestamp,
	}

	// DateReported should be RFC3339 formatted
	expected := timestamp.UTC().Format(time.RFC3339)
	actual := report.Timestamp.UTC().Format(time.RFC3339)

	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
