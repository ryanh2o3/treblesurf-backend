package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

type recordingSender struct {
	sent []model.PushPayload
	to   []string
}

func (r *recordingSender) Send(_ context.Context, _, deviceToken string, push model.PushPayload) error {
	r.sent = append(r.sent, push)
	r.to = append(r.to, deviceToken)
	return nil
}

func testNotificationService(t *testing.T, tokens *mockrepo.DeviceTokenRepo, alerts *mockrepo.SpotAlertRepo, predictions *mockrepo.SwellPredictionRepo, sender PushSender) *NotificationService {
	t.Helper()
	svc, err := NewNotificationService(tokens, alerts, predictions, sender)
	if err != nil {
		t.Fatalf("NewNotificationService: %v", err)
	}
	svc.now = func() time.Time {
		return time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	}
	return svc
}

func TestEvaluateGoodSurf_MeetsThresholds(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	arrival := now.Add(3 * time.Hour)
	predictions := []model.SwellPrediction{{
		SpotID:            "Ireland#Donegal#Rossnowlagh",
		ForecastTimestamp: "1",
		Data: map[string]interface{}{
			"direction_quality": 0.85,
			"surf_size":         1.2,
			"confidence":        0.5,
			"arrival_time":      arrival.Format(time.RFC3339),
		},
	}}

	hit, key, body := EvaluateGoodSurf(predictions, now, "")
	if !hit {
		t.Fatal("expected a good-surf hit")
	}
	if key != arrival.Format(time.RFC3339) {
		t.Fatalf("unexpected key %q", key)
	}
	if !strings.HasPrefix(body, "Good surf predicted at Rossnowlagh around ") {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestEvaluateGoodSurf_DedupesArrival(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	arrival := now.Add(2 * time.Hour).Format(time.RFC3339)
	predictions := []model.SwellPrediction{{
		SpotID: "Ireland#Donegal#Rossnowlagh",
		Data: map[string]interface{}{
			"direction_quality": 0.9,
			"surf_size":         1.0,
			"confidence":        0.6,
			"arrival_time":      arrival,
		},
	}}

	hit, _, _ := EvaluateGoodSurf(predictions, now, arrival)
	if hit {
		t.Fatal("expected deduped arrival to be skipped")
	}
}

func TestEvaluateGoodSurf_RejectsLowQuality(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	predictions := []model.SwellPrediction{{
		SpotID: "Ireland#Donegal#Rossnowlagh",
		Data: map[string]interface{}{
			"direction_quality": 0.5,
			"surf_size":         1.5,
			"confidence":        0.9,
			"arrival_time":      now.Add(2 * time.Hour).Format(time.RFC3339),
		},
	}}
	hit, _, _ := EvaluateGoodSurf(predictions, now, "")
	if hit {
		t.Fatal("expected low direction quality to be rejected")
	}
}

func TestNotifyNewReport_SkipsReporter(t *testing.T) {
	sender := &recordingSender{}
	alerts := &mockrepo.SpotAlertRepo{
		GetBySpotFn: func(_ context.Context, spotID string) ([]*model.SpotAlertSubscription, error) {
			if spotID != "Ireland#Donegal#Ballyhiernan" {
				t.Fatalf("unexpected spot id %s", spotID)
			}
			return []*model.SpotAlertSubscription{
				{UserUUID: "reporter-1", ReportsEnabled: true, Spot: "Ballyhiernan"},
				{UserUUID: "watcher-2", ReportsEnabled: true, Spot: "Ballyhiernan", Country: "Ireland", Region: "Donegal"},
			}, nil
		},
	}
	tokens := &mockrepo.DeviceTokenRepo{
		GetByUserFn: func(_ context.Context, userUUID string) ([]*model.DeviceToken, error) {
			return []*model.DeviceToken{{
				UserUUID:    userUUID,
				Token:       "token-" + userUUID,
				Environment: model.DeviceEnvironmentSandbox,
			}}, nil
		},
	}
	svc := testNotificationService(t, tokens, alerts, &mockrepo.SwellPredictionRepo{}, sender)
	svc.NotifyNewReport(context.Background(), "Ireland", "Donegal", "Ballyhiernan", "reporter-1", "chest-shoulder", "good")

	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 push, got %d", len(sender.sent))
	}
	if sender.to[0] != "token-watcher-2" {
		t.Fatalf("expected watcher token, got %s", sender.to[0])
	}
	if sender.sent[0].Body != "New report at Ballyhiernan — chest-shoulder, good" {
		t.Fatalf("unexpected body %q", sender.sent[0].Body)
	}
}

func TestNotifyNewReport_SkipsReportsDisabled(t *testing.T) {
	sender := &recordingSender{}
	alerts := &mockrepo.SpotAlertRepo{
		GetBySpotFn: func(_ context.Context, _ string) ([]*model.SpotAlertSubscription, error) {
			return []*model.SpotAlertSubscription{
				{UserUUID: "watcher-2", ReportsEnabled: false, GoodSurfEnabled: true},
			}, nil
		},
	}
	svc := testNotificationService(t, &mockrepo.DeviceTokenRepo{}, alerts, &mockrepo.SwellPredictionRepo{}, sender)
	svc.NotifyNewReport(context.Background(), "Ireland", "Donegal", "Ballyhiernan", "reporter-1", "head-high", "excellent")
	if len(sender.sent) != 0 {
		t.Fatalf("expected no pushes, got %d", len(sender.sent))
	}
}

func TestUpsertSpotAlert_DeletesWhenBothOff(t *testing.T) {
	deleted := false
	alerts := &mockrepo.SpotAlertRepo{
		DeleteFn: func(_ context.Context, spotID, userUUID string) error {
			deleted = true
			if spotID != "Ireland#Donegal#Marble Hill" || userUUID != "user-1" {
				t.Fatalf("unexpected delete %s %s", spotID, userUUID)
			}
			return nil
		},
	}
	svc := testNotificationService(t, &mockrepo.DeviceTokenRepo{}, alerts, &mockrepo.SwellPredictionRepo{}, NoopPushSender{})
	if err := svc.UpsertSpotAlert(context.Background(), "user-1", "Ireland", "Donegal", "Marble Hill", false, false); err != nil {
		t.Fatalf("UpsertSpotAlert: %v", err)
	}
	if !deleted {
		t.Fatal("expected subscription to be deleted")
	}
}

func TestRunGoodSurfAlerts_SendsAndDedupes(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	arrival := now.Add(4 * time.Hour)
	updatedKey := ""
	sender := &recordingSender{}
	alerts := &mockrepo.SpotAlertRepo{
		ListGoodSurfEnabledFn: func(_ context.Context) ([]*model.SpotAlertSubscription, error) {
			return []*model.SpotAlertSubscription{{
				SpotID:          "Ireland#Donegal#Rossnowlagh",
				UserUUID:        "user-1",
				Country:         "Ireland",
				Region:          "Donegal",
				Spot:            "Rossnowlagh",
				GoodSurfEnabled: true,
			}}, nil
		},
		UpdateLastNotifiedKeyFn: func(_ context.Context, _, _, key string) error {
			updatedKey = key
			return nil
		},
	}
	predictions := &mockrepo.SwellPredictionRepo{
		GetSpotPredictionRangeFn: func(_ context.Context, _ string, _, _ time.Time) ([]model.SwellPrediction, error) {
			return []model.SwellPrediction{{
				SpotID: "Ireland#Donegal#Rossnowlagh",
				Data: map[string]interface{}{
					"direction_quality": 0.9,
					"surf_size":         1.1,
					"confidence":        0.7,
					"arrival_time":      arrival.Format(time.RFC3339),
				},
			}}, nil
		},
	}
	tokens := &mockrepo.DeviceTokenRepo{
		GetByUserFn: func(_ context.Context, userUUID string) ([]*model.DeviceToken, error) {
			return []*model.DeviceToken{{UserUUID: userUUID, Token: "tok", Environment: "sandbox"}}, nil
		},
	}
	svc := testNotificationService(t, tokens, alerts, predictions, sender)
	if err := svc.RunGoodSurfAlerts(context.Background()); err != nil {
		t.Fatalf("RunGoodSurfAlerts: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 push, got %d", len(sender.sent))
	}
	if updatedKey != arrival.Format(time.RFC3339) {
		t.Fatalf("expected dedupe key to be stored, got %q", updatedKey)
	}
}

func TestRegisterDeviceToken_Invalid(t *testing.T) {
	svc := testNotificationService(t, &mockrepo.DeviceTokenRepo{}, &mockrepo.SpotAlertRepo{}, &mockrepo.SwellPredictionRepo{}, NoopPushSender{})
	err := svc.RegisterDeviceToken(context.Background(), "user", "  ", "sandbox")
	if !errors.Is(err, repository.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestDeleteUserData(t *testing.T) {
	tokensDeleted := false
	alertsDeleted := false
	tokens := &mockrepo.DeviceTokenRepo{
		DeleteByUserFn: func(_ context.Context, userUUID string) error {
			tokensDeleted = true
			if userUUID != "user-1" {
				t.Fatalf("unexpected uuid %s", userUUID)
			}
			return nil
		},
	}
	alerts := &mockrepo.SpotAlertRepo{
		DeleteByUserFn: func(_ context.Context, userUUID string) error {
			alertsDeleted = true
			if userUUID != "user-1" {
				t.Fatalf("unexpected uuid %s", userUUID)
			}
			return nil
		},
	}
	svc := testNotificationService(t, tokens, alerts, &mockrepo.SwellPredictionRepo{}, NoopPushSender{})
	if err := svc.DeleteUserData(context.Background(), "user-1"); err != nil {
		t.Fatalf("DeleteUserData: %v", err)
	}
	if !tokensDeleted || !alertsDeleted {
		t.Fatal("expected tokens and alerts to be deleted")
	}
}
