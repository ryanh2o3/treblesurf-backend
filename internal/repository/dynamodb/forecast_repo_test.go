package dynamodb

import (
	"testing"
)

func TestNewForecastRepo(t *testing.T) {
	repo := NewForecastRepo(nil, "TestTable")
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != "TestTable" {
		t.Fatalf("expected table name TestTable, got %s", repo.tableName)
	}
}
