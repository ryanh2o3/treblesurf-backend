package model

import "time"

type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	JSON      string    `json:"json_data"`
	TTL       int64     `json:"ttl"`
}
