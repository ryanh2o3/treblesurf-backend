// Package model provides data models used throughout the application.
package model

import "time"

type APIKey struct {
	KeyID       string    `json:"key_id"`
	KeyValue    string    `json:"key_value"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scopes      []string  `json:"scopes"`
}
