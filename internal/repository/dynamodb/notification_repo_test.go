package dynamodb

import "testing"

func TestNewDeviceTokenRepo(t *testing.T) {
	repo := NewDeviceTokenRepo(nil, "DeviceTokens")
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
	if repo.tableName != "DeviceTokens" {
		t.Fatalf("expected table name DeviceTokens, got %s", repo.tableName)
	}
}

func TestNewSpotAlertRepo(t *testing.T) {
	repo := NewSpotAlertRepo(nil, "SpotAlertSubscriptions")
	if repo == nil {
		t.Fatal("expected non-nil repo")
	}
	if repo.tableName != "SpotAlertSubscriptions" {
		t.Fatalf("unexpected table name %s", repo.tableName)
	}
}
