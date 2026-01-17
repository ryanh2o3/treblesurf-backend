package controller

import (
	"log"
	"net/http"

	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// UserController handles user-related routes.
type UserController struct {
	users *service.UserService
}

func NewUserController(users *service.UserService) *UserController {
	return &UserController{users: users}
}

// Note: Session handlers (TerminateSessionHandler, GetUserSessionsHandler, GetWebSocketTokenHandler)
// are implemented in internal/auth/service.go and routed from there.

func (uc *UserController) SetUserTheme(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}

	// First check if the user exists
	user, err := uc.users.GetUserByEmail(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	if user == nil {
		log.Print("email address", emailStr)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	theme := c.Query("theme")
	if theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme is required"})
		return
	}

	err = uc.users.UpdateUserTheme(emailStr, theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme updated successfully"})
}

func (uc *UserController) DeleteMyAccount(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}

	// First check if the user exists
	user, err := uc.users.GetUserByEmail(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete the user account
	err = uc.users.DeleteUser(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	// Here you could add additional cleanup for user data if needed
	// For example:
	// - Delete user preferences
	// - Delete saved spots
	// - Remove user reports/contributions

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}

func (uc *UserController) GetUserTheme(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email in context"})
		return
	}

	// First check if the user exists
	user, err := uc.users.GetUserByEmail(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	theme := user.Theme
	c.JSON(http.StatusOK, gin.H{"theme": theme})
}