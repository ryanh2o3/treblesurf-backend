package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

// validateGoogleIDToken validates a Google ID token against client IDs.
func validateGoogleIDToken(ctx context.Context, idToken string, clientIDs map[string]bool) (*idtoken.Payload, error) {
	var payload *idtoken.Payload
	var err error

	for id := range clientIDs {
		payload, err = idtoken.Validate(ctx, idToken, id)
		if err == nil {
			return payload, nil
		}
	}

	return nil, err
}

// extractUserClaims extracts user claims from a Google ID token payload.
//
//nolint:unparam,gocritic // Error return maintained for API consistency; multiple return values needed for all claims
func extractUserClaims(payload *idtoken.Payload) (email, name, picture, familyName, givenName string, err error) {
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
func (s *Service) handleNewUser(ctx context.Context, email, name, picture, familyName, givenName string) (*User, error) {
	newUser := User{
		Email:      email,
		Name:       name,
		Picture:    picture,
		FamilyName: familyName,
		GivenName:  givenName,
	}

	if err := s.createUser(ctx, newUser); err != nil {
		return nil, err
	}

	slog.Info("created new user", slog.String("email", email))
	return s.getUserByEmail(ctx, email)
}

// handleNewAppleUser creates a user keyed by Apple sub (and email when present).
func (s *Service) handleNewAppleUser(ctx context.Context, email, appleID string) (*User, error) {
	newUser := User{
		Email:   email,
		AppleID: appleID,
		// Apple does not provide avatar or name in the identity token for native iOS
		// when the client only posts identity_token (name is only available on first
		// ASAuthorizationAppleIDCredential and is not sent by the current iOS client).
		Picture: "",
	}

	if err := s.createUser(ctx, newUser); err != nil {
		return nil, err
	}

	slog.Info("created new apple user", slog.String("email", email), slog.String("apple_id", appleID))
	return s.getUserByEmail(ctx, email)
}

// handleExistingAppleUser updates last login for a known Apple user.
func (s *Service) handleExistingAppleUser(ctx context.Context, existing *User) (*User, error) {
	if err := s.updateUserLastLogin(ctx, existing.Email); err != nil {
		slog.Warn("error updating last login", slog.Any("error", err))
	}

	if err := s.ensureUserHasUUID(ctx, existing.Email); err != nil {
		slog.Warn("error ensuring user has UUID", slog.Any("error", err))
	}

	userData, err := s.getUserByEmail(ctx, existing.Email)
	if err != nil {
		return nil, err
	}

	slog.Info("apple user logged in", slog.String("email", existing.Email), slog.String("apple_id", existing.AppleID))
	return userData, nil
}

// linkAppleIDToUser attaches an Apple sub to an existing email-keyed account.
func (s *Service) linkAppleIDToUser(ctx context.Context, existing *User, appleID string) (*User, error) {
	if existing.AppleID != "" && existing.AppleID != appleID {
		return nil, fmt.Errorf("user already linked to a different apple_id")
	}

	if existing.AppleID == "" {
		modelUser := toModelUser(existing)
		modelUser.AppleID = appleID
		if err := s.userRepo.Update(ctx, modelUser); err != nil {
			return nil, err
		}
	}

	return s.handleExistingAppleUser(ctx, &User{
		Email:   existing.Email,
		AppleID: appleID,
	})
}

// handleExistingUser updates last login and ensures UUID, returns updated user.
func (s *Service) handleExistingUser(ctx context.Context, email string) (*User, error) {
	if err := s.updateUserLastLogin(ctx, email); err != nil {
		slog.Warn("error updating last login", slog.Any("error", err))
	}

	if err := s.ensureUserHasUUID(ctx, email); err != nil {
		slog.Warn("error ensuring user has UUID", slog.Any("error", err))
	}

	userData, err := s.getUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	slog.Info("user logged in", slog.String("email", email))
	return userData, nil
}

// setupCSRFToken sets CSRF token cookie and header.
func setupCSRFToken(c *gin.Context, secure bool) string {
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return ""
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"csrf_token",
		csrfToken,
		int(24*time.Hour.Seconds()),
		"/",
		"",
		secure, // Secure
		false,  // Not HTTP-only (JS needs to access it)
	)
	c.Header("X-CSRF-Token", csrfToken)

	return csrfToken
}

// createSession creates a session for the user if session service is available.
func (s *Service) createSession(email, csrfToken string, c *gin.Context) {
	if s.sessionService == nil {
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

	_, err = s.sessionService.IssueUserSession(email, string(jsonBytes), c.Writer)
	if err != nil {
		slog.Warn("error creating session", slog.Any("error", err))
	}
}

// setAuthCookie sets the authentication cookie.
func setAuthCookie(c *gin.Context, secure bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"auth_token",
		"authenticated",
		int(24*time.Hour.Seconds()),
		"/",
		"",
		secure, // Secure (HTTPS only)
		true,   // HTTP-only
	)
}

func buildClientIDMap(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		out[id] = true
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
func (s *Service) ensureUserUUID(ctx context.Context, userData *User, email string) *User {
	if userData.UUID != "" {
		return userData
	}

	if err := s.ensureUserHasUUID(ctx, email); err != nil {
		slog.Warn("error ensuring user has UUID", slog.Any("error", err))
	}

	updatedUser, err := s.getUserByEmail(ctx, email)
	if err != nil {
		slog.Warn("error getting updated user data", slog.Any("error", err))
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
		slog.Warn("failed to marshal session data", slog.Any("error", err))
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
			"email":       userData.Email,
			"name":        userData.Name,
			"picture":     userData.Picture,
			"family_name": userData.FamilyName,
			"given_name":  userData.GivenName,
			"theme":       userData.Theme,
			"uuid":        userData.UUID,
		},
	}
}
