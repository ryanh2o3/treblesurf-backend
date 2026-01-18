package service

import (
	"testing"
	"time"

	"treblesurf-backend/internal/config"
)

func TestNewTideService(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		wantDev  bool
	}{
		{
			name:    "development config",
			cfg:     &config.Config{Env: config.EnvDevelopment},
			wantDev: true,
		},
		{
			name:    "production config",
			cfg:     &config.Config{Env: config.EnvProduction},
			wantDev: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantDev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewTideService(tt.cfg)
			if service == nil {
				t.Fatal("expected service but got nil")
			}
			if service.isDevelopment != tt.wantDev {
				t.Errorf("isDevelopment = %v, want %v", service.isDevelopment, tt.wantDev)
			}
		})
	}
}

func TestTideService_GetCurrentTides(t *testing.T) {
	tests := []struct {
		name       string
		location   string
		isDev      bool
		wantTides  bool
	}{
		{
			name:      "development mode returns sample tides",
			location:  "Bundoran",
			isDev:     true,
			wantTides: true,
		},
		{
			name:      "production mode returns empty",
			location:  "Bundoran",
			isDev:     false,
			wantTides: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &TideService{isDevelopment: tt.isDev}
			tides := service.GetCurrentTides(tt.location)

			if tt.wantTides {
				if len(tides) == 0 {
					t.Error("expected tides but got empty")
				}
				// Verify tide structure
				for _, tide := range tides {
					if _, ok := tide["time"].(string); !ok {
						t.Error("expected tide to have time field")
					}
					if _, ok := tide["height"].(float64); !ok {
						t.Error("expected tide to have height field")
					}
					if _, ok := tide["type"].(string); !ok {
						t.Error("expected tide to have type field")
					}
				}
			} else {
				if len(tides) != 0 {
					t.Errorf("expected empty tides, got %d", len(tides))
				}
			}
		})
	}
}

func TestTideService_GetBeforeAfterTides(t *testing.T) {
	service := &TideService{isDevelopment: true}
	location := "Bundoran"

	prevTide, nextTide := service.GetBeforeAfterTides(location)

	// In development mode, we should get sample tides
	// The function should find the most recent tide before now and earliest after now
	if prevTide == nil && nextTide == nil {
		t.Error("expected at least one tide (prev or next)")
	}

	// If we have a previous tide, verify its structure
	if prevTide != nil {
		if _, ok := prevTide["time"].(string); !ok {
			t.Error("expected prevTide to have time field")
		}
		timeStr, ok := prevTide["time"].(string)
		if ok {
			tideTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
			if err == nil {
				if tideTime.After(time.Now()) {
					t.Error("prevTide should be before now")
				}
			}
		}
	}

	// If we have a next tide, verify its structure
	if nextTide != nil {
		if _, ok := nextTide["time"].(string); !ok {
			t.Error("expected nextTide to have time field")
		}
		timeStr, ok := nextTide["time"].(string)
		if ok {
			tideTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
			if err == nil {
				if tideTime.Before(time.Now()) {
					t.Error("nextTide should be after now")
				}
			}
		}
	}
}

func TestTideService_GetDayTides(t *testing.T) {
	service := &TideService{isDevelopment: true}
	location := "Bundoran"
	startDay := "2024-01-15"

	tideData := service.GetDayTides(location, startDay)

	if tideData == nil {
		t.Fatal("expected tide data but got nil")
	}

	// Should have data for multiple days (up to 10 days)
	if len(tideData) == 0 {
		t.Error("expected tide data for at least one day")
	}

	// Verify structure for each day
	for day, data := range tideData {
		if day == "" {
			t.Error("expected day key to be non-empty")
		}
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			t.Errorf("expected data for day %s to be a map", day)
			continue
		}
		if _, ok := dataMap["location"].(string); !ok {
			t.Error("expected day data to have location field")
		}
		if _, ok := dataMap["date"].(string); !ok {
			t.Error("expected day data to have date field")
		}
		if _, ok := dataMap["tides"].([]map[string]interface{}); !ok {
			// Could be empty slice
			if tides, ok := dataMap["tides"].([]interface{}); ok {
				if len(tides) > 0 {
					t.Error("expected tides to be array of maps or empty")
				}
			}
		}
	}
}

func TestTideService_GetDayTides_InvalidDate(t *testing.T) {
	service := &TideService{isDevelopment: true}
	location := "Bundoran"
	invalidDate := "invalid-date"

	tideData := service.GetDayTides(location, invalidDate)

	if tideData == nil {
		t.Fatal("expected tide data but got nil")
	}

	// Should return error in the data
	if err, ok := tideData["error"].(string); ok {
		if err == "" {
			t.Error("expected error message for invalid date")
		}
	}
}

func TestTideService_getTides(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		date      string
		isDev     bool
		wantTides bool
	}{
		{
			name:      "development mode returns sample tides",
			location:  "Bundoran",
			date:      "2024-01-15",
			isDev:     true,
			wantTides: true,
		},
		{
			name:      "production mode returns nil",
			location:  "Bundoran",
			date:      "2024-01-15",
			isDev:     false,
			wantTides: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &TideService{isDevelopment: tt.isDev}
			tides := service.getTides(tt.location, tt.date)

			if tt.wantTides {
				if tides == nil {
					t.Fatal("expected tides but got nil")
				}
				if _, ok := tides["location"].(string); !ok {
					t.Error("expected tides to have location field")
				}
				if _, ok := tides["date"].(string); !ok {
					t.Error("expected tides to have date field")
				}
				if _, ok := tides["tides"].([]map[string]interface{}); !ok {
					t.Error("expected tides to have tides array")
				}
			} else {
				if tides != nil {
					t.Error("expected nil tides in production mode")
				}
			}
		})
	}
}
