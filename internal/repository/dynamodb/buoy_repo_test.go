package dynamodb

import (
	"testing"
	"time"
)

const buoyDataTableName = "TestBuoyDataTable"
const buoyLocationTableName = "TestBuoyLocationTable"

func TestNewBuoyRepo(t *testing.T) {
	repo := NewBuoyRepo(nil, buoyDataTableName, buoyLocationTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.dataTableName != buoyDataTableName {
		t.Fatalf("expected data table name %s, got %s", buoyDataTableName, repo.dataTableName)
	}
	if repo.locationTableName != buoyLocationTableName {
		t.Fatalf("expected location table name %s, got %s", buoyLocationTableName, repo.locationTableName)
	}
	if repo.regionPrefix != "Ireland" {
		t.Errorf("expected region prefix 'Ireland', got %s", repo.regionPrefix)
	}
}

func TestBuoyDataItem_ToModel(t *testing.T) {
	now := time.Now()
	item := buoyDataItem{
		Timestamp:     now,
		BuoyName:      "TestBuoy",
		WaveHeight:    2.5,
		WavePeriod:    8.0,
		MaxPeriod:     10.0,
		WaveDirection: 180.0,
		WindSpeed:     15.0,
		WindDirection: 200.0,
		Temperature:   12.5,
		Pressure:      1013.25,
	}

	// Test conversion: Item -> Model
	convertedData := item.toModel()

	if !convertedData.Timestamp.Equal(item.Timestamp) {
		t.Errorf("expected Timestamp %v, got %v", item.Timestamp, convertedData.Timestamp)
	}
	if convertedData.BuoyName != item.BuoyName {
		t.Errorf("expected BuoyName %s, got %s", item.BuoyName, convertedData.BuoyName)
	}
	if convertedData.WaveHeight != item.WaveHeight {
		t.Errorf("expected WaveHeight %f, got %f", item.WaveHeight, convertedData.WaveHeight)
	}
	if convertedData.WavePeriod != item.WavePeriod {
		t.Errorf("expected WavePeriod %f, got %f", item.WavePeriod, convertedData.WavePeriod)
	}
	if convertedData.MaxPeriod != item.MaxPeriod {
		t.Errorf("expected MaxPeriod %f, got %f", item.MaxPeriod, convertedData.MaxPeriod)
	}
	if convertedData.WaveDirection != item.WaveDirection {
		t.Errorf("expected WaveDirection %f, got %f", item.WaveDirection, convertedData.WaveDirection)
	}
	if convertedData.WindSpeed != item.WindSpeed {
		t.Errorf("expected WindSpeed %f, got %f", item.WindSpeed, convertedData.WindSpeed)
	}
	if convertedData.WindDirection != item.WindDirection {
		t.Errorf("expected WindDirection %f, got %f", item.WindDirection, convertedData.WindDirection)
	}
	if convertedData.Temperature != item.Temperature {
		t.Errorf("expected Temperature %f, got %f", item.Temperature, convertedData.Temperature)
	}
	if convertedData.Pressure != item.Pressure {
		t.Errorf("expected Pressure %f, got %f", item.Pressure, convertedData.Pressure)
	}
}

func TestBuoyDataItem_ToModel_EmptyItem(t *testing.T) {
	item := buoyDataItem{}
	buoyData := item.toModel()

	if buoyData == nil {
		t.Fatal("expected non-nil buoyData")
	}
	if buoyData.BuoyName != "" {
		t.Error("expected empty BuoyName")
	}
	if !buoyData.Timestamp.IsZero() {
		t.Error("expected zero Timestamp")
	}
	if buoyData.WaveHeight != 0 {
		t.Error("expected zero WaveHeight")
	}
}

func TestBuoyLocationItem_ToModel(t *testing.T) {
	item := buoyLocationItem{
		Name:      "TestBuoy",
		Country:   "Ireland",
		Region:    "Donegal",
		Spot:      "Bundoran",
		Latitude:  54.4775,
		Longitude: -8.2803,
	}

	// Test conversion: Item -> Model
	convertedLocation := item.toModel()

	if convertedLocation.Name != item.Name {
		t.Errorf("expected Name %s, got %s", item.Name, convertedLocation.Name)
	}
	if convertedLocation.Country != item.Country {
		t.Errorf("expected Country %s, got %s", item.Country, convertedLocation.Country)
	}
	if convertedLocation.Region != item.Region {
		t.Errorf("expected Region %s, got %s", item.Region, convertedLocation.Region)
	}
	if convertedLocation.Spot != item.Spot {
		t.Errorf("expected Spot %s, got %s", item.Spot, convertedLocation.Spot)
	}
	if convertedLocation.Latitude != item.Latitude {
		t.Errorf("expected Latitude %f, got %f", item.Latitude, convertedLocation.Latitude)
	}
	if convertedLocation.Longitude != item.Longitude {
		t.Errorf("expected Longitude %f, got %f", item.Longitude, convertedLocation.Longitude)
	}
}

func TestBuoyLocationItem_ToModel_EmptyItem(t *testing.T) {
	item := buoyLocationItem{}
	buoyLocation := item.toModel()

	if buoyLocation == nil {
		t.Fatal("expected non-nil buoyLocation")
	}
	if buoyLocation.Name != "" {
		t.Error("expected empty Name")
	}
	if buoyLocation.Latitude != 0 {
		t.Error("expected zero Latitude")
	}
}

// Note: Full repository method tests (GetLiveData, GetDataAtTime, GetDataRange,
// GetBatchDataRanges, GetLocations) would require DynamoDB client mocking or
// Localstack integration tests. These tests focus on model/item conversions and
// constructor validation that can be tested without AWS SDK dependencies.
