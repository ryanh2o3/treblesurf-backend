package service

import (
	"context"
	"fmt"
	"log/slog"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

// UserService provides business logic for user operations.
type UserService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	reports  repository.ReportRepository
}

// NewUserService creates a new UserService with the given repository.
// Returns an error if the repository is nil.
func NewUserService(users repository.UserRepository) (*UserService, error) {
	if users == nil {
		return nil, fmt.Errorf("user repository is required")
	}
	return &UserService{users: users}, nil
}

// WithAccountCleanup attaches session and report repositories used when deleting an account.
func (s *UserService) WithAccountCleanup(
	sessions repository.SessionRepository,
	reports repository.ReportRepository,
) *UserService {
	if s == nil {
		return nil
	}
	s.sessions = sessions
	s.reports = reports
	return s
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

// Delete removes a user by email, invalidates sessions, and anonymizes surf reports.
func (s *UserService) Delete(ctx context.Context, email string) error {
	if s.reports != nil {
		if err := s.reports.AnonymizeByUserEmail(ctx, email); err != nil {
			slog.Warn("failed to anonymize surf reports during account deletion",
				slog.String("email", email),
				slog.Any("error", err),
			)
			return fmt.Errorf("anonymizing surf reports: %w", err)
		}
	}

	if err := s.deleteUserSessions(ctx, email); err != nil {
		return err
	}

	return s.users.Delete(ctx, email)
}

func (s *UserService) deleteUserSessions(ctx context.Context, email string) error {
	if s.sessions == nil {
		return nil
	}

	sessions, err := s.sessions.GetByUserID(ctx, email)
	if err != nil {
		slog.Warn("failed to list sessions during account deletion",
			slog.String("email", email),
			slog.Any("error", err),
		)
		return fmt.Errorf("listing sessions: %w", err)
	}

	for _, session := range sessions {
		if session == nil || session.SessionID == "" {
			continue
		}
		if err := s.sessions.Delete(ctx, session.SessionID); err != nil {
			slog.Warn("failed to delete session during account deletion",
				slog.String("session_id", session.SessionID),
				slog.Any("error", err),
			)
			return fmt.Errorf("deleting session: %w", err)
		}
	}

	return nil
}

// Legacy helpers removed - use context-aware methods.
