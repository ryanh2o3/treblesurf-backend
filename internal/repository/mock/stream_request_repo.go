package mock

import (
	"context"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

var _ repository.StreamRequestRepository = (*StreamRequestRepo)(nil)

type StreamRequestRepo struct {
	SaveFn        func(ctx context.Context, request *model.StreamRequest) error
	GetBySpotIDFn func(ctx context.Context, spotID string) (*model.StreamRequest, error)
}

func (m *StreamRequestRepo) Save(ctx context.Context, request *model.StreamRequest) error {
	if m.SaveFn != nil {
		return m.SaveFn(ctx, request)
	}
	return nil
}

func (m *StreamRequestRepo) GetBySpotID(ctx context.Context, spotID string) (*model.StreamRequest, error) {
	if m.GetBySpotIDFn != nil {
		return m.GetBySpotIDFn(ctx, spotID)
	}
	return nil, repository.ErrNotFound
}
