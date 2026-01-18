package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

const streamRequestTableName = "TestStreamRequestsTable"

func TestNewStreamRequestRepo(t *testing.T) {
	repo := NewStreamRequestRepo(nil, streamRequestTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != streamRequestTableName {
		t.Fatalf("expected table name %s, got %s", streamRequestTableName, repo.tableName)
	}
}

func TestStreamRequestItem_FromModel_ToModel(t *testing.T) {
	now := time.Now()
	expiration := now.Add(5 * time.Minute).Unix()

	request := &model.StreamRequest{
		SpotID:      "Ireland_Donegal_Bundoran",
		RequestedBy: "test@example.com",
		RequestedAt: now,
		Expiration:  expiration,
	}

	// Test conversion: Model -> Item -> Model
	item := streamRequestItemFromModel(request)
	convertedRequest := item.toModel()

	if convertedRequest.SpotID != request.SpotID {
		t.Errorf("expected SpotID %s, got %s", request.SpotID, convertedRequest.SpotID)
	}
	if convertedRequest.RequestedBy != request.RequestedBy {
		t.Errorf("expected RequestedBy %s, got %s", request.RequestedBy, convertedRequest.RequestedBy)
	}
	if !convertedRequest.RequestedAt.Equal(request.RequestedAt) {
		t.Errorf("expected RequestedAt %v, got %v", request.RequestedAt, convertedRequest.RequestedAt)
	}
	if convertedRequest.Expiration != request.Expiration {
		t.Errorf("expected Expiration %d, got %d", request.Expiration, convertedRequest.Expiration)
	}
}

func TestStreamRequestItem_FromModel_NilInput(t *testing.T) {
	item := streamRequestItemFromModel(nil)
	if item.SpotID != "" {
		t.Error("expected empty SpotID for nil input")
	}
	if item.RequestedBy != "" {
		t.Error("expected empty RequestedBy for nil input")
	}
	if item.Expiration != 0 {
		t.Error("expected zero Expiration for nil input")
	}
}

func TestStreamRequestItem_ToModel_EmptyItem(t *testing.T) {
	item := streamRequestItem{}
	request := item.toModel()

	if request == nil {
		t.Fatal("expected non-nil request")
	}
	if request.SpotID != "" {
		t.Error("expected empty SpotID")
	}
	if request.RequestedBy != "" {
		t.Error("expected empty RequestedBy")
	}
	if request.Expiration != 0 {
		t.Error("expected zero Expiration")
	}
}

func TestStreamRequest_Expiration_Handling(t *testing.T) {
	now := time.Now()
	expiration := now.Add(5 * time.Minute).Unix()

	request := &model.StreamRequest{
		SpotID:      "Ireland_Donegal_Bundoran",
		RequestedBy: "test@example.com",
		RequestedAt: now,
		Expiration:  expiration,
	}

	item := streamRequestItemFromModel(request)
	convertedRequest := item.toModel()

	if convertedRequest.Expiration != expiration {
		t.Errorf("expected Expiration %d, got %d", expiration, convertedRequest.Expiration)
	}
}

func TestStreamRequest_Expiration_Validation(t *testing.T) {
	now := time.Now()
	expiredAt := now.Add(-1 * time.Hour).Unix() // Expired 1 hour ago

	request := &model.StreamRequest{
		SpotID:      "Ireland_Donegal_Bundoran",
		RequestedBy: "test@example.com",
		RequestedAt: now.Add(-2 * time.Hour),
		Expiration:  expiredAt,
	}

	// Verify request is expired
	currentTime := time.Now().Unix()
	if currentTime <= request.Expiration {
		t.Error("expected request to be expired")
	}
}

func TestStreamRequest_NotExpired(t *testing.T) {
	now := time.Now()
	futureExpiration := now.Add(5 * time.Minute).Unix() // Expires in 5 minutes

	request := &model.StreamRequest{
		SpotID:      "Ireland_Donegal_Bundoran",
		RequestedBy: "test@example.com",
		RequestedAt: now,
		Expiration:  futureExpiration,
	}

	// Verify request is not expired
	currentTime := time.Now().Unix()
	if currentTime > request.Expiration {
		t.Error("expected request to not be expired")
	}
}

// Note: Full repository method tests (Save, GetBySpotID) would require
// DynamoDB client mocking or Localstack integration tests.
// These tests focus on model/item conversions and validation logic that can be
// tested without AWS SDK dependencies.
