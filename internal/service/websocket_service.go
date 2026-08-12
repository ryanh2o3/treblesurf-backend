package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/golang-jwt/jwt/v5"
)

type WebSocketService struct {
	connections   repository.WebSocketRepository
	subscriptions repository.SpotSubscriptionRepository
	apiClientErr  error
	apiClient     *apigatewaymanagementapi.ApiGatewayManagementApi
	endpoint      string
	stage         string
	jwtSecret     []byte
	apiClientOnce sync.Once
}

func NewWebSocketService(
	connections repository.WebSocketRepository,
	subscriptions repository.SpotSubscriptionRepository,
	jwtSecret []byte,
	endpoint string,
	stage string,
) (*WebSocketService, error) {
	switch {
	case connections == nil:
		return nil, fmt.Errorf("websocket connections repository is required")
	case subscriptions == nil:
		return nil, fmt.Errorf("spot subscription repository is required")
	case len(jwtSecret) == 0:
		return nil, fmt.Errorf("JWT secret is required")
	}
	return &WebSocketService{
		connections:   connections,
		subscriptions: subscriptions,
		jwtSecret:     jwtSecret,
		endpoint:      endpoint,
		stage:         stage,
	}, nil
}

func (s *WebSocketService) ValidateWebSocketToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
}

// GetEmailFromToken extracts the email claim from a validated WebSocket token.
func (s *WebSocketService) GetEmailFromToken(token *jwt.Token) (string, error) {
	claims, okClaims := token.Claims.(jwt.MapClaims)
	if !okClaims {
		return "", fmt.Errorf("invalid token claims")
	}

	// Check subject is "websocket"
	if sub, okSub := claims["sub"].(string); !okSub || sub != "websocket" {
		return "", fmt.Errorf("invalid token subject")
	}

	email, ok := claims["email"].(string)
	if !ok || email == "" {
		return "", fmt.Errorf("email claim not found in token")
	}

	return email, nil
}

func (s *WebSocketService) SaveConnection(ctx context.Context, conn *model.ConnectionInfo) error {
	if conn.TTL == 0 {
		conn.TTL = time.Now().Add(24 * time.Hour).Unix()
	}
	return s.connections.SaveConnection(ctx, conn)
}

func (s *WebSocketService) DeleteConnection(ctx context.Context, connectionID string) error {
	return s.connections.DeleteConnection(ctx, connectionID)
}

func (s *WebSocketService) UpdateConnectionLastActive(ctx context.Context, connectionID string) error {
	return s.connections.UpdateLastActive(ctx, connectionID)
}

func (s *WebSocketService) GetConnection(ctx context.Context, connectionID string) (*model.ConnectionInfo, error) {
	return s.connections.GetConnection(ctx, connectionID)
}

func (s *WebSocketService) UpdateConnectionSpot(ctx context.Context, connectionID, spotIdentifier string) error {
	return s.connections.UpdateSpot(ctx, connectionID, spotIdentifier)
}

func (s *WebSocketService) SaveSubscription(ctx context.Context, spotIdentifier, userID, connectionID string) error {
	return s.subscriptions.Save(ctx, spotIdentifier, userID, connectionID)
}

func (s *WebSocketService) GetSubscribersBySpot(ctx context.Context, spotIdentifier string) ([]string, error) {
	if s.subscriptions == nil {
		return nil, fmt.Errorf("subscription repository not initialized")
	}
	return s.subscriptions.GetSubscribersBySpot(ctx, spotIdentifier)
}

// SendToConnection sends a message to a specific WebSocket client
func (s *WebSocketService) SendToConnection(connectionID, message string) error {
	client, err := s.apiGatewayClient()
	if err != nil {
		return err
	}

	_, err = client.PostToConnection(&apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         []byte(message),
	})

	return err
}

func (s *WebSocketService) GetSourceIP(headers map[string]string) string {
	// Try CloudFront-specific header first
	if ip, ok := headers["CloudFront-Viewer-Address"]; ok && ip != "" {
		return strings.Split(ip, ":")[0]
	}

	// Try standard headers
	if ip, ok := headers["X-Forwarded-For"]; ok && ip != "" {
		// X-Forwarded-For may contain multiple IPs, take the first one
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fallback to source IP from API Gateway
	if ip, ok := headers["X-Forwarded-For"]; ok && ip != "" {
		return ip
	}

	return "unknown"
}

func (s *WebSocketService) CreateSubscriptionResponse(spotIdentifier string) *model.WebSocketResponse {
	return &model.WebSocketResponse{
		Action: "subscribed",
		Data: map[string]interface{}{
			"spot_id": spotIdentifier,
			"success": true,
		},
	}
}

func (s *WebSocketService) CreatePongResponse() *model.WebSocketResponse {
	return &model.WebSocketResponse{
		Action: "pong",
		Data: map[string]interface{}{
			"time": time.Now().Format(time.RFC3339),
		},
	}
}

// BroadcastToUsers sends a message to multiple users via their WebSocket connections
func (s *WebSocketService) BroadcastToUsers(ctx context.Context, userIDs []string, message interface{}) error {
	// Get all connections for the given user IDs
	connections, err := s.GetConnectionsByUserIDs(ctx, userIDs)
	if err != nil {
		return fmt.Errorf("failed to get connections: %w", err)
	}

	// Send message to each connection
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	for _, conn := range connections {
		if err := s.SendToConnection(conn.ConnectionID, string(messageJSON)); err != nil {
			slog.Warn("failed to send message to connection", slog.String("connection_id", conn.ConnectionID), slog.Any("error", err))
			// Continue with other connections
		}
	}

	return nil
}

func (s *WebSocketService) GetConnectionsByUserIDs(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error) {
	return s.connections.GetConnectionsByUserIDs(ctx, userIDs)
}

func (s *WebSocketService) apiGatewayClient() (*apigatewaymanagementapi.ApiGatewayManagementApi, error) {
	s.apiClientOnce.Do(func() {
		if s.endpoint == "" {
			s.apiClientErr = fmt.Errorf("WEBSOCKET_API_ENDPOINT not configured")
			return
		}
		stage := s.stage
		if stage == "" {
			stage = "production"
		}

		sess, err := session.NewSession()
		if err != nil {
			s.apiClientErr = fmt.Errorf("failed to create AWS session: %w", err)
			return
		}

		apiEndpoint := fmt.Sprintf("https://%s/%s", s.endpoint, stage)
		s.apiClient = apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(apiEndpoint))
	})

	if s.apiClientErr != nil {
		return nil, s.apiClientErr
	}
	if s.apiClient == nil {
		return nil, fmt.Errorf("websocket api client not initialized")
	}
	return s.apiClient, nil
}
