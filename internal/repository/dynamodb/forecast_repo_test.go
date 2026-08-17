package dynamodb

import (
	"testing"
	"time"
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

func TestParseForecastTimestamp(t *testing.T) {
	t.Run("unix_only", func(t *testing.T) {
		ts, err := parseForecastTimestamp("1705312800")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts.Unix() != 1705312800 {
			t.Fatalf("expected unix 1705312800, got %d", ts.Unix())
		}
	})
	t.Run("unix_with_source", func(t *testing.T) {
		ts, err := parseForecastTimestamp("1705312800#stormglass")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts.Unix() != 1705312800 {
			t.Fatalf("expected unix 1705312800, got %d", ts.Unix())
		}
	})
	t.Run("unix_with_another_source", func(t *testing.T) {
		ts, err := parseForecastTimestamp("1705312800#weatherkit")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts.Unix() != 1705312800 {
			t.Fatalf("expected unix 1705312800, got %d", ts.Unix())
		}
	})
	t.Run("empty", func(t *testing.T) {
		_, err := parseForecastTimestamp("")
		if err == nil {
			t.Fatalf("expected error for empty timestamp")
		}
	})
}

func TestBaseTimestampFromForecast(t *testing.T) {
	// Ensure lexicographic order: "1705312800" < "1705312800#stormglass" < "1705312801"
	t0 := "1705312800"
	t1 := "1705312800#stormglass"
	t2 := "1705312800#weatherkit"
	t3 := "1705312801"
	if t0 >= t1 || t1 >= t2 || t2 >= t3 {
		t.Fatalf("sort key order broken")
	}
	ts, _ := parseForecastTimestamp(t1)
	if !ts.Equal(time.Unix(1705312800, 0).UTC()) {
		t.Fatalf("expected same time for ts#source")
	}
}
