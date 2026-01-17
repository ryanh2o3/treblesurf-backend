package model

import "time"

type Session struct {
	SessionID string    `json:"session_id" dynamodbav:"session_id"`
	UserID    string    `json:"user_id" dynamodbav:"user_id"`
	ExpiresAt time.Time `json:"expires_at" dynamodbav:"expires_at"`
	JSON      string    `json:"json_data" dynamodbav:"json_data"`
	TTL       int64     `json:"ttl" dynamodbav:"ttl"`
}
