package service

import (
	"context"
	"testing"
	"time"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestNewStreamService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewStreamService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}

func TestStreamService_RequestStream(t *testing.T) {
	t.Run("creates valid request", func(t *testing.T) {
		ctx := context.Background()
		var saved *model.StreamRequest
		repo := &mockrepo.StreamRequestRepo{
			SaveFn: func(_ context.Context, request *model.StreamRequest) error {
				saved = request
				return nil
			},
		}

		service, err := NewStreamService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		request, err := service.RequestStream(ctx, "Ireland_Donegal_Bundoran", "user@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if request == nil {
			t.Fatalf("expected request to be returned")
		}
		if saved == nil {
			t.Fatalf("expected request to be saved")
		}
		if request.SpotID != "Ireland_Donegal_Bundoran" {
			t.Fatalf("unexpected spot ID: %s", request.SpotID)
		}
		if request.RequestedBy != "user@example.com" {
			t.Fatalf("unexpected requested by: %s", request.RequestedBy)
		}
		if request.Expiration == 0 {
			t.Fatalf("expected expiration to be set")
		}
	})

	t.Run("returns error for empty spot ID", func(t *testing.T) {
		repo := &mockrepo.StreamRequestRepo{}
		service, _ := NewStreamService(repo)

		_, err := service.RequestStream(context.Background(), "", "user@example.com")
		if err == nil {
			t.Fatalf("expected error for empty spot ID")
		}
	})

	t.Run("returns error for empty requested by", func(t *testing.T) {
		repo := &mockrepo.StreamRequestRepo{}
		service, _ := NewStreamService(repo)

		_, err := service.RequestStream(context.Background(), "spot-id", "")
		if err == nil {
			t.Fatalf("expected error for empty requested by")
		}
	})
}

func TestStreamService_IsStreamRequested(t *testing.T) {
	t.Run("returns true for active request", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.StreamRequestRepo{
			GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
				return &model.StreamRequest{
					SpotID:     spotID,
					Expiration: time.Now().Add(5 * time.Minute).Unix(),
				}, nil
			},
		}

		service, err := NewStreamService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		requested, err := service.IsStreamRequested(ctx, "Ireland_Donegal_Bundoran")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !requested {
			t.Fatalf("expected stream to be requested")
		}
	})

	t.Run("returns false for expired request", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.StreamRequestRepo{
			GetBySpotIDFn: func(_ context.Context, spotID string) (*model.StreamRequest, error) {
				return &model.StreamRequest{
					SpotID:     spotID,
					Expiration: time.Now().Add(-5 * time.Minute).Unix(), // Expired
				}, nil
			},
		}

		service, err := NewStreamService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		requested, err := service.IsStreamRequested(ctx, "Ireland_Donegal_Bundoran")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requested {
			t.Fatalf("expected stream to not be requested (expired)")
		}
	})

	t.Run("returns false for no request", func(t *testing.T) {
		ctx := context.Background()
		repo := &mockrepo.StreamRequestRepo{
			GetBySpotIDFn: func(_ context.Context, _ string) (*model.StreamRequest, error) {
				return nil, nil
			},
		}

		service, err := NewStreamService(repo)
		if err != nil {
			t.Fatalf("unexpected error creating service: %v", err)
		}

		requested, err := service.IsStreamRequested(ctx, "Ireland_Donegal_Bundoran")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requested {
			t.Fatalf("expected stream to not be requested")
		}
	})

	t.Run("returns error for empty spot ID", func(t *testing.T) {
		repo := &mockrepo.StreamRequestRepo{}
		service, _ := NewStreamService(repo)

		_, err := service.IsStreamRequested(context.Background(), "")
		if err == nil {
			t.Fatalf("expected error for empty spot ID")
		}
	})
}
