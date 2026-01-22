package storage

import (
	"testing"
)

func TestNewS3Storage(t *testing.T) {
	t.Run("creates storage with valid region", func(t *testing.T) {
		storage, err := NewS3Storage("us-east-1")
		if err != nil {
			t.Fatalf("unexpected error creating S3 storage: %v", err)
		}
		if storage == nil {
			t.Fatal("expected non-nil storage")
		}
		if storage.client == nil {
			t.Error("expected non-nil S3 client")
		}
	})

	t.Run("creates storage with different region", func(t *testing.T) {
		storage, err := NewS3Storage("us-west-2")
		if err != nil {
			t.Fatalf("unexpected error creating S3 storage: %v", err)
		}
		if storage == nil {
			t.Fatal("expected non-nil storage")
		}
	})

	t.Run("handles empty region", func(t *testing.T) {
		// Empty region should still work (defaults may apply)
		storage, err := NewS3Storage("")
		if err != nil {
			// Empty region might fail or use defaults - both are acceptable
			t.Logf("empty region resulted in error (acceptable): %v", err)
		} else if storage == nil {
			t.Error("expected non-nil storage when error is nil")
		}
	})
}

func TestGetS3Client(t *testing.T) {
	storage, err := NewS3Storage("us-east-1")
	if err != nil {
		t.Fatalf("unexpected error creating storage: %v", err)
	}

	client := storage.GetS3Client()
	if client == nil {
		t.Error("expected non-nil S3 client")
	}
	if client != storage.client {
		t.Error("expected GetS3Client to return the same client instance")
	}
}

func TestS3Storage_Interface(t *testing.T) {
	storage, err := NewS3Storage("us-east-1")
	if err != nil {
		t.Fatalf("unexpected error creating storage: %v", err)
	}

	// Verify that S3Client implements S3Storage interface
	var _ S3Storage = storage
}

// Note: Full method tests (GetObject, PutObject, DeleteObject, GeneratePresignedUploadURL,
// GeneratePresignedViewURL) would require mocking the S3 client or using Localstack for
// integration tests. These tests focus on constructor and client initialization that can
// be tested without complex AWS SDK mocking.
