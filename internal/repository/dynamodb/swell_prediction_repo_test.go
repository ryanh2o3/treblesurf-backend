package dynamodb

import (
	"testing"
)

const swellPredictionTableName = "TestSwellPredictionsTable"

func TestNewSwellPredictionRepo(t *testing.T) {
	repo := NewSwellPredictionRepo(nil, swellPredictionTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != swellPredictionTableName {
		t.Fatalf("expected table name %s, got %s", swellPredictionTableName, repo.tableName)
	}
}

// Note: SwellPredictionRepo uses dynamodbattribute.MarshalMap/UnmarshalMap directly
// with model.SwellPrediction (no item conversions), so model/item conversion tests
// are not applicable here.
// Full repository method tests (GetSpotPredictions, GetListSpotsPredictions,
// GetRegionPredictions, GetRecentPredictions, GetSpotPredictionsRange, Save)
// would require DynamoDB client mocking or Localstack integration tests.
// These tests focus on constructor validation that can be tested without AWS SDK dependencies.
