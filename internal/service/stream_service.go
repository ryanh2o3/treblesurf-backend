// Package service provides business logic services for the application.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

const streamRequestTTL = 5 * time.Minute

// StreamService provides business logic for stream request operations.
type StreamService struct {
	requests repository.StreamRequestRepository
}

// NewStreamService creates a new StreamService with the given repository.
// Returns an error if the repository is nil.
func NewStreamService(requests repository.StreamRequestRepository) (*StreamService, error) {
	if requests == nil {
		return nil, fmt.Errorf("stream request repository is required")
	}
	return &StreamService{requests: requests}, nil
}

func (s *StreamService) RequestStream(ctx context.Context, spotID, requestedBy string) (*model.StreamRequest, error) {
	if spotID == "" {
		return nil, fmt.Errorf("spot id is required")
	}
	if requestedBy == "" {
		return nil, fmt.Errorf("requested by is required")
	}

	now := time.Now()
	request := &model.StreamRequest{
		SpotID:      spotID,
		RequestedBy: requestedBy,
		RequestedAt: now,
		Expiration:  now.Add(streamRequestTTL).Unix(),
	}

	if err := s.requests.Save(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

func (s *StreamService) IsStreamRequested(ctx context.Context, spotID string) (bool, error) {
	if spotID == "" {
		return false, fmt.Errorf("spot id is required")
	}

	request, err := s.requests.GetBySpotID(ctx, spotID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if request == nil {
		return false, nil
	}

	if time.Now().Unix() > request.Expiration {
		return false, nil
	}

	return true, nil
}
