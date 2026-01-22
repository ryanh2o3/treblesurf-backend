// Package service provides business logic services for the application.
package service

import (
	"context"
	"fmt"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// UserService provides business logic for user operations.
type UserService struct {
	users repository.UserRepository
}

// NewUserService creates a new UserService with the given repository.
// Returns an error if the repository is nil.
func NewUserService(users repository.UserRepository) (*UserService, error) {
	if users == nil {
		return nil, fmt.Errorf("user repository is required")
	}
	return &UserService{users: users}, nil
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

// Legacy helpers removed - use context-aware methods.
