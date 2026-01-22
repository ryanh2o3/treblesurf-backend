package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
	GetByEmailFn      func(ctx context.Context, email string) (*model.User, error)
	GetByUUIDFn       func(ctx context.Context, uuid string) (*model.User, error)
	CreateFn          func(ctx context.Context, user *model.User) error
	UpdateFn          func(ctx context.Context, user *model.User) error
	DeleteFn          func(ctx context.Context, email string) error
	UpdateThemeFn     func(ctx context.Context, email, theme string) error
	UpdateLastLoginFn func(ctx context.Context, email string, at time.Time) error
}

func (m *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return nil, repository.ErrNotFound
}

func (m *UserRepo) GetByUUID(ctx context.Context, uuid string) (*model.User, error) {
	if m.GetByUUIDFn != nil {
		return m.GetByUUIDFn(ctx, uuid)
	}
	return nil, repository.ErrNotFound
}

func (m *UserRepo) Create(ctx context.Context, user *model.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *UserRepo) Update(ctx context.Context, user *model.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}

func (m *UserRepo) Delete(ctx context.Context, email string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, email)
	}
	return nil
}

func (m *UserRepo) UpdateTheme(ctx context.Context, email, theme string) error {
	if m.UpdateThemeFn != nil {
		return m.UpdateThemeFn(ctx, email, theme)
	}
	return nil
}

func (m *UserRepo) UpdateLastLogin(ctx context.Context, email string, at time.Time) error {
	if m.UpdateLastLoginFn != nil {
		return m.UpdateLastLoginFn(ctx, email, at)
	}
	return nil
}
