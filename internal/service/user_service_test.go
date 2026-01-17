package service

import (
	"context"
	"testing"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

func TestUserService_GetByEmail_ReturnsUser(t *testing.T) {
	ctx := context.WithValue(context.Background(), "ctx-key", "ctx-value")
	expected := &model.User{Email: "test@example.com"}
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(callCtx context.Context, email string) (*model.User, error) {
			if callCtx.Value("ctx-key") != "ctx-value" {
				t.Fatalf("expected context value to be propagated")
			}
			if email != expected.Email {
				t.Fatalf("unexpected email: %s", email)
			}
			return expected, nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetByEmail(ctx, expected.Email)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected user %+v, got %+v", expected, got)
	}
}

func TestUserService_UpdateTheme_UsesContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "ctx-key", "ctx-value")
	called := false
	repo := &mockrepo.UserRepo{
		UpdateThemeFn: func(callCtx context.Context, email, theme string) error {
			called = true
			if callCtx.Value("ctx-key") != "ctx-value" {
				t.Fatalf("expected context value to be propagated")
			}
			if email != "test@example.com" || theme != "dark" {
				t.Fatalf("unexpected args: %s %s", email, theme)
			}
			return nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if err := service.UpdateTheme(ctx, "test@example.com", "dark"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected UpdateTheme to be called")
	}
}

func TestUserService_GetByUUID_ReturnsUser(t *testing.T) {
	ctx := context.Background()
	expected := &model.User{UUID: "test-uuid-123", Email: "test@example.com"}
	repo := &mockrepo.UserRepo{
		GetByUUIDFn: func(callCtx context.Context, uuid string) (*model.User, error) {
			if uuid != expected.UUID {
				t.Fatalf("unexpected uuid: %s", uuid)
			}
			return expected, nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	got, err := service.GetByUUID(ctx, expected.UUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected user %+v, got %+v", expected, got)
	}
}

func TestUserService_Delete_UsesContext(t *testing.T) {
	ctx := context.Background()
	deleted := false
	repo := &mockrepo.UserRepo{
		DeleteFn: func(callCtx context.Context, email string) error {
			deleted = true
			if email != "test@example.com" {
				t.Fatalf("unexpected email: %s", email)
			}
			return nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if err := service.Delete(ctx, "test@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatalf("expected Delete to be called")
	}
}

func TestNewUserService_NilRepository_ReturnsError(t *testing.T) {
	_, err := NewUserService(nil)
	if err == nil {
		t.Fatalf("expected error for nil repository")
	}
}
