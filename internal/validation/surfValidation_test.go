package validation

import "testing"

func TestIsValidSwellSize(t *testing.T) {
	tests := []struct {
		name     string
		swellSize string
		want     bool
	}{
		{"valid flat", "flat", true},
		{"valid 0-0.5", "0-0.5", true},
		{"valid 0.5-1", "0.5-1", true},
		{"valid 1-1.5", "1-1.5", true},
		{"valid 1.5-2.5", "1.5-2.5", true},
		{"valid 2.5+", "2.5+", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "FLAT", false},
		{"invalid partial", "flat-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSwellSize(tt.swellSize); got != tt.want {
				t.Errorf("IsValidSwellSize(%q) = %v, want %v", tt.swellSize, got, tt.want)
			}
		})
	}
}

func TestIsValidSurfSize(t *testing.T) {
	tests := []struct {
		name     string
		swellSize string
		want     bool
	}{
		{"valid flat", "flat", true},
		{"valid knee-waist", "knee-waist", true},
		{"valid chest-shoulder", "chest-shoulder", true},
		{"valid head-high", "head-high", true},
		{"valid overhead", "overhead", true},
		{"valid double-overhead", "double-overhead", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "FLAT", false},
		{"invalid partial", "knee", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSurfSize(tt.swellSize); got != tt.want {
				t.Errorf("IsValidSurfSize(%q) = %v, want %v", tt.swellSize, got, tt.want)
			}
		})
	}
}

func TestIsValidWindAmount(t *testing.T) {
	tests := []struct {
		name      string
		windAmount string
		want      bool
	}{
		{"valid light", "light", true},
		{"valid moderate", "moderate", true},
		{"valid strong", "strong", true},
		{"valid very-strong", "very-strong", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "LIGHT", false},
		{"invalid partial", "light-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidWindAmount(tt.windAmount); got != tt.want {
				t.Errorf("IsValidWindAmount(%q) = %v, want %v", tt.windAmount, got, tt.want)
			}
		})
	}
}

func TestIsValidWindDirection(t *testing.T) {
	tests := []struct {
		name          string
		windDirection string
		want          bool
	}{
		{"valid onshore", "onshore", true},
		{"valid offshore", "offshore", true},
		{"valid cross-shore", "cross-shore", true},
		{"valid no-wind", "no-wind", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "ONSHORE", false},
		{"invalid partial", "on", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidWindDirection(tt.windDirection); got != tt.want {
				t.Errorf("IsValidWindDirection(%q) = %v, want %v", tt.windDirection, got, tt.want)
			}
		})
	}
}

func TestIsValidSurfConditions(t *testing.T) {
	tests := []struct {
		name          string
		surfConditions string
		want          bool
	}{
		{"valid mushy", "mushy", true},
		{"valid average", "average", true},
		{"valid okay", "okay", true},
		{"valid good", "good", true},
		{"valid excellent", "excellent", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "GOOD", false},
		{"invalid partial", "goo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSurfConditions(tt.surfConditions); got != tt.want {
				t.Errorf("IsValidSurfConditions(%q) = %v, want %v", tt.surfConditions, got, tt.want)
			}
		})
	}
}

func TestIsValidSurfDifficulty(t *testing.T) {
	tests := []struct {
		name           string
		surfDifficulty string
		want           bool
	}{
		{"valid setty", "setty", true},
		{"valid consistent", "consistent", true},
		{"valid inconsistent", "inconsistent", true},
		{"valid sporadic", "sporadic", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "SETTY", false},
		{"invalid partial", "set", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSurfDifficulty(tt.surfDifficulty); got != tt.want {
				t.Errorf("IsValidSurfDifficulty(%q) = %v, want %v", tt.surfDifficulty, got, tt.want)
			}
		})
	}
}

func TestIsValidMessiness(t *testing.T) {
	tests := []struct {
		name     string
		messiness string
		want     bool
	}{
		{"valid clean", "clean", true},
		{"valid slight-chop", "slight-chop", true},
		{"valid choppy", "choppy", true},
		{"valid messy", "messy", true},
		{"invalid empty", "", false},
		{"invalid value", "invalid", false},
		{"invalid case", "CLEAN", false},
		{"invalid partial", "cle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMessiness(tt.messiness); got != tt.want {
				t.Errorf("IsValidMessiness(%q) = %v, want %v", tt.messiness, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		item  string
		want  bool
	}{
		{"found in middle", []string{"a", "b", "c"}, "b", true},
		{"found at start", []string{"a", "b", "c"}, "a", true},
		{"found at end", []string{"a", "b", "c"}, "c", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
		{"empty string item", []string{"a", "b", "c"}, "", false},
		{"case sensitive", []string{"a", "b", "c"}, "A", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.slice, tt.item); got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, got, tt.want)
			}
		})
	}
}
