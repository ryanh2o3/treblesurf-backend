package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.APIKeyRepository = (*APIKeyRepo)(nil)

type APIKeyRepo struct {
	CreateFn func(ctx context.Context, key *model.APIKey) error
	GetByKeyFn func(ctx context.Context, key string) (*model.APIKey, error)
	ListFn   func(ctx context.Context) ([]*model.APIKey, error)
	RevokeFn func(ctx context.Context, keyID string) error
}

func (m *APIKeyRepo) Create(ctx context.Context, key *model.APIKey) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, key)
	}
	return nil
}

func (m *APIKeyRepo) GetByKey(ctx context.Context, key string) (*model.APIKey, error) {
	if m.GetByKeyFn != nil {
		return m.GetByKeyFn(ctx, key)
	}
	return nil, repository.ErrNotFound
}

func (m *APIKeyRepo) List(ctx context.Context) ([]*model.APIKey, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return []*model.APIKey{}, nil
}

func (m *APIKeyRepo) Revoke(ctx context.Context, keyID string) error {
	if m.RevokeFn != nil {
		return m.RevokeFn(ctx, keyID)
	}
	return nil
}
