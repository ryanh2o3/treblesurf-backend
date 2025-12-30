package auth

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

// getGoogleClientIDs retrieves Google OAuth client IDs from environment variables.
//nolint:unparam // Error return maintained for API consistency
func getGoogleClientIDs() (map[string]bool, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		return nil, nil
	}

	clientIDs := make(map[string]bool)
	clientIDs[clientID] = true

	iosClientID := os.Getenv("GOOGLE_IOS_CLIENT_ID")
	if iosClientID != "" {
		clientIDs[iosClientID] = true
	}

	return clientIDs, nil
}

// validateGoogleIDToken validates a Google ID token against client IDs.
func validateGoogleIDToken(idToken string, clientIDs map[string]bool) (*idtoken.Payload, error) {
	var payload *idtoken.Payload
	var err error

	for id := range clientIDs {
		payload, err = idtoken.Validate(context.Background(), idToken, id)
		if err == nil {
			return payload, nil
		}
	}

	return nil, err
}

// extractUserClaims extracts user claims from a Google ID token payload.
//nolint:unparam // Error return maintained for API consistency; multiple return values needed for all claims
func extractUserClaims(payload *idtoken.Payload) (email, name, picture, familyName, givenName string, err error) {
	log.Printf("JWT claims available: %v", payload.Claims)

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return "", "", "", "", "", nil // Will be handled by caller
	}

	if nameVal, ok := payload.Claims["name"].(string); ok {
		name = nameVal
	}
	if pictureVal, ok := payload.Claims["picture"].(string); ok {
		picture = pictureVal
	}
	if familyNameVal, ok := payload.Claims["family_name"].(string); ok {
		familyName = familyNameVal
	}
	if givenNameVal, ok := payload.Claims["given_name"].(string); ok {
		givenName = givenNameVal
	}
	// Note: Type assertions above are safe - these are optional fields

	return email, name, picture, familyName, givenName, nil
}

// handleNewUser creates a new user and returns the user data.
func handleNewUser(email, name, picture, familyName, givenName string) (*User, error) {
	newUser := User{
		Email:      email,
		Name:       name,
		Picture:    picture,
		FamilyName: familyName,
		GivenName:  givenName,
	}

	if err := createUser(newUser); err != nil {
		return nil, err
	}

	log.Printf("Created new user: %s", email)
	return getUserByEmail(email)
}

// handleExistingUser updates last login and ensures UUID, returns updated user.
func handleExistingUser(email string) (*User, error) {
	if err := updateUserLastLogin(email); err != nil {
		log.Printf("Error updating last login: %v", err)
	}

	if err := ensureUserHasUUID(email); err != nil {
		log.Printf("Error ensuring user has UUID: %v", err)
	}

	userData, err := getUserByEmail(email)
	if err != nil {
		return nil, err
	}

	log.Printf("User logged in: %s", email)
	return userData, nil
}

// setupCSRFToken sets CSRF token cookie and header.
func setupCSRFToken(c *gin.Context) string {
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return ""
	}

	c.SetCookie(
		"csrf_token",
		csrfToken,
		int(24*time.Hour.Seconds()),
		"/",
		"",
		true,  // Secure
		false, // Not HTTP-only (JS needs to access it)
	)
	c.Header("X-CSRF-Token", csrfToken)

	return csrfToken
}

// createSession creates a session for the user if session service is available.
func createSession(email, csrfToken string, c *gin.Context) {
	if sessionService == nil {
		return
	}

	sessionData := SessionJSON{
		CSRF:       csrfToken,
		UserAgent:  c.Request.UserAgent(),
		IPAddress:  getClientIP(c),
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	jsonBytes, err := json.Marshal(sessionData)
	if err != nil {
		return
	}

	_, err = sessionService.IssueUserSession(email, string(jsonBytes), c.Writer)
	if err != nil {
		log.Printf("Error creating session: %v", err)
	}
}

// setAuthCookie sets the authentication cookie.
func setAuthCookie(c *gin.Context) {
	c.SetCookie(
		"auth_token",
		"authenticated",
		int(24*time.Hour.Seconds()),
		"/",
		"",
		true, // Secure (HTTPS only)
		true, // HTTP-only
	)
}

// buildUserResponse builds the user response object.
func buildUserResponse(userData *User, email, name, picture, familyName, givenName, theme string) gin.H {
	var userUUID string
	if userData != nil && userData.UUID != "" {
		userUUID = userData.UUID
	}

	return gin.H{
		"uuid":        userUUID,
		"email":       email,
		"name":        name,
		"picture":     picture,
		"family_name": familyName,
		"given_name":  givenName,
		"theme":       theme,
	}
}

// setCacheControlHeaders sets cache control headers to prevent caching.
func setCacheControlHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// ensureUserUUID ensures a user has a UUID and refreshes user data if needed.
func ensureUserUUID(userData *User, email string) *User {
	if userData.UUID != "" {
		return userData
	}

	if err := ensureUserHasUUID(email); err != nil {
		log.Printf("Error ensuring user has UUID: %v", err)
	}

	updatedUser, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Error getting updated user data: %v", err)
		return userData
	}

	return updatedUser
}

// updateSessionLastActive updates the session's last active time and CSRF token.
func updateSessionLastActive(userSession *user.Session, c *gin.Context) {
	var sessionData SessionJSON
	if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err != nil {
		return
	}

	sessionData.LastActive = time.Now()
	updatedJSON, err := json.Marshal(sessionData)
	if err != nil {
		log.Printf("Failed to marshal session data: %v", err)
		return
	}
	userSession.JSON = string(updatedJSON)

	if sessionData.CSRF != "" {
		c.Header("X-CSRF-Token", sessionData.CSRF)
	}
}

// buildValidateTokenResponse builds the response for token validation.
func buildValidateTokenResponse(userData *User, authType string) gin.H {
	return gin.H{
		"valid":     true,
		"auth_type": authType,
		"user": gin.H{
			"email":      userData.Email,
			"name":       userData.Name,
			"picture":    userData.Picture,
			"family_name": userData.FamilyName,
			"given_name":  userData.GivenName,
			"theme":       userData.Theme,
			"uuid":        userData.UUID,
		},
	}
}

