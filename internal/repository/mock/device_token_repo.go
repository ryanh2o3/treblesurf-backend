package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.DeviceTokenRepository = (*DeviceTokenRepo)(nil)

type DeviceTokenRepo struct {
	SaveFn         func(ctx context.Context, token *model.DeviceToken) error
	DeleteFn       func(ctx context.Context, userUUID, token string) error
	GetByUserFn    func(ctx context.Context, userUUID string) ([]*model.DeviceToken, error)
	DeleteByUserFn func(ctx context.Context, userUUID string) error
}

func (m *DeviceTokenRepo) Save(ctx context.Context, token *model.DeviceToken) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, token)
	}
	return nil
}

func (m *DeviceTokenRepo) Delete(ctx context.Context, userUUID, token string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, userUUID, token)
	}
	return nil
}

func (m *DeviceTokenRepo) GetByUser(ctx context.Context, userUUID string) ([]*model.DeviceToken, error) {
	if m.GetByUserFn != nil {
		return m.GetByUserFn(ctx, userUUID)
	}
	return nil, nil
}

func (m *DeviceTokenRepo) DeleteByUser(ctx context.Context, userUUID string) error {
	if m.DeleteByUserFn != nil {
		return m.DeleteByUserFn(ctx, userUUID)
	}
	return nil
}
