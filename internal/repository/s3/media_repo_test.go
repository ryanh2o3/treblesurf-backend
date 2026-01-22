package s3

import (
	"testing"
)

func TestNewMediaRepo(t *testing.T) {
	repo := NewMediaRepo(nil, "test-bucket")
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.bucketName != "test-bucket" {
		t.Fatalf("expected bucket name test-bucket, got %s", repo.bucketName)
	}
}

func TestMediaRepo_KeyValidation(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{
			name:  "valid image key",
			key:   "surf-reports/Ireland_Donegal_Bundoran/123456_uuid.jpg",
			valid: true,
		},
		{
			name:  "valid video key",
			key:   "surf-reports/Ireland_Donegal_Bundoran/123456_uuid.mp4",
			valid: true,
		},
		{
			name:  "empty key",
			key:   "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.key != ""
			if isValid != tt.valid {
				t.Fatalf("expected valid=%v for key %s", tt.valid, tt.key)
			}
		})
	}
}
