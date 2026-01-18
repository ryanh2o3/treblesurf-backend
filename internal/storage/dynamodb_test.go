package storage

import (
	"testing"
)

func TestNewDynamoDBStorage(t *testing.T) {
	t.Run("creates storage with valid region", func(t *testing.T) {
		storage, err := NewDynamoDBStorage("us-east-1")
		if err != nil {
			t.Fatalf("unexpected error creating DynamoDB storage: %v", err)
		}
		if storage == nil {
			t.Fatal("expected non-nil storage")
		}
		if storage.client == nil {
			t.Error("expected non-nil DynamoDB client")
		}
	})

	t.Run("creates storage with different region", func(t *testing.T) {
		storage, err := NewDynamoDBStorage("us-west-2")
		if err != nil {
			t.Fatalf("unexpected error creating DynamoDB storage: %v", err)
		}
		if storage == nil {
			t.Fatal("expected non-nil storage")
		}
	})

	t.Run("handles empty region", func(t *testing.T) {
		// Empty region should still work (defaults may apply)
		storage, err := NewDynamoDBStorage("")
		if err != nil {
			// Empty region might fail or use defaults - both are acceptable
			t.Logf("empty region resulted in error (acceptable): %v", err)
		} else if storage == nil {
			t.Error("expected non-nil storage when error is nil")
		}
	})
}

func TestGetDynamoDBClient(t *testing.T) {
	storage, err := NewDynamoDBStorage("us-east-1")
	if err != nil {
		t.Fatalf("unexpected error creating storage: %v", err)
	}

	client := storage.GetDynamoDBClient()
	if client == nil {
		t.Error("expected non-nil DynamoDB client")
	}
	if client != storage.client {
		t.Error("expected GetDynamoDBClient to return the same client instance")
	}
}

func TestDynamoDBStorage_Interface(t *testing.T) {
	storage, err := NewDynamoDBStorage("us-east-1")
	if err != nil {
		t.Fatalf("unexpected error creating storage: %v", err)
	}

	// Verify that DynamoDBClient implements DynamoDBStorage interface
	var _ DynamoDBStorage = storage
}

// Note: Full method tests (Scan, Query, GetItem, PutItem, UpdateItem, DeleteItem)
// would require mocking the DynamoDB client or using Localstack for integration tests.
// These tests focus on constructor and client initialization that can be tested
// without complex AWS SDK mocking.
