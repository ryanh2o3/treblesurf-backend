package model

import (
	"encoding/json"
	"time"
)

// ConnectionInfo represents a WebSocket connection with user info
type ConnectionInfo struct {
	ConnectionID string    `json:"connection_id"`
	UserID       string    `json:"user_id"` // Email
	ConnectedAt  time.Time `json:"connected_at"`
	LastActive   time.Time `json:"last_active"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	CurrentSpot  string    `json:"current_spot,omitempty"`
	TTL          int64     `json:"ttl"`
}

// WebSocketMessage represents the structure of incoming messages
type WebSocketMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// SubscriptionRequest represents a spot subscription request
type SubscriptionRequest struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	Spot    string `json:"spot"`
}

// WebSocketResponse represents a response to send back to clients
type WebSocketResponse struct {
	Action string      `json:"action"`
	Data   interface{} `json:"data"`
}

// SessionJSON defines the structure for session data
type SessionJSON struct {
	CSRF       string    `json:"csrf"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	LastActive time.Time `json:"last_active,omitempty"`
}
