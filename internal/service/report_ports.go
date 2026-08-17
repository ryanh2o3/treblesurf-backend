package service

import (
	"context"

	"treblesurf-backend/internal/model"
)

// UserByEmail loads a user by email for report flows.
type UserByEmail interface {
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

// SpotNotificationBroadcaster notifies subscribers for a spot (e.g. WebSocket).
type SpotNotificationBroadcaster interface {
	GetSubscribersBySpot(ctx context.Context, spotIdentifier string) ([]string, error)
	BroadcastToUsers(ctx context.Context, userIDs []string, message interface{}) error
}
