package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	mockrepo "treblesurf-backend/internal/repository/mock"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	testUserEmail     = "test@example.com"
	testUserThemeDark = "dark"
)

func setupUserController(repo *mockrepo.UserRepo) *UserController {
	svc, _ := service.NewUserService(repo)
	return NewUserController(svc)
}

func TestUserController_SetUserTheme(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, Theme: "light"}, nil
		},
		UpdateThemeFn: func(_ context.Context, email, theme string) error {
			if email != testUserEmail {
				t.Errorf("unexpected email: %s", email)
			}
			if theme != testUserThemeDark {
				t.Errorf("unexpected theme: %s", theme)
			}
			return nil
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/setTheme?theme="+testUserThemeDark, http.NoBody)
	c.Set("email", testUserEmail)

	controller.SetUserTheme(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "Theme updated successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}
}

func TestUserController_SetUserTheme_Unauthorized(t *testing.T) {
	repo := &mockrepo.UserRepo{}
	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/setTheme?theme="+testUserThemeDark, http.NoBody)
	// No email in context

	controller.SetUserTheme(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestUserController_SetUserTheme_MissingTheme(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email}, nil
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/setTheme", http.NoBody)
	c.Set("email", testUserEmail)

	controller.SetUserTheme(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestUserController_SetUserTheme_UserNotFound(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, repository.ErrNotFound
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/setTheme?theme="+testUserThemeDark, http.NoBody)
	c.Set("email", testUserEmail)

	controller.SetUserTheme(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserController_GetUserTheme(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, Theme: testUserThemeDark}, nil
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getTheme", http.NoBody)
	c.Set("email", testUserEmail)

	controller.GetUserTheme(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["theme"] != testUserThemeDark {
		t.Errorf("expected theme %q, got %v", testUserThemeDark, response["theme"])
	}
}

func TestUserController_GetUserTheme_Unauthorized(t *testing.T) {
	repo := &mockrepo.UserRepo{}
	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/getTheme", http.NoBody)
	// No email in context

	controller.GetUserTheme(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestUserController_GetUserPreferences(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email, Theme: testUserThemeDark}, nil
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/preferences", http.NoBody)
	c.Set("email", testUserEmail)

	controller.GetUserPreferences(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["theme"] != testUserThemeDark {
		t.Errorf("expected theme %q, got %v", testUserThemeDark, response["theme"])
	}
}

func TestUserController_GetUserPreferences_Unauthorized(t *testing.T) {
	repo := &mockrepo.UserRepo{}
	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/user/preferences", http.NoBody)

	controller.GetUserPreferences(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestUserController_DeleteMyAccount(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email}, nil
		},
		DeleteFn: func(_ context.Context, email string) error {
			if email != testUserEmail {
				t.Errorf("unexpected email: %s", email)
			}
			return nil
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMyAccount", http.NoBody)
	c.Set("email", testUserEmail)

	controller.DeleteMyAccount(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "Account deleted successfully" {
		t.Errorf("expected success message, got %v", response["message"])
	}
}

func TestUserController_DeleteMyAccount_Unauthorized(t *testing.T) {
	repo := &mockrepo.UserRepo{}
	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMyAccount", http.NoBody)
	// No email in context

	controller.DeleteMyAccount(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestUserController_DeleteMyAccount_UserNotFound(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*model.User, error) {
			return nil, repository.ErrNotFound
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMyAccount", http.NoBody)
	c.Set("email", testUserEmail)

	controller.DeleteMyAccount(c)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUserController_DeleteMyAccount_DeleteError(t *testing.T) {
	repo := &mockrepo.UserRepo{
		GetByEmailFn: func(_ context.Context, email string) (*model.User, error) {
			return &model.User{Email: email}, nil
		},
		DeleteFn: func(_ context.Context, _ string) error {
			return errors.New("database error")
		},
	}

	controller := setupUserController(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/deleteMyAccount", http.NoBody)
	c.Set("email", testUserEmail)

	controller.DeleteMyAccount(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
