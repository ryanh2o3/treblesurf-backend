package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

const snapshotTableName = "TestSnapshotsTable"

func TestNewSnapshotRepo(t *testing.T) {
	repo := NewSnapshotRepo(nil, snapshotTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != snapshotTableName {
		t.Fatalf("expected table name %s, got %s", snapshotTableName, repo.tableName)
	}
}

func TestSpotSnapshotItem_FromModel_ToModel(t *testing.T) {
	now := time.Now()
	snapshot := &model.SpotSnapshot{
		SpotID:     "Ireland_Donegal_Bundoran",
		ImageKey:   "snapshots/Ireland_Donegal_Bundoran/test-image.jpg",
		Timestamp:  now,
		UploadedAt: now,
	}

	// Test conversion: Model -> Item -> Model
	item := spotSnapshotItemFromModel(snapshot)
	convertedSnapshot := item.toModel()

	if convertedSnapshot.SpotID != snapshot.SpotID {
		t.Errorf("expected SpotID %s, got %s", snapshot.SpotID, convertedSnapshot.SpotID)
	}
	if convertedSnapshot.ImageKey != snapshot.ImageKey {
		t.Errorf("expected ImageKey %s, got %s", snapshot.ImageKey, convertedSnapshot.ImageKey)
	}
	if !convertedSnapshot.Timestamp.Equal(snapshot.Timestamp) {
		t.Errorf("expected Timestamp %v, got %v", snapshot.Timestamp, convertedSnapshot.Timestamp)
	}
	if !convertedSnapshot.UploadedAt.Equal(snapshot.UploadedAt) {
		t.Errorf("expected UploadedAt %v, got %v", snapshot.UploadedAt, convertedSnapshot.UploadedAt)
	}
}

func TestSpotSnapshotItem_FromModel_NilInput(t *testing.T) {
	item := spotSnapshotItemFromModel(nil)
	if item.SpotID != "" {
		t.Error("expected empty SpotID for nil input")
	}
	if item.ImageKey != "" {
		t.Error("expected empty ImageKey for nil input")
	}
}

func TestSpotSnapshotItem_ToModel_EmptyItem(t *testing.T) {
	item := spotSnapshotItem{}
	snapshot := item.toModel()

	if snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if snapshot.SpotID != "" {
		t.Error("expected empty SpotID")
	}
	if snapshot.ImageKey != "" {
		t.Error("expected empty ImageKey")
	}
}

func TestSpotSnapshot_Timestamp_Handling(t *testing.T) {
	now := time.Now()
	timestamp := now.Add(-1 * time.Hour) // 1 hour ago

	snapshot := &model.SpotSnapshot{
		SpotID:     "Ireland_Donegal_Bundoran",
		ImageKey:   "snapshots/Ireland_Donegal_Bundoran/test-image.jpg",
		Timestamp:  timestamp,
		UploadedAt: now,
	}

	// Verify timestamps are preserved through conversion
	item := spotSnapshotItemFromModel(snapshot)
	convertedSnapshot := item.toModel()

	if !convertedSnapshot.Timestamp.Equal(timestamp) {
		t.Errorf("expected Timestamp %v, got %v", timestamp, convertedSnapshot.Timestamp)
	}
	if !convertedSnapshot.UploadedAt.Equal(now) {
		t.Errorf("expected UploadedAt %v, got %v", now, convertedSnapshot.UploadedAt)
	}
}

func TestSpotSnapshot_ImageKey_Format(t *testing.T) {
	snapshot := &model.SpotSnapshot{
		SpotID:     "Ireland_Donegal_Bundoran",
		ImageKey:   "snapshots/Ireland_Donegal_Bundoran/2024-01-15T14:30:00Z_test-uuid.jpg",
		Timestamp:  time.Now(),
		UploadedAt: time.Now(),
	}

	item := spotSnapshotItemFromModel(snapshot)
	convertedSnapshot := item.toModel()

	if convertedSnapshot.ImageKey != snapshot.ImageKey {
		t.Errorf("expected ImageKey %s, got %s", snapshot.ImageKey, convertedSnapshot.ImageKey)
	}
}

// Note: Full repository method tests (Save, GetLatestBySpot) would require
// DynamoDB client mocking or Localstack integration tests.
// These tests focus on model/item conversions and validation logic that can be
// tested without AWS SDK dependencies.
