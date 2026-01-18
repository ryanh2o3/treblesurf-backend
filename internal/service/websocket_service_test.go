package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewWebSocketService(t *testing.T) {
	connections := &mockrepo.WebSocketRepo{}
	subscriptions := &mockrepo.SpotSubscriptionRepo{}
	jwtSecret := []byte("test-secret")
	endpoint := "test-endpoint"
	stage := "test-stage"

	service := NewWebSocketService(connections, subscriptions, jwtSecret, endpoint, stage)

	if service == nil {
		t.Fatal("expected service but got nil")
	}
	if service.connections != connections {
		t.Error("connections not set correctly")
	}
	if service.subscriptions != subscriptions {
		t.Error("subscriptions not set correctly")
	}
	if string(service.jwtSecret) != string(jwtSecret) {
		t.Error("jwtSecret not set correctly")
	}
	if service.endpoint != endpoint {
		t.Error("endpoint not set correctly")
	}
	if service.stage != stage {
		t.Error("stage not set correctly")
	}
}

func TestWebSocketService_ValidateWebSocketToken(t *testing.T) {
	jwtSecret := []byte("test-secret")
	service := NewWebSocketService(nil, nil, jwtSecret, "", "")

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		setupToken func() string
	}{
		{
			name: "valid token",
			setupToken: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub":   "websocket",
					"email": "test@example.com",
					"exp":   time.Now().Add(time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString(jwtSecret)
				return tokenString
			},
			wantErr: false,
		},
		{
			name:      "invalid token",
			token:     "invalid-token",
			wantErr:   true,
			setupToken: func() string { return "invalid-token" },
		},
		{
			name: "expired token",
			setupToken: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub":   "websocket",
					"email": "test@example.com",
					"exp":   time.Now().Add(-time.Hour).Unix(),
				})
				tokenString, _ := token.SignedString(jwtSecret)
				return tokenString
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.setupToken()
			token, err := service.ValidateWebSocketToken(tokenString)
			if tt.wantErr {
				if err == nil && token != nil && token.Valid {
					t.Error("expected error or invalid token")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if token == nil {
					t.Fatal("expected token but got nil")
				}
			}
		})
	}
}

func TestWebSocketService_GetEmailFromToken(t *testing.T) {
	jwtSecret := []byte("test-secret")
	service := NewWebSocketService(nil, nil, jwtSecret, "", "")

	tests := []struct {
		name      string
		setupToken func() *jwt.Token
		wantEmail string
		wantErr   bool
	}{
		{
			name: "valid token with email",
			setupToken: func() *jwt.Token {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub":   "websocket",
					"email": "test@example.com",
					"exp":   time.Now().Add(time.Hour).Unix(),
				})
				return token
			},
			wantEmail: "test@example.com",
			wantErr:  false,
		},
		{
			name: "token without email",
			setupToken: func() *jwt.Token {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub": "websocket",
					"exp": time.Now().Add(time.Hour).Unix(),
				})
				return token
			},
			wantErr: true,
		},
		{
			name: "token with wrong subject",
			setupToken: func() *jwt.Token {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub":   "wrong",
					"email": "test@example.com",
					"exp":   time.Now().Add(time.Hour).Unix(),
				})
				return token
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			email, err := service.GetEmailFromToken(token)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if email != tt.wantEmail {
					t.Errorf("email = %q, want %q", email, tt.wantEmail)
				}
			}
		})
	}
}

