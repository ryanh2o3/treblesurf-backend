package service

import (
	"strings"
	"testing"

	"treblesurf-backend/internal/config"
)

func TestNewPushSender_MissingCredentialsIsNoop(t *testing.T) {
	sender, err := NewPushSender(&config.Config{})
	if err != nil {
		t.Fatalf("NewPushSender: %v", err)
	}
	if _, ok := sender.(NoopPushSender); !ok {
		t.Fatalf("expected NoopPushSender, got %T", sender)
	}
}

func TestAPNSReason(t *testing.T) {
	if got := apnsReason([]byte(`{"reason":"BadDeviceToken"}`)); got != "BadDeviceToken" {
		t.Fatalf("got %q", got)
	}
}

func TestParseAPNSAuthKey_Invalid(t *testing.T) {
	_, err := parseAPNSAuthKey("not-a-key")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "PEM") {
		t.Fatalf("unexpected error %v", err)
	}
}
