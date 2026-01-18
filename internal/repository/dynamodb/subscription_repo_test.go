package dynamodb

import (
	"testing"
)

const subscriptionTableName = "TestSubscriptionsTable"

func TestNewSpotSubscriptionRepo(t *testing.T) {
	repo := NewSpotSubscriptionRepo(nil, subscriptionTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != subscriptionTableName {
		t.Fatalf("expected table name %s, got %s", subscriptionTableName, repo.tableName)
	}
}

// Note: SpotSubscriptionRepo uses DynamoDB PutItem directly (no item conversions),
// so model/item conversion tests are not applicable here.
// Full repository method tests (Save, GetSubscribersBySpot, Delete) would require
// DynamoDB client mocking or Localstack integration tests.
// These tests focus on constructor validation that can be tested without AWS SDK dependencies.
