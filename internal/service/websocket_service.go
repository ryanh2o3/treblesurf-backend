package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/golang-jwt/jwt"
)

type WebSocketService struct {
	connections   repository.WebSocketRepository
	subscriptions repository.SpotSubscriptionRepository
	jwtSecret     []byte
}

func NewWebSocketService(
	connections repository.WebSocketRepository,
	subscriptions repository.SpotSubscriptionRepository,
	jwtSecret []byte,
) *WebSocketService {
	return &WebSocketService{
		connections:   connections,
		subscriptions: subscriptions,
		jwtSecret:     jwtSecret,
	}
}

func (s *WebSocketService) ValidateWebSocketToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
}

func (s *WebSocketService) SaveConnection(conn *model.ConnectionInfo) error {
	if conn.TTL == 0 {
		conn.TTL = time.Now().Add(24 * time.Hour).Unix()
	}
	return s.connections.SaveConnection(context.Background(), conn)
}

func (s *WebSocketService) DeleteConnection(connectionID string) error {
	return s.connections.DeleteConnection(context.Background(), connectionID)
}

func (s *WebSocketService) UpdateConnectionLastActive(connectionID string) error {
	return s.connections.UpdateLastActive(context.Background(), connectionID)
}

func (s *WebSocketService) GetConnection(connectionID string) (*model.ConnectionInfo, error) {
	return s.connections.GetConnection(context.Background(), connectionID)
}

func (s *WebSocketService) UpdateConnectionSpot(connectionID, spotIdentifier string) error {
	return s.connections.UpdateSpot(context.Background(), connectionID, spotIdentifier)
}

func (s *WebSocketService) SaveSubscription(spotIdentifier, userID, connectionID string) error {
	return s.subscriptions.Save(context.Background(), spotIdentifier, userID, connectionID)
}

// SendToConnection sends a message to a specific WebSocket client
func (s *WebSocketService) SendToConnection(connectionID, message string) error {
	// Get endpoint and stage from environment variables
	endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
	if endpoint == "" {
		return fmt.Errorf("WEBSOCKET_API_ENDPOINT not configured")
	}

	stage := os.Getenv("WEBSOCKET_API_STAGE")
	if stage == "" {
		stage = "production" // Default stage name
	}

	sess := session.Must(session.NewSession())

	// Create API Gateway Management API client
	apiEndpoint := fmt.Sprintf("https://%s/%s", endpoint, stage)
	client := apigatewaymanagementapi.New(sess, aws.NewConfig().WithEndpoint(apiEndpoint))

	_, err := client.PostToConnection(&apigatewaymanagementapi.PostToConnectionInput{
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
func (s *WebSocketService) BroadcastToUsers(userIDs []string, message interface{}) error {
	// Get all connections for the given user IDs
	connections, err := s.GetConnectionsByUserIDs(userIDs)
	if err != nil {
		return fmt.Errorf("failed to get connections: %v", err)
	}

	// Send message to each connection
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	for _, conn := range connections {
		if err := s.SendToConnection(conn.ConnectionID, string(messageJSON)); err != nil {
			log.Printf("Failed to send message to connection %s: %v", conn.ConnectionID, err)
			// Continue with other connections
		}
	}

	return nil
}

func (s *WebSocketService) GetConnectionsByUserIDs(userIDs []string) ([]*model.ConnectionInfo, error) {
	return s.connections.GetConnectionsByUserIDs(context.Background(), userIDs)
}
