package dynamodb

import (
	"testing"
)

const locationTableName = "TestLocationsTable"

func TestNewLocationRepo(t *testing.T) {
	repo := NewLocationRepo(nil, locationTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != locationTableName {
		t.Fatalf("expected table name %s, got %s", locationTableName, repo.tableName)
	}
}

// Note: LocationRepo uses model.LocationInfo directly (not item conversions),
// so model/item conversion tests are not applicable here.
// Full repository method tests (GetRegions, GetSpots, GetLocationInfo, GetCoordinates)
// would require DynamoDB client mocking or Localstack integration tests.
// These tests focus on constructor validation that can be tested without AWS SDK dependencies.
