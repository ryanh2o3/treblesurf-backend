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

	service := NewUserService(repo)
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

	service := NewUserService(repo)
	if err := service.UpdateTheme(ctx, "test@example.com", "dark"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("expected UpdateTheme to be called")
	}
}
