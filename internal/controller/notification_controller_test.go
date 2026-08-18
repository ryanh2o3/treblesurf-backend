package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func setupNotificationController(t *testing.T, tokens *mockrepo.DeviceTokenRepo, alerts *mockrepo.SpotAlertRepo) *NotificationController {
	t.Helper()
	notify, err := service.NewNotificationService(tokens, alerts, &mockrepo.SwellPredictionRepo{}, service.NoopPushSender{})
	if err != nil {
		t.Fatalf("NewNotificationService: %v", err)
	}
	users, err := service.NewUserService(&mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, UUID: "user-uuid-1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewUserService: %v", err)
	}
	return NewNotificationController(notify, users)
}

func TestNotificationController_PutDeviceToken(t *testing.T) {
	saved := false
	controller := setupNotificationController(t, &mockrepo.DeviceTokenRepo{
		SaveFn: func(_ context.Context, token *model.DeviceToken) error {
			saved = true
			if token.Token != "abc123" || token.Environment != model.DeviceEnvironmentSandbox {
				t.Fatalf("unexpected token %+v", token)
			}
			return nil
		},
	}, &mockrepo.SpotAlertRepo{})

	body, _ := json.Marshal(model.DeviceTokenRequest{Token: "abc123", Environment: "sandbox"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/notification/device-token", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", testUserEmail)

	controller.PutDeviceToken(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body %s", w.Code, w.Body.String())
	}
	if !saved {
		t.Fatal("expected token to be saved")
	}
}

func TestNotificationController_PutDeviceToken_Unauthorized(t *testing.T) {
	controller := setupNotificationController(t, &mockrepo.DeviceTokenRepo{}, &mockrepo.SpotAlertRepo{})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/notification/device-token", http.NoBody)
	controller.PutDeviceToken(c)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestNotificationController_GetPreferences(t *testing.T) {
	controller := setupNotificationController(t, &mockrepo.DeviceTokenRepo{}, &mockrepo.SpotAlertRepo{
		GetByUserFn: func(_ context.Context, userUUID string) ([]*model.SpotAlertSubscription, error) {
			if userUUID != "user-uuid-1" {
				t.Fatalf("unexpected uuid %s", userUUID)
			}
			return []*model.SpotAlertSubscription{{
				Country:         "Ireland",
				Region:          "Donegal",
				Spot:            "Ballyhiernan",
				ReportsEnabled:  true,
				GoodSurfEnabled: false,
			}}, nil
		},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/notification/preferences", http.NoBody)
	c.Set("email", testUserEmail)
	controller.GetPreferences(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp model.NotificationPreferencesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Spots) != 1 || !resp.Spots[0].ReportsEnabled {
		t.Fatalf("unexpected prefs %+v", resp)
	}
}

func TestNotificationController_PutSpotAlert_MissingQuery(t *testing.T) {
	controller := setupNotificationController(t, &mockrepo.DeviceTokenRepo{}, &mockrepo.SpotAlertRepo{})
	body, _ := json.Marshal(model.SpotAlertRequest{ReportsEnabled: true})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/notification/spot", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("email", testUserEmail)
	controller.PutSpotAlert(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
