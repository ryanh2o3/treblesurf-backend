package mock

import (
	"context"
	"time"

	"treblesurf-backend/internal/repository"
)

var _ repository.MediaRepository = (*MediaRepo)(nil)

type MediaRepo struct {
	UploadFn            func(ctx context.Context, key string, data []byte, contentType string) error
	DownloadFn          func(ctx context.Context, key string) ([]byte, error)
	ExistsFn            func(ctx context.Context, key string) (bool, error)
	DeleteFn            func(ctx context.Context, key string) error
	GenerateUploadURLFn func(ctx context.Context, key string, expires time.Duration) (string, error)
	GenerateViewURLFn   func(ctx context.Context, key string, expires time.Duration) (string, error)
}

func (m *MediaRepo) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, key, data, contentType)
	}
	return nil
}

func (m *MediaRepo) Download(ctx context.Context, key string) ([]byte, error) {
	if m.DownloadFn != nil {
		return m.DownloadFn(ctx, key)
	}
	return nil, repository.ErrNotFound
}

func (m *MediaRepo) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFn != nil {
		return m.ExistsFn(ctx, key)
	}
	return false, repository.ErrNotFound
}

func (m *MediaRepo) Delete(ctx context.Context, key string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}

func (m *MediaRepo) GenerateUploadURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if m.GenerateUploadURLFn != nil {
		return m.GenerateUploadURLFn(ctx, key, expires)
	}
	return "", nil
}

func (m *MediaRepo) GenerateViewURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if m.GenerateViewURLFn != nil {
		return m.GenerateViewURLFn(ctx, key, expires)
	}
	return "", nil
}
