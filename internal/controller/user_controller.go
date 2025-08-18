package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// This controller uses the shared service registry

// TerminateSessionHandler terminates a user session
func TerminateSessionHandler(c *gin.Context) {
	// TODO: Implement session termination
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// GetUserSessionsHandler retrieves user sessions
func GetUserSessionsHandler(c *gin.Context) {
	// TODO: Implement session retrieval
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// GetWebSocketTokenHandler generates a WebSocket token
func GetWebSocketTokenHandler(c *gin.Context) {
	// TODO: Implement WebSocket token generation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// SetUserTheme updates user theme preference
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

// DeleteMyAccount deletes user account
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

// GetUserTheme retrieves user theme preference
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