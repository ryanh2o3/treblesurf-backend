package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.SpotAlertRepository = (*SpotAlertRepo)(nil)

type SpotAlertRepo struct {
	SaveFn                  func(ctx context.Context, sub *model.SpotAlertSubscription) error
	GetFn                   func(ctx context.Context, spotID, userUUID string) (*model.SpotAlertSubscription, error)
	DeleteFn                func(ctx context.Context, spotID, userUUID string) error
	GetByUserFn             func(ctx context.Context, userUUID string) ([]*model.SpotAlertSubscription, error)
	GetBySpotFn             func(ctx context.Context, spotID string) ([]*model.SpotAlertSubscription, error)
	ListGoodSurfEnabledFn   func(ctx context.Context) ([]*model.SpotAlertSubscription, error)
	DeleteByUserFn          func(ctx context.Context, userUUID string) error
	UpdateLastNotifiedKeyFn func(ctx context.Context, spotID, userUUID, key string) error
}

func (m *SpotAlertRepo) Save(ctx context.Context, sub *model.SpotAlertSubscription) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, sub)
	}
	return nil
}

func (m *SpotAlertRepo) Get(ctx context.Context, spotID, userUUID string) (*model.SpotAlertSubscription, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, spotID, userUUID)
	}
	return nil, repository.ErrNotFound
}

func (m *SpotAlertRepo) Delete(ctx context.Context, spotID, userUUID string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, spotID, userUUID)
	}
	return nil
}

func (m *SpotAlertRepo) GetByUser(ctx context.Context, userUUID string) ([]*model.SpotAlertSubscription, error) {
	if m.GetByUserFn != nil {
		return m.GetByUserFn(ctx, userUUID)
	}
	return nil, nil
}

func (m *SpotAlertRepo) GetBySpot(ctx context.Context, spotID string) ([]*model.SpotAlertSubscription, error) {
	if m.GetBySpotFn != nil {
		return m.GetBySpotFn(ctx, spotID)
	}
	return nil, nil
}

func (m *SpotAlertRepo) ListGoodSurfEnabled(ctx context.Context) ([]*model.SpotAlertSubscription, error) {
	if m.ListGoodSurfEnabledFn != nil {
		return m.ListGoodSurfEnabledFn(ctx)
	}
	return nil, nil
}

func (m *SpotAlertRepo) DeleteByUser(ctx context.Context, userUUID string) error {
	if m.DeleteByUserFn != nil {
		return m.DeleteByUserFn(ctx, userUUID)
	}
	return nil
}

func (m *SpotAlertRepo) UpdateLastNotifiedKey(ctx context.Context, spotID, userUUID, key string) error {
	if m.UpdateLastNotifiedKeyFn != nil {
		return m.UpdateLastNotifiedKeyFn(ctx, spotID, userUUID, key)
	}
	return nil
}
