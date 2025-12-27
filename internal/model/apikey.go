// Package model provides data models used throughout the application.
package model

import "time"

// APIKey represents an API key in the system
type APIKey struct {
	KeyID       string    `json:"key_id" dynamodbav:"key_id"`
	KeyValue    string    `json:"key_value" dynamodbav:"key_value"`
	Description string    `json:"description" dynamodbav:"description"`
	CreatedBy   string    `json:"created_by" dynamodbav:"created_by"`
	CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
	ExpiresAt   time.Time `json:"expires_at" dynamodbav:"expires_at"`
	Scopes      []string  `json:"scopes" dynamodbav:"scopes"`
}
