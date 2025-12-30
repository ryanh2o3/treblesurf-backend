package model

import "time"

type Stream struct {
	ID              string    `json:"id" dynamodbav:"id"`
	UserEmail      string    `json:"user_email" dynamodbav:"user_email"`
	Country        string    `json:"country" dynamodbav:"country"`
	Region         string    `json:"region" dynamodbav:"region"`
	Spot           string    `json:"spot" dynamodbav:"spot"`
	Status         string    `json:"status" dynamodbav:"status"` // requested, active, completed, cancelled
	RequestedAt    time.Time `json:"requested_at" dynamodbav:"requested_at"`
	StartedAt      *time.Time `json:"started_at,omitempty" dynamodbav:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
	StreamURL      string    `json:"stream_url,omitempty" dynamodbav:"stream_url,omitempty"`
	PlaybackURL    string    `json:"playback_url,omitempty" dynamodbav:"playback_url,omitempty"`
	CreatedAt      time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

type StreamCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	Region          string `json:"region"`
	StreamName      string `json:"stream_name"`
}

type Snapshot struct {
	ID         string    `json:"id" dynamodbav:"id"`
	StreamID   string    `json:"stream_id" dynamodbav:"stream_id"`
	ImageKey   string    `json:"image_key" dynamodbav:"image_key"`
	Timestamp  time.Time `json:"timestamp" dynamodbav:"timestamp"`
	CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
}