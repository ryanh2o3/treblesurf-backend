package service

import (
	"context"
	"fmt"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
)

const streamRequestTTL = 5 * time.Minute

type StreamService struct {
	requests repository.StreamRequestRepository
}

func NewStreamService(requests repository.StreamRequestRepository) *StreamService {
	return &StreamService{requests: requests}
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
