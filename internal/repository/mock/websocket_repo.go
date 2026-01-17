package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.WebSocketRepository = (*WebSocketRepo)(nil)

type WebSocketRepo struct {
	SaveConnectionFn        func(ctx context.Context, conn *model.ConnectionInfo) error
	GetConnectionFn         func(ctx context.Context, connectionID string) (*model.ConnectionInfo, error)
	DeleteConnectionFn      func(ctx context.Context, connectionID string) error
	UpdateSpotFn            func(ctx context.Context, connectionID, spot string) error
	GetConnectionsByUserIDsFn func(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error)
}

func (m *WebSocketRepo) SaveConnection(ctx context.Context, conn *model.ConnectionInfo) error {
	if m.SaveConnectionFn != nil {
		return m.SaveConnectionFn(ctx, conn)
	}
	return nil
}

func (m *WebSocketRepo) GetConnection(ctx context.Context, connectionID string) (*model.ConnectionInfo, error) {
	if m.GetConnectionFn != nil {
		return m.GetConnectionFn(ctx, connectionID)
	}
	return nil, model.ErrWebSocketConnectionNotFound
}

func (m *WebSocketRepo) DeleteConnection(ctx context.Context, connectionID string) error {
	if m.DeleteConnectionFn != nil {
		return m.DeleteConnectionFn(ctx, connectionID)
	}
	return nil
}

func (m *WebSocketRepo) UpdateSpot(ctx context.Context, connectionID, spot string) error {
	if m.UpdateSpotFn != nil {
		return m.UpdateSpotFn(ctx, connectionID, spot)
	}
	return nil
}

func (m *WebSocketRepo) GetConnectionsByUserIDs(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error) {
	if m.GetConnectionsByUserIDsFn != nil {
		return m.GetConnectionsByUserIDsFn(ctx, userIDs)
	}
	return nil, nil
}
