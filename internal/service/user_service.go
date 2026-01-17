package service

import (
	"context"
	"errors"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

// GetByEmail retrieves a user by email with context propagation.
func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByUUID retrieves a user by UUID with context propagation.
func (s *UserService) GetByUUID(ctx context.Context, uuid string) (*model.User, error) {
	user, err := s.users.GetByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateTheme updates a user's theme with context propagation.
func (s *UserService) UpdateTheme(ctx context.Context, email, theme string) error {
	return s.users.UpdateTheme(ctx, email, theme)
}

// Delete removes a user by email with context propagation.
func (s *UserService) Delete(ctx context.Context, email string) error {
	return s.users.Delete(ctx, email)
}

// Legacy helpers (no context) for existing callers.
func (s *UserService) GetUserByEmail(email string) (*model.User, error) {
	user, err := s.GetByEmail(context.Background(), email)
	if errors.Is(err, model.ErrUserNotFound) {
		return nil, nil
	}
	return user, err
}

func (s *UserService) GetUserByUUID(uuid string) (*model.User, error) {
	user, err := s.GetByUUID(context.Background(), uuid)
	if errors.Is(err, model.ErrUserNotFound) {
		return nil, nil
	}
	return user, err
}

func (s *UserService) UpdateUserTheme(email, theme string) error {
	return s.UpdateTheme(context.Background(), email, theme)
}

func (s *UserService) DeleteUser(email string) error {
	return s.Delete(context.Background(), email)
}
