package controller

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

func TestParseTime(t *testing.T) {
	t.Run("parses RFC3339 timestamp", func(t *testing.T) {
		ts, err := parseTime("2024-01-15T14:30:00Z")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts.IsZero() {
			t.Error("expected non-zero time")
		}
	})

	t.Run("parses date only", func(t *testing.T) {
		ts, err := parseTime("2024-01-15")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts.IsZero() {
			t.Error("expected non-zero time")
		}
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		_, err := parseTime("invalid-date")
		if err == nil {
			t.Error("expected error for invalid date format")
		}
	})
}

func TestBuoyLocationToResponse_Nil(t *testing.T) {
	result := buoyLocationToResponse("test", nil)
	if result != nil {
		t.Error("expected nil for nil location")
	}
}

func TestBuoyLocationToResponse_Valid(t *testing.T) {
	location := &model.BuoyLocation{
		Name:      "M4",
		Region:    "Atlantic",
		Country:   "Ireland",
		Spot:      "Bundoran",
		Latitude:  51.0,
		Longitude: -10.0,
	}
	result := buoyLocationToResponse("M4", location)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != "M4" {
		t.Errorf("expected Name M4, got %s", result.Name)
	}
}

func TestBuoyDataToResponse_Nil(t *testing.T) {
	result := buoyDataToResponse(nil)
	if result != nil {
		t.Error("expected nil for nil data")
	}
}

func TestBuoyDataToResponse_Valid(t *testing.T) {
	now := time.Now()
	data := &model.BuoyData{
		BuoyName:     "M4",
		WaveHeight:   2.5,
		WavePeriod:   8.0,
		MaxPeriod:    10.0,
		WaveDirection: 180.0,
		WindSpeed:    15.0,
		WindDirection: 200.0,
		Temperature:  12.5,
		Pressure:     1013.25,
		Timestamp:    now,
	}
	result := buoyDataToResponse(data)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.WaveHeight != 2.5 {
		t.Errorf("expected WaveHeight 2.5, got %f", result.WaveHeight)
	}
}

func TestBuoyDataSliceToResponses_NilData(t *testing.T) {
	result := buoyDataSliceToResponses(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestBuoyDataSliceToResponses_WithNilItems(t *testing.T) {
	data := []*model.BuoyData{
		{BuoyName: "M4", WaveHeight: 2.5, Timestamp: time.Now()},
		nil,
		{BuoyName: "M5", WaveHeight: 3.0, Timestamp: time.Now()},
	}
	result := buoyDataSliceToResponses(data)
	// Should have 2 items (nil should be skipped)
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}
