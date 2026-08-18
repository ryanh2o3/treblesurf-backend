package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

const (
	goodSurfDirectionQuality = 0.8
	goodSurfMinSize          = 0.8
	goodSurfMinConfidence    = 0.4
	goodSurfHorizon          = 24 * time.Hour
)

// NotificationService manages device tokens, spot watches, and APNs fan-out.
type NotificationService struct {
	tokens      repository.DeviceTokenRepository
	alerts      repository.SpotAlertRepository
	predictions repository.SwellPredictionRepository
	sender      PushSender
	now         func() time.Time
}

func NewNotificationService(
	tokens repository.DeviceTokenRepository,
	alerts repository.SpotAlertRepository,
	predictions repository.SwellPredictionRepository,
	sender PushSender,
) (*NotificationService, error) {
	if tokens == nil {
		return nil, fmt.Errorf("device token repository is required")
	}
	if alerts == nil {
		return nil, fmt.Errorf("spot alert repository is required")
	}
	if sender == nil {
		sender = NoopPushSender{}
	}
	return &NotificationService{
		tokens:      tokens,
		alerts:      alerts,
		predictions: predictions,
		sender:      sender,
		now:         func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *NotificationService) RegisterDeviceToken(ctx context.Context, userUUID, token, environment string) error {
	token = strings.TrimSpace(token)
	if userUUID == "" || token == "" {
		return fmt.Errorf("%w: token is required", repository.ErrInvalidInput)
	}
	env := normalizeDeviceEnvironment(environment)
	return s.tokens.Save(ctx, &model.DeviceToken{
		UserUUID:    userUUID,
		Token:       token,
		Platform:    "ios",
		Environment: env,
		UpdatedAt:   s.now().Format(time.RFC3339),
	})
}

func (s *NotificationService) UnregisterDeviceToken(ctx context.Context, userUUID, token string) error {
	token = strings.TrimSpace(token)
	if userUUID == "" || token == "" {
		return fmt.Errorf("%w: token is required", repository.ErrInvalidInput)
	}
	return s.tokens.Delete(ctx, userUUID, token)
}

func (s *NotificationService) GetPreferences(ctx context.Context, userUUID string) (*model.NotificationPreferencesResponse, error) {
	subs, err := s.alerts.GetByUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	spots := make([]model.SpotAlertPreference, 0, len(subs))
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		spots = append(spots, model.SpotAlertPreference{
			Country:         sub.Country,
			Region:          sub.Region,
			Spot:            sub.Spot,
			ReportsEnabled:  sub.ReportsEnabled,
			GoodSurfEnabled: sub.GoodSurfEnabled,
		})
	}
	return &model.NotificationPreferencesResponse{Spots: spots}, nil
}

func (s *NotificationService) UpsertSpotAlert(
	ctx context.Context,
	userUUID, country, region, spot string,
	reportsEnabled, goodSurfEnabled bool,
) error {
	country, region, spot, err := normalizeSpotParts(country, region, spot)
	if err != nil {
		return err
	}
	if !reportsEnabled && !goodSurfEnabled {
		return s.alerts.Delete(ctx, AlertSpotID(country, region, spot), userUUID)
	}

	spotID := AlertSpotID(country, region, spot)
	existing, err := s.alerts.Get(ctx, spotID, userUUID)
	lastKey := ""
	if err == nil && existing != nil {
		lastKey = existing.LastGoodSurfNotifiedKey
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	return s.alerts.Save(ctx, &model.SpotAlertSubscription{
		SpotID:                  spotID,
		UserUUID:                userUUID,
		Country:                 country,
		Region:                  region,
		Spot:                    spot,
		ReportsEnabled:          reportsEnabled,
		GoodSurfEnabled:         goodSurfEnabled,
		LastGoodSurfNotifiedKey: lastKey,
		UpdatedAt:               s.now().Format(time.RFC3339),
	})
}

func (s *NotificationService) DeleteSpotAlert(ctx context.Context, userUUID, country, region, spot string) error {
	country, region, spot, err := normalizeSpotParts(country, region, spot)
	if err != nil {
		return err
	}
	return s.alerts.Delete(ctx, AlertSpotID(country, region, spot), userUUID)
}

func (s *NotificationService) DeleteUserData(ctx context.Context, userUUID string) error {
	if userUUID == "" {
		return nil
	}
	if err := s.tokens.DeleteByUser(ctx, userUUID); err != nil {
		return fmt.Errorf("deleting device tokens: %w", err)
	}
	if err := s.alerts.DeleteByUser(ctx, userUUID); err != nil {
		return fmt.Errorf("deleting spot alerts: %w", err)
	}
	return nil
}

// NotifyNewReport fans out an APNs alert to watchers of the spot, skipping the reporter.
func (s *NotificationService) NotifyNewReport(
	ctx context.Context,
	country, region, spot, reporterUUID, surfSize, quality string,
) {
	if s == nil {
		return
	}
	country = strings.TrimSpace(country)
	region = strings.TrimSpace(region)
	spot = strings.TrimSpace(spot)
	if country == "" || region == "" || spot == "" {
		return
	}

	subs, err := s.alerts.GetBySpot(ctx, AlertSpotID(country, region, spot))
	if err != nil {
		slog.Warn("failed to load report watchers", slog.Any("error", err))
		return
	}

	body := formatNewReportBody(spot, surfSize, quality)
	payload := model.PushPayload{
		Title:   fmt.Sprintf("New report at %s", spot),
		Body:    body,
		Type:    model.PushTypeNewReport,
		Country: country,
		Region:  region,
		Spot:    spot,
	}

	for _, sub := range subs {
		if sub == nil || !sub.ReportsEnabled || sub.UserUUID == "" || sub.UserUUID == reporterUUID {
			continue
		}
		s.sendToUser(ctx, sub.UserUUID, payload)
	}
}

// RunGoodSurfAlerts evaluates swell predictions for watched spots and sends at most one push per arrival window.
func (s *NotificationService) RunGoodSurfAlerts(ctx context.Context) error {
	if s.predictions == nil {
		return fmt.Errorf("swell prediction repository is required")
	}
	subs, err := s.alerts.ListGoodSurfEnabled(ctx)
	if err != nil {
		return err
	}

	now := s.now()
	start := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	end := now.Add(goodSurfHorizon)

	for _, sub := range subs {
		if sub == nil || !sub.GoodSurfEnabled {
			continue
		}
		predictions, err := s.predictions.GetSpotPredictionRange(ctx, sub.SpotID, start, end)
		if err != nil {
			slog.Warn("failed to load swell predictions for good-surf alert",
				slog.String("spot_id", sub.SpotID),
				slog.Any("error", err),
			)
			continue
		}
		hit, key, body := EvaluateGoodSurf(predictions, now, sub.LastGoodSurfNotifiedKey)
		if !hit {
			continue
		}
		payload := model.PushPayload{
			Title:   "Good surf predicted",
			Body:    body,
			Type:    model.PushTypeGoodSurf,
			Country: sub.Country,
			Region:  sub.Region,
			Spot:    sub.Spot,
		}
		s.sendToUser(ctx, sub.UserUUID, payload)
		if err := s.alerts.UpdateLastNotifiedKey(ctx, sub.SpotID, sub.UserUUID, key); err != nil {
			slog.Warn("failed to store good-surf dedupe key",
				slog.String("spot_id", sub.SpotID),
				slog.Any("error", err),
			)
		}
	}
	return nil
}

func (s *NotificationService) sendToUser(ctx context.Context, userUUID string, payload model.PushPayload) {
	tokens, err := s.tokens.GetByUser(ctx, userUUID)
	if err != nil {
		slog.Warn("failed to load device tokens", slog.String("user_uuid", userUUID), slog.Any("error", err))
		return
	}
	for _, token := range tokens {
		if token == nil || token.Token == "" {
			continue
		}
		if err := s.sender.Send(ctx, token.Environment, token.Token, payload); err != nil {
			if errors.Is(err, ErrInvalidDeviceToken) {
				_ = s.tokens.Delete(ctx, userUUID, token.Token)
				continue
			}
			slog.Warn("failed to send push notification",
				slog.String("user_uuid", userUUID),
				slog.Any("error", err),
			)
		}
	}
}

func AlertSpotID(country, region, spot string) string {
	return fmt.Sprintf("%s#%s#%s", country, region, spot)
}

func normalizeSpotParts(country, region, spot string) (string, string, string, error) {
	country = strings.TrimSpace(country)
	region = strings.TrimSpace(region)
	spot = strings.TrimSpace(spot)
	if country == "" || region == "" || spot == "" {
		return "", "", "", fmt.Errorf("%w: country, region, and spot are required", repository.ErrInvalidInput)
	}
	return country, region, spot, nil
}

func normalizeDeviceEnvironment(environment string) string {
	if strings.EqualFold(strings.TrimSpace(environment), model.DeviceEnvironmentProduction) {
		return model.DeviceEnvironmentProduction
	}
	return model.DeviceEnvironmentSandbox
}

func formatNewReportBody(spot, surfSize, quality string) string {
	parts := make([]string, 0, 2)
	if s := strings.TrimSpace(surfSize); s != "" {
		parts = append(parts, s)
	}
	if q := strings.TrimSpace(quality); q != "" {
		parts = append(parts, q)
	}
	if len(parts) == 0 {
		return fmt.Sprintf("New report at %s", spot)
	}
	return fmt.Sprintf("New report at %s — %s", spot, strings.Join(parts, ", "))
}
