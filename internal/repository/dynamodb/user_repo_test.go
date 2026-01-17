package dynamodb

import (
	"testing"
)

func TestNewUserRepo(t *testing.T) {
	repo := NewUserRepo(nil, "TestTable")
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != "TestTable" {
		t.Fatalf("expected table name TestTable, got %s", repo.tableName)
	}
}
