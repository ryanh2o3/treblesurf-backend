package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This controller uses the shared service registry
// Note: Session handlers (TerminateSessionHandler, GetUserSessionsHandler, GetWebSocketTokenHandler)
// are implemented in internal/auth/service.go and routed from there

func SetUserTheme(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// First check if the user exists
	user, err := UserService.GetUserByEmail(email.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	if user == nil {
		log.Print("email address", email)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	theme := c.Query("theme")
	if theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme is required"})
		return
	}

	err = UserService.UpdateUserTheme(email.(string), theme)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme updated successfully"})
}

func DeleteMyAccount(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// First check if the user exists
	user, err := UserService.GetUserByEmail(email.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user information"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Delete the user account
	err = UserService.DeleteUser(email.(string))
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

func GetUserTheme(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// First check if the user exists
	user, err := UserService.GetUserByEmail(email.(string))
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