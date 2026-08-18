package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/model"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidDeviceToken is returned when APNs rejects a device token.
var ErrInvalidDeviceToken = errors.New("invalid device token")

const (
	apnsProductionHost = "https://api.push.apple.com"
	apnsSandboxHost    = "https://api.sandbox.push.apple.com"
)

// PushSender delivers APNs alerts.
type PushSender interface {
	Send(ctx context.Context, environment, deviceToken string, push model.PushPayload) error
}

// NoopPushSender drops notifications (used when APNs credentials are not configured).
type NoopPushSender struct{}

func (NoopPushSender) Send(context.Context, string, string, model.PushPayload) error {
	return nil
}

type apnsPushSender struct {
	httpClient *http.Client
	key        *ecdsa.PrivateKey
	keyID      string
	teamID     string
	topic      string

	mu        sync.Mutex
	cachedJWT string
	jwtExpiry time.Time
}

// NewPushSender builds an APNs client. Missing credentials yield a no-op sender.
func NewPushSender(cfg *config.Config) (PushSender, error) {
	if cfg == nil {
		return NoopPushSender{}, nil
	}
	keyP8 := strings.TrimSpace(cfg.APNS.KeyP8)
	keyID := strings.TrimSpace(cfg.APNS.KeyID)
	teamID := strings.TrimSpace(cfg.APNS.TeamID)
	topic := strings.TrimSpace(cfg.APNS.Topic)
	if topic == "" {
		topic = "treble.TrebleSurf"
	}
	if keyP8 == "" || keyID == "" || teamID == "" {
		slog.Warn("APNs credentials not configured; push notifications disabled")
		return NoopPushSender{}, nil
	}

	privateKey, err := parseAPNSAuthKey(keyP8)
	if err != nil {
		return nil, fmt.Errorf("parsing APNs auth key: %w", err)
	}

	return &apnsPushSender{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		key:        privateKey,
		keyID:      keyID,
		teamID:     teamID,
		topic:      topic,
	}, nil
}

func (s *apnsPushSender) Send(ctx context.Context, environment, deviceToken string, push model.PushPayload) error {
	if s == nil {
		return nil
	}
	host := apnsSandboxHost
	if strings.EqualFold(environment, model.DeviceEnvironmentProduction) {
		host = apnsProductionHost
	}

	body, err := json.Marshal(map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{
				"title": push.Title,
				"body":  push.Body,
			},
			"sound": "default",
		},
		"type":    push.Type,
		"country": push.Country,
		"region":  push.Region,
		"spot":    push.Spot,
	})
	if err != nil {
		return fmt.Errorf("encoding APNs payload: %w", err)
	}

	bearer, err := s.bearerToken()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/3/device/%s", host, deviceToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating APNs request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+bearer)
	req.Header.Set("apns-topic", s.topic)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending APNs notification: %w", err)
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(res.Body)

	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusNoContent {
		return nil
	}
	reason := apnsReason(respBody)
	if res.StatusCode == http.StatusGone || reason == "BadDeviceToken" || reason == "Unregistered" || reason == "DeviceTokenNotForTopic" {
		return fmt.Errorf("%w: %s", ErrInvalidDeviceToken, reason)
	}
	if reason == "" {
		reason = res.Status
	}
	return fmt.Errorf("APNs rejected notification: %s", reason)
}

func (s *apnsPushSender) bearerToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedJWT != "" && time.Now().Before(s.jwtExpiry) {
		return s.cachedJWT, nil
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": s.teamID,
		"iat": now.Unix(),
	})
	token.Header["kid"] = s.keyID
	signed, err := token.SignedString(s.key)
	if err != nil {
		return "", fmt.Errorf("signing APNs JWT: %w", err)
	}
	s.cachedJWT = signed
	s.jwtExpiry = now.Add(50 * time.Minute)
	return signed, nil
}

func parseAPNSAuthKey(pemData string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("no PEM block in APNs key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("APNs key is not an ECDSA private key")
	}
	return key, nil
}

func apnsReason(body []byte) string {
	var parsed struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return strings.TrimSpace(string(body))
	}
	return parsed.Reason
}
