package controller

import (
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
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if _, ok := loadUserOr404(c, uc.users, emailStr); !ok {
		return
	}
	theme := c.Query("theme")
	if theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme is required"})
		return
	}

	err = uc.users.UpdateTheme(c.Request.Context(), emailStr, theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme updated successfully"})
}

func (uc *UserController) DeleteMyAccount(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if _, ok := loadUserOr404(c, uc.users, emailStr); !ok {
		return
	}

	// Delete the user account
	err = uc.users.Delete(c.Request.Context(), emailStr)
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
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := loadUserOr404(c, uc.users, emailStr)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"theme": user.Theme})
}

// GetUserPreferences returns basic user preference data for the iOS client.
func (uc *UserController) GetUserPreferences(c *gin.Context) {
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, ok := loadUserOr404(c, uc.users, emailStr)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"theme": user.Theme})
}