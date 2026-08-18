package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.SessionRepository = (*SessionRepo)(nil)

type SessionRepo struct {
	SaveFn        func(ctx context.Context, session *model.Session) error
	GetFn         func(ctx context.Context, sessionID string) (*model.Session, error)
	DeleteFn      func(ctx context.Context, sessionID string) error
	GetByUserIDFn func(ctx context.Context, userID string) ([]*model.Session, error)
}

func (m *SessionRepo) Save(ctx context.Context, session *model.Session) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, session)
	}
	return nil
}

func (m *SessionRepo) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, sessionID)
	}
	return nil, repository.ErrNotFound
}

func (m *SessionRepo) Delete(ctx context.Context, sessionID string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, sessionID)
	}
	return nil
}

func (m *SessionRepo) GetByUserID(ctx context.Context, userID string) ([]*model.Session, error) {
	if m.GetByUserIDFn != nil {
		return m.GetByUserIDFn(ctx, userID)
	}
	return []*model.Session{}, nil
}