func TestWebSocketService_SaveConnection(t *testing.T) {
	ctx := context.Background()
	connections := &mockrepo.WebSocketRepo{
		SaveConnectionFn: func(_ context.Context, conn *model.ConnectionInfo) error {
			if conn.ConnectionID == "" {
				return errors.New("connection ID required")
			}
			return nil
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	conn := &model.ConnectionInfo{
		ConnectionID: "test-connection-id",
		UserID:       "test-user-id",
		ConnectedAt:  time.Now(),
		LastActive:   time.Now(),
	}

	err := service.SaveConnection(ctx, conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test with TTL set
	conn.TTL = 0
	err = service.SaveConnection(ctx, conn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn.TTL == 0 {
		t.Error("expected TTL to be set")
	}
}

func TestWebSocketService_DeleteConnection(t *testing.T) {
	ctx := context.Background()
	connections := &mockrepo.WebSocketRepo{
		DeleteConnectionFn: func(_ context.Context, connectionID string) error {
			if connectionID == "" {
				return errors.New("connection ID required")
			}
			return nil
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	err := service.DeleteConnection(ctx, "test-connection-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSocketService_GetConnection(t *testing.T) {
	ctx := context.Background()
	expectedConn := &model.ConnectionInfo{
		ConnectionID: "test-connection-id",
		UserID:       "test-user-id",
	}
	connections := &mockrepo.WebSocketRepo{
		GetConnectionFn: func(_ context.Context, connectionID string) (*model.ConnectionInfo, error) {
			if connectionID == "test-connection-id" {
				return expectedConn, nil
			}
			return nil, errors.New("not found")
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	conn, err := service.GetConnection(ctx, "test-connection-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn != expectedConn {
		t.Errorf("connection = %+v, want %+v", conn, expectedConn)
	}
}

func TestWebSocketService_UpdateConnectionLastActive(t *testing.T) {
	ctx := context.Background()
	connections := &mockrepo.WebSocketRepo{
		UpdateLastActiveFn: func(_ context.Context, connectionID string) error {
			if connectionID == "" {
				return errors.New("connection ID required")
			}
			return nil
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	err := service.UpdateConnectionLastActive(ctx, "test-connection-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSocketService_UpdateConnectionSpot(t *testing.T) {
	ctx := context.Background()
	connections := &mockrepo.WebSocketRepo{
		UpdateSpotFn: func(_ context.Context, connectionID, spot string) error {
			if connectionID == "" || spot == "" {
				return errors.New("connection ID and spot required")
			}
			return nil
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	err := service.UpdateConnectionSpot(ctx, "test-connection-id", "Ireland/Donegal/Bundoran")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSocketService_SaveSubscription(t *testing.T) {
	ctx := context.Background()
	subscriptions := &mockrepo.SpotSubscriptionRepo{
		SaveFn: func(_ context.Context, spotIdentifier, userID, connectionID string) error {
			if spotIdentifier == "" || userID == "" || connectionID == "" {
				return errors.New("all parameters required")
			}
			return nil
		},
	}
	service := NewWebSocketService(nil, subscriptions, []byte("secret"), "", "")

	err := service.SaveSubscription(ctx, "Ireland/Donegal/Bundoran", "test-user-id", "test-connection-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebSocketService_GetSubscribersBySpot(t *testing.T) {
	ctx := context.Background()
	expectedSubscribers := []string{"user1", "user2"}
	subscriptions := &mockrepo.SpotSubscriptionRepo{
		GetSubscribersBySpotFn: func(_ context.Context, spotIdentifier string) ([]string, error) {
			if spotIdentifier == "Ireland/Donegal/Bundoran" {
				return expectedSubscribers, nil
			}
			return nil, nil
		},
	}
	service := NewWebSocketService(nil, subscriptions, []byte("secret"), "", "")

	subscribers, err := service.GetSubscribersBySpot(ctx, "Ireland/Donegal/Bundoran")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subscribers) != len(expectedSubscribers) {
		t.Errorf("subscribers = %v, want %v", subscribers, expectedSubscribers)
	}
}

func TestWebSocketService_GetSubscribersBySpot_NilRepository(t *testing.T) {
	ctx := context.Background()
	service := NewWebSocketService(nil, nil, []byte("secret"), "", "")

	_, err := service.GetSubscribersBySpot(ctx, "Ireland/Donegal/Bundoran")
	if err == nil {
		t.Fatal("expected error for nil subscription repository")
	}
}

func TestWebSocketService_CreateSubscriptionResponse(t *testing.T) {
	service := NewWebSocketService(nil, nil, []byte("secret"), "", "")
	spotIdentifier := "Ireland/Donegal/Bundoran"

	response := service.CreateSubscriptionResponse(spotIdentifier)

	if response == nil {
		t.Fatal("expected response but got nil")
	}
	if response.Action != "subscribed" {
		t.Errorf("Action = %q, want %q", response.Action, "subscribed")
	}
	if response.Data == nil {
		t.Fatal("expected data but got nil")
	}
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if dataMap["spot_id"] != spotIdentifier {
		t.Errorf("spot_id = %v, want %q", dataMap["spot_id"], spotIdentifier)
	}
	if success, ok := dataMap["success"].(bool); !ok || !success {
		t.Error("expected success to be true")
	}
}

func TestWebSocketService_CreatePongResponse(t *testing.T) {
	service := NewWebSocketService(nil, nil, []byte("secret"), "", "")

	response := service.CreatePongResponse()

	if response == nil {
		t.Fatal("expected response but got nil")
	}
	if response.Action != "pong" {
		t.Errorf("Action = %q, want %q", response.Action, "pong")
	}
	if response.Data == nil {
		t.Fatal("expected data but got nil")
	}
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if _, ok := dataMap["time"].(string); !ok {
		t.Error("expected time field in pong response")
	}
}

func TestWebSocketService_GetConnectionsByUserIDs(t *testing.T) {
	ctx := context.Background()
	expectedConnections := []*model.ConnectionInfo{
		{ConnectionID: "conn1", UserID: "user1"},
		{ConnectionID: "conn2", UserID: "user1"},
	}
	connections := &mockrepo.WebSocketRepo{
		GetConnectionsByUserIDsFn: func(_ context.Context, userIDs []string) ([]*model.ConnectionInfo, error) {
			if len(userIDs) == 1 && userIDs[0] == "user1" {
				return expectedConnections, nil
			}
			return nil, nil
		},
	}
	service := NewWebSocketService(connections, nil, []byte("secret"), "", "")

	conns, err := service.GetConnectionsByUserIDs(ctx, []string{"user1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != len(expectedConnections) {
		t.Errorf("connections = %v, want %v", conns, expectedConnections)
	}
}
