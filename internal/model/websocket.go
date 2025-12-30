package model

import (
	"encoding/json"
	"time"
)

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

type WebSocketMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type SubscriptionRequest struct {
	Country string `json:"country"`
	Region  string `json:"region"`
	Spot    string `json:"spot"`
}

type WebSocketResponse struct {
	Data   interface{} `json:"data"`
	Action string      `json:"action"`
}

type SessionJSON struct {
	CreatedAt  time.Time `json:"created_at,omitempty"`
	LastActive time.Time `json:"last_active,omitempty"`
	CSRF       string    `json:"csrf"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}
