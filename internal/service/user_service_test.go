package service

import (
	"context"
	"testing"

	"treblesurf-backend/internal/model"
	mockrepo "treblesurf-backend/internal/repository/mock"
)

type userCtxKey string

const (
	userCtxKeyValue userCtxKey = "ctx-key"
	userCtxValue    string     = "ctx-value"
	testUserEmail              = "test@example.com"
)

func TestUserService_GetByEmail_ReturnsUser(t *testing.T) {
	ctx := context.WithValue(context.Background(), userCtxKeyValue, userCtxValue)
	expected := &model.User{Email: testUserEmail}
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(callCtx context.Context, email string) (*model.User, error) {
			if callCtx.Value(userCtxKeyValue) != userCtxValue {
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
	ctx := context.WithValue(context.Background(), userCtxKeyValue, userCtxValue)
	called := false
	repo := &mockrepo.UserRepo{
		UpdateThemeFn: func(callCtx context.Context, email, theme string) error {
			called = true
			if callCtx.Value(userCtxKeyValue) != userCtxValue {
				t.Fatalf("expected context value to be propagated")
			}
			if email != testUserEmail || theme != "dark" {
				t.Fatalf("unexpected args: %s %s", email, theme)
			}
			return nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if err := service.UpdateTheme(ctx, testUserEmail, "dark"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected UpdateTheme to be called")
	}
}

func TestUserService_GetByUUID_ReturnsUser(t *testing.T) {
	ctx := context.Background()
	expected := &model.User{UUID: "test-uuid-123", Email: testUserEmail}
	repo := &mockrepo.UserRepo{
		GetByUUIDFn: func(_ context.Context, uuid string) (*model.User, error) {
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
		DeleteFn: func(_ context.Context, email string) error {
			deleted = true
			if email != testUserEmail {
				t.Fatalf("unexpected email: %s", email)
			}
			return nil
		},
	}

	service, err := NewUserService(repo)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	if err := service.Delete(ctx, testUserEmail); err != nil {
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
