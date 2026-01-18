package store

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/adam-hanna/sessions/user"
)

const sessionStoreTimeout = 10 * time.Second

type sessionTableManager interface {
	EnsureSessionsTable(ctx context.Context) error
	EnableTTL(ctx context.Context) error
}

// DynamoDBStore implements the session store interface using a session repository.
type DynamoDBStore struct {
	sessions repository.SessionRepository
}

func NewDynamoDBStore(sessionsRepo repository.SessionRepository) *DynamoDBStore {
	return &DynamoDBStore{
		sessions: sessionsRepo,
	}
}

func (s *DynamoDBStore) SaveUserSession(userSession *user.Session) error {
	if s.sessions == nil {
		return fmt.Errorf("session repository not initialized")
	}
	sessionItem := &model.Session{
		SessionID: userSession.ID,
		UserID:    userSession.UserID,
		ExpiresAt: userSession.ExpiresAt,
		JSON:      userSession.JSON,
		TTL:       userSession.ExpiresAt.Unix(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	return s.sessions.Save(ctx, sessionItem)
}

func (s *DynamoDBStore) DeleteUserSession(sessionID string) error {
	if s.sessions == nil {
		return fmt.Errorf("session repository not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	return s.sessions.Delete(ctx, sessionID)
}

func (s *DynamoDBStore) FetchValidUserSession(sessionID string) (*user.Session, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session repository not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	sessionItem, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sessionItem == nil || time.Now().After(sessionItem.ExpiresAt) {
		return nil, nil
	}

	return &user.Session{
		ID:        sessionItem.SessionID,
		UserID:    sessionItem.UserID,
		ExpiresAt: sessionItem.ExpiresAt,
		JSON:      sessionItem.JSON,
	}, nil
}

func (s *DynamoDBStore) EnableTTL() error {
	manager, ok := s.sessions.(sessionTableManager)
	if !ok {
		return fmt.Errorf("session repository does not support TTL management")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	return manager.EnableTTL(ctx)
}

func (s *DynamoDBStore) GetSessionsByUserID(userID string) ([]*user.Session, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session repository not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	sessionItems, err := s.sessions.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	userSessions := make([]*user.Session, 0, len(sessionItems))
	now := time.Now()
	for _, sessionItem := range sessionItems {
		if sessionItem == nil || now.After(sessionItem.ExpiresAt) {
			continue
		}
		userSessions = append(userSessions, &user.Session{
			ID:        sessionItem.SessionID,
			UserID:    sessionItem.UserID,
			ExpiresAt: sessionItem.ExpiresAt,
			JSON:      sessionItem.JSON,
		})
	}

	return userSessions, nil
}

func (s *DynamoDBStore) EnsureSessionsTable() error {
	manager, ok := s.sessions.(sessionTableManager)
	if !ok {
		return fmt.Errorf("session repository does not support table management")
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionStoreTimeout)
	defer cancel()
	return manager.EnsureSessionsTable(ctx)
}
