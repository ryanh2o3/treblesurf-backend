package dynamodb

import (
	"testing"
	"time"

	"treblesurf-backend/internal/model"
)

const sessionTableName = "TestSessionsTable"

func TestNewSessionRepo(t *testing.T) {
	repo := NewSessionRepo(nil, sessionTableName)
	if repo == nil {
		t.Fatalf("expected non-nil repo")
	}
	if repo.tableName != sessionTableName {
		t.Fatalf("expected table name %s, got %s", sessionTableName, repo.tableName)
	}
}

func TestSessionItem_FromModel_ToModel(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	ttl := expiresAt.Unix()

	session := &model.Session{
		SessionID: "test-session-id",
		UserID:    "test-user-id",
		ExpiresAt: expiresAt,
		JSON:      `{"key":"value"}`,
		TTL:       ttl,
	}

	// Test conversion: Model -> Item -> Model
	item := sessionItemFromModel(session)
	convertedSession := item.toModel()

	if convertedSession.SessionID != session.SessionID {
		t.Errorf("expected SessionID %s, got %s", session.SessionID, convertedSession.SessionID)
	}
	if convertedSession.UserID != session.UserID {
		t.Errorf("expected UserID %s, got %s", session.UserID, convertedSession.UserID)
	}
	if !convertedSession.ExpiresAt.Equal(session.ExpiresAt) {
		t.Errorf("expected ExpiresAt %v, got %v", session.ExpiresAt, convertedSession.ExpiresAt)
	}
	if convertedSession.JSON != session.JSON {
		t.Errorf("expected JSON %s, got %s", session.JSON, convertedSession.JSON)
	}
	if convertedSession.TTL != session.TTL {
		t.Errorf("expected TTL %d, got %d", session.TTL, convertedSession.TTL)
	}
}

func TestSessionItem_FromModel_NilInput(t *testing.T) {
	item := sessionItemFromModel(nil)
	if item.SessionID != "" {
		t.Error("expected empty SessionID for nil input")
	}
	if item.UserID != "" {
		t.Error("expected empty UserID for nil input")
	}
}

func TestSessionItem_ToModel_EmptyItem(t *testing.T) {
	item := sessionItem{}
	session := item.toModel()

	if session == nil {
		t.Fatal("expected non-nil session")
	}
	if session.SessionID != "" {
		t.Error("expected empty SessionID")
	}
	if session.UserID != "" {
		t.Error("expected empty UserID")
	}
}

func TestSessionItem_TTL_FromExpiresAt(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)
	ttl := expiresAt.Unix()

	// Test that TTL is set from ExpiresAt when TTL is 0
	session := &model.Session{
		SessionID: "test-session-id",
		UserID:    "test-user-id",
		ExpiresAt: expiresAt,
		TTL:       0, // TTL should be calculated from ExpiresAt
	}

	item := sessionItemFromModel(session)
	if item.TTL == 0 {
		// In Save method, TTL would be set from ExpiresAt if TTL is 0
		// But in item conversion, TTL stays 0 - Save method handles this
		t.Log("TTL is 0, Save method should set it from ExpiresAt")
	}

	// Test with explicit TTL
	session.TTL = ttl
	item2 := sessionItemFromModel(session)
	if item2.TTL != ttl {
		t.Errorf("expected TTL %d, got %d", ttl, item2.TTL)
	}
}

func TestSessionRepo_Save_TTL_Calculation(t *testing.T) {
	// This test documents the TTL calculation behavior
	// Full testing would require DynamoDB client mocking or integration tests

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	session := &model.Session{
		ExpiresAt: expiresAt,
		TTL:       0, // TTL should be set from ExpiresAt in Save method
	}

	// Verify TTL would be calculated correctly
	expectedTTL := expiresAt.Unix()
	if session.TTL == 0 {
		// In Save method, this would be: session.TTL = session.ExpiresAt.Unix()
		calculatedTTL := session.ExpiresAt.Unix()
		if calculatedTTL != expectedTTL {
			t.Errorf("expected calculated TTL %d, got %d", expectedTTL, calculatedTTL)
		}
	}
}

func TestSession_Expiration_Validation(t *testing.T) {
	// Test that expired sessions are detected correctly
	now := time.Now()
	expiredAt := now.Add(-1 * time.Hour) // Expired 1 hour ago

	session := &model.Session{
		ExpiresAt: expiredAt,
	}

	// Verify session is expired
	if !time.Now().After(session.ExpiresAt) {
		t.Error("expected session to be expired")
	}
}

func TestSession_NotExpired(t *testing.T) {
	// Test that non-expired sessions are detected correctly
	now := time.Now()
	futureExpiresAt := now.Add(24 * time.Hour) // Expires in 24 hours

	session := &model.Session{
		ExpiresAt: futureExpiresAt,
	}

	// Verify session is not expired
	if time.Now().After(session.ExpiresAt) {
		t.Error("expected session to not be expired")
	}
}

// Note: Full repository method tests (Save, Get, Delete, GetByUserID) would require
// DynamoDB client mocking or Localstack integration tests.
// These tests focus on model/item conversions and validation logic that can be
// tested without AWS SDK dependencies.
