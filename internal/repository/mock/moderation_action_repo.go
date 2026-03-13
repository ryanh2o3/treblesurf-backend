package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.ModerationActionRepository = (*ModerationActionRepo)(nil)

// ModerationActionRepo is a mock implementation of ModerationActionRepository.
type ModerationActionRepo struct {
	CreateFn        func(ctx context.Context, action *model.ModerationAction) error
	GetByReportIDFn func(ctx context.Context, reportID string) ([]*model.ModerationAction, error)
	ListFn          func(ctx context.Context, limit, offset int) ([]*model.ModerationAction, error)
}

func (m *ModerationActionRepo) Create(ctx context.Context, action *model.ModerationAction) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, action)
	}
	return nil
}

func (m *ModerationActionRepo) GetByReportID(
	ctx context.Context,
	reportID string,
) ([]*model.ModerationAction, error) {
	if m.GetByReportIDFn != nil {
		return m.GetByReportIDFn(ctx, reportID)
	}
	return []*model.ModerationAction{}, nil
}

func (m *ModerationActionRepo) List(
	ctx context.Context,
	limit, offset int,
) ([]*model.ModerationAction, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return []*model.ModerationAction{}, nil
}
