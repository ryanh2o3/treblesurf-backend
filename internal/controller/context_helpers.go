package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"treblesurf-backend/internal/logging"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	errEmailMissing = errors.New("email missing from context")
	errEmailInvalid = errors.New("email in context is invalid")
)

func requestLogger(c *gin.Context) *slog.Logger {
	logger := logging.FromContext(c.Request.Context())
	if email, ok := c.Get("email"); ok {
		if emailStr, ok := email.(string); ok && emailStr != "" {
			logger = logger.With(slog.String("user_email", emailStr))
		}
	}
	return logger
}

func getEmailFromContext(c *gin.Context) (string, error) {
	email, exists := c.Get("email")
	if !exists {
		return "", errEmailMissing
	}
	emailStr, ok := email.(string)
	if !ok || emailStr == "" {
		return "", errEmailInvalid
	}
	return emailStr, nil
}

func loadUserOr404(c *gin.Context, userSvc *service.UserService, email string) (*model.User, bool) {
	user, err := userSvc.GetByEmail(c.Request.Context(), email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return nil, false
		}
		requestLogger(c).Error("failed to fetch user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return nil, false
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return nil, false
	}
	return user, true
}
