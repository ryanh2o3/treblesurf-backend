package dynamodb

import (
	"testing"
)

const forecastTableName = "TestTable"

func TestNewForecastRepo(t *testing.T) {
	repo := NewForecastRepo(nil, forecastTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != forecastTableName {
		t.Fatalf("expected table name TestTable, got %s", repo.tableName)
	}
}
