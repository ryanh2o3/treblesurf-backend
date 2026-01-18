package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

const websocketTableName = "TestWebSocketConnectionsTable"

func TestNewWebSocketRepo(t *testing.T) {
	repo := NewWebSocketRepo(nil, websocketTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != websocketTableName {
		t.Fatalf("expected table name %s, got %s", websocketTableName, repo.tableName)
	}
}

func TestConnectionItem_FromModel_ToModel(t *testing.T) {
	now := time.Now()
	connection := &model.ConnectionInfo{
		ConnectionID: "test-connection-id-123",
		UserID:       "test@example.com",
		ConnectedAt:  now,
		LastActive:   now,
		UserAgent:    "Mozilla/5.0",
		IPAddress:    "192.168.1.1",
		CurrentSpot:  "Ireland/Donegal/Bundoran",
		TTL:          now.Add(24 * time.Hour).Unix(),
	}

	// Test conversion: Model -> Item -> Model
	item := connectionItemFromModel(connection)
	convertedConn := item.toModel()

	if convertedConn.ConnectionID != connection.ConnectionID {
		t.Errorf("expected ConnectionID %s, got %s", connection.ConnectionID, convertedConn.ConnectionID)
	}
	if convertedConn.UserID != connection.UserID {
		t.Errorf("expected UserID %s, got %s", connection.UserID, convertedConn.UserID)
	}
	if !convertedConn.ConnectedAt.Equal(connection.ConnectedAt) {
		t.Errorf("expected ConnectedAt %v, got %v", connection.ConnectedAt, convertedConn.ConnectedAt)
	}
	if !convertedConn.LastActive.Equal(connection.LastActive) {
		t.Errorf("expected LastActive %v, got %v", connection.LastActive, convertedConn.LastActive)
	}
	if convertedConn.UserAgent != connection.UserAgent {
		t.Errorf("expected UserAgent %s, got %s", connection.UserAgent, convertedConn.UserAgent)
	}
	if convertedConn.IPAddress != connection.IPAddress {
		t.Errorf("expected IPAddress %s, got %s", connection.IPAddress, convertedConn.IPAddress)
	}
	if convertedConn.CurrentSpot != connection.CurrentSpot {
		t.Errorf("expected CurrentSpot %s, got %s", connection.CurrentSpot, convertedConn.CurrentSpot)
	}
	if convertedConn.TTL != connection.TTL {
		t.Errorf("expected TTL %d, got %d", connection.TTL, convertedConn.TTL)
	}
}

func TestConnectionItem_FromModel_NilInput(t *testing.T) {
	item := connectionItemFromModel(nil)
	if item.ConnectionID != "" {
		t.Error("expected empty ConnectionID for nil input")
	}
	if item.UserID != "" {
		t.Error("expected empty UserID for nil input")
	}
	if !item.ConnectedAt.IsZero() {
		t.Error("expected zero ConnectedAt for nil input")
	}
}

func TestConnectionItem_ToModel_EmptyItem(t *testing.T) {
	item := connectionItem{}
	connection := item.toModel()

	if connection == nil {
		t.Fatal("expected non-nil connection")
	}
	if connection.ConnectionID != "" {
		t.Error("expected empty ConnectionID")
	}
	if connection.UserID != "" {
		t.Error("expected empty UserID")
	}
	if !connection.ConnectedAt.IsZero() {
		t.Error("expected zero ConnectedAt")
	}
}

func TestConnectionItem_TTL_Handling(t *testing.T) {
	now := time.Now()
	ttl := now.Add(24 * time.Hour).Unix()

	connection := &model.ConnectionInfo{
		ConnectionID: "test-connection-id",
		UserID:       "test@example.com",
		ConnectedAt:  now,
		TTL:          ttl,
	}

	item := connectionItemFromModel(connection)
	convertedConn := item.toModel()

	if convertedConn.TTL != ttl {
		t.Errorf("expected TTL %d, got %d", ttl, convertedConn.TTL)
	}

	// Test zero TTL
	connection.TTL = 0
	item = connectionItemFromModel(connection)
	convertedConn = item.toModel()

	if convertedConn.TTL != 0 {
		t.Errorf("expected TTL 0, got %d", convertedConn.TTL)
	}
}

// Note: Full repository method tests (SaveConnection, GetConnection, DeleteConnection,
// UpdateSpot, UpdateLastActive, GetConnectionsByUserIDs) would require DynamoDB client
// mocking or Localstack integration tests. These tests focus on model/item conversions
// and constructor validation that can be tested without AWS SDK dependencies.
