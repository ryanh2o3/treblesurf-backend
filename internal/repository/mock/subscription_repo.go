package mock

import (
	"context"

	"treblesurf-backend/internal/repository"
)

var _ repository.SpotSubscriptionRepository = (*SpotSubscriptionRepo)(nil)

type SpotSubscriptionRepo struct {
	SaveFn func(ctx context.Context, spotIdentifier, userID, connectionID string) error
	GetSubscribersBySpotFn func(ctx context.Context, spotIdentifier string) ([]string, error)
}

func (m *SpotSubscriptionRepo) Save(ctx context.Context, spotIdentifier, userID, connectionID string) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, spotIdentifier, userID, connectionID)
	}
	return nil
}

func (m *SpotSubscriptionRepo) GetSubscribersBySpot(ctx context.Context, spotIdentifier string) ([]string, error) {
	if m.GetSubscribersBySpotFn != nil {
		return m.GetSubscribersBySpotFn(ctx, spotIdentifier)
	}
	return []string{}, nil
}
