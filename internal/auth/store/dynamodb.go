package store

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/adam-hanna/sessions/user"
)

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
	return s.sessions.Save(context.Background(), sessionItem)
}

func (s *DynamoDBStore) DeleteUserSession(sessionID string) error {
	if s.sessions == nil {
		return fmt.Errorf("session repository not initialized")
	}
	return s.sessions.Delete(context.Background(), sessionID)
}

func (s *DynamoDBStore) FetchValidUserSession(sessionID string) (*user.Session, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session repository not initialized")
	}
	sessionItem, err := s.sessions.Get(context.Background(), sessionID)
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
	return nil
}

func (s *DynamoDBStore) GetSessionsByUserID(userID string) ([]*user.Session, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session repository not initialized")
	}
	sessionItems, err := s.sessions.GetByUserID(context.Background(), userID)
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
	return nil
}
