package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"treblesurf-backend/internal/auth/store"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/constants"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/adam-hanna/sessions"
	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/transport"
	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

type TokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type TokenRequest struct {
	IDToken string `json:"id_token"`
}

type User struct {
	UUID       string `json:"uuid"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
	FamilyName string `json:"family_name"`
	GivenName  string `json:"given_name"`
	CreatedAt  string `json:"created_at"`
	LastLogin  string `json:"last_login"`
	Theme      string `json:"theme"`
}

type SessionJSON struct {
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
	CSRF       string    `json:"csrf"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
}

type SessionInfo struct {
	ExpiresAt  time.Time `json:"expires_at"`
	LastActive time.Time `json:"last_active,omitempty"`
	SessionID  string    `json:"session_id"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	Current    bool      `json:"current"`
}

// Service provides authentication handlers and middleware with injected dependencies.
type Service struct {
	userRepo       repository.UserRepository
	sessionRepo    repository.SessionRepository
	sessionService *sessions.Service
	sessionStore   *store.DynamoDBStore
	logger         *slog.Logger
	jwtSecret      []byte
	googleClientIDs map[string]bool
	cookieSecure    bool
	isDevelopment   bool
}

// NewService constructs an auth service with its required dependencies.
func NewService(
	cfg *config.Config,
	users repository.UserRepository,
	sessionsRepo repository.SessionRepository,
	logger *slog.Logger,
) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must be set")
	}
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret must be set")
	}
	if users == nil {
		return nil, fmt.Errorf("user repository must be set")
	}
	if sessionsRepo == nil {
		return nil, fmt.Errorf("session repository must be set")
	}
	if logger == nil {
		logger = slog.Default()
	}

	service := &Service{
		jwtSecret:       []byte(cfg.Auth.JWTSecret),
		userRepo:        users,
		sessionRepo:     sessionsRepo,
		logger:          logger,
		googleClientIDs: buildClientIDMap(cfg.Auth.GoogleClientIDs),
		cookieSecure:    cfg.Auth.CookieSecure,
		isDevelopment:   cfg.IsDevelopment(),
	}

	if err := service.initSessionService(); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *Service) initSessionService() error {
	sessionStore := store.NewDynamoDBStore(s.sessionRepo)
	s.sessionStore = sessionStore

	if err := sessionStore.EnsureSessionsTable(); err != nil {
		return fmt.Errorf("failed to ensure Sessions table: %w", err)
	}

	if err := sessionStore.EnableTTL(); err != nil {
		s.logger.Warn("failed to enable TTL on sessions table", slog.Any("error", err))
	}

	authService, err := auth.New(auth.Options{
		Key: s.jwtSecret,
	})
	if err != nil {
		return err
	}

	transportService := transport.New(transport.Options{
		HTTPOnly:   true,
		Secure:     s.cookieSecure,
		CookiePath: "/",
		CookieName: "session_id",
	})

	s.sessionService = sessions.New(sessionStore, authService, transportService, sessions.Options{
		ExpirationDuration: 30 * 24 * time.Hour,
	})

	return nil
}

func (s *Service) getUserByEmail(ctx context.Context, email string) (*User, error) {
	userData, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return toAuthUser(userData), nil
}

//nolint:gocritic // User struct size is acceptable for this use case
func (s *Service) createUser(ctx context.Context, user User) error {
	userUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}
	user.UUID = userUUID.String()

	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = now
	user.LastLogin = now
	user.Theme = constants.DefaultUserTheme
	return s.userRepo.Create(ctx, toModelUser(&user))
}

func (s *Service) updateUserLastLogin(ctx context.Context, email string) error {
	return s.userRepo.UpdateLastLogin(ctx, email, time.Now())
}

func (s *Service) ensureUserHasUUID(ctx context.Context, email string) error {
	// Get the current user to check if they have a UUID
	userData, err := s.getUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if userData == nil {
		return fmt.Errorf("user not found")
	}

	if userData.UUID != "" {
		return nil
	}

	newUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	modelUser := toModelUser(userData)
	modelUser.UUID = newUUID.String()
	if err := s.userRepo.Update(ctx, modelUser); err != nil {
		return fmt.Errorf("failed to update user with UUID: %w", err)
	}

	s.logger.Info("assigned UUID to user", slog.String("uuid", newUUID.String()), slog.String("email", email))
	return nil
}

func toAuthUser(userData *model.User) *User {
	if userData == nil {
		return nil
	}
	return &User{
		UUID:       userData.UUID,
		Email:      userData.Email,
		Name:       userData.Name,
		Picture:    userData.Picture,
		FamilyName: userData.FamilyName,
		GivenName:  userData.GivenName,
		CreatedAt:  userData.CreatedAt,
		LastLogin:  userData.LastLogin,
		Theme:      userData.Theme,
	}
}

func toModelUser(userData *User) *model.User {
	if userData == nil {
		return nil
	}
	return &model.User{
		UUID:       userData.UUID,
		Email:      userData.Email,
		Name:       userData.Name,
		Picture:    userData.Picture,
		FamilyName: userData.FamilyName,
		GivenName:  userData.GivenName,
		CreatedAt:  userData.CreatedAt,
		LastLogin:  userData.LastLogin,
		Theme:      userData.Theme,
	}
}

func GenerateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func getClientIP(c *gin.Context) string {
	// Try to get real IP from various headers
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}

func (s *Service) GoogleAuthHandler(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	payload, email, name, picture, familyName, givenName := s.validateAndExtractGoogleClaims(c, req.IDToken)
	if payload == nil {
		return
	}

	finalUser, theme := s.processGoogleAuthUser(email, name, picture, familyName, givenName, c)
	if finalUser == nil {
		return
	}

	s.setupAuthSession(email, c)
	c.JSON(http.StatusOK, gin.H{
		"user": buildUserResponse(finalUser, email, name, picture, familyName, givenName, theme),
	})
}

//nolint:gocritic // Multiple return values needed for all claims
func (s *Service) validateAndExtractGoogleClaims(
	c *gin.Context,
	idToken string,
) (*idtoken.Payload, string, string, string, string, string) {
	if len(s.googleClientIDs) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google OAuth not configured"})
		return nil, "", "", "", "", ""
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	payload, err := validateGoogleIDToken(ctx, idToken, s.googleClientIDs)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return nil, "", "", "", "", ""
	}

	email, name, picture, familyName, givenName, err := extractUserClaims(payload)
	if err != nil || email == "" {
		s.logger.Warn("missing or invalid email claim in JWT")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token: missing email"})
		return nil, "", "", "", "", ""
	}

	s.logger.Info("validated user email", slog.String("email", email))
	return payload, email, name, picture, familyName, givenName
}

func (s *Service) processGoogleAuthUser(
	email, name, picture, familyName, givenName string,
	c *gin.Context,
) (authUser *User, authType string) {
	existingUser, err := s.getUserByEmail(c.Request.Context(), email)
	if err != nil {
		s.logger.Error("error checking user", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return nil, ""
	}

	authType = constants.DefaultUserTheme
	var finalUser *User

	if existingUser == nil {
		finalUser, err = s.handleNewUser(c.Request.Context(), email, name, picture, familyName, givenName)
		if err != nil {
			s.logger.Error("error creating user", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return nil, ""
		}
	} else {
		finalUser, err = s.handleExistingUser(c.Request.Context(), email)
		if err != nil {
			s.logger.Error("error handling existing user", slog.Any("error", err))
		}
		if finalUser != nil {
			authType = finalUser.Theme
		}
	}

	authUser = finalUser
	return authUser, authType
}

func (s *Service) setupAuthSession(email string, c *gin.Context) {
	csrfToken := setupCSRFToken(c, s.cookieSecure)
	s.createSession(email, csrfToken, c)
	setAuthCookie(c, s.cookieSecure)
}

func (s *Service) ValidateTokenHandler(c *gin.Context) {
	setCacheControlHeaders(c)

	if s.isDevelopment {
		s.logger.Info("development mode: returning mock user for validation")
		c.JSON(http.StatusOK, gin.H{
			"valid":     true,
			"auth_type": "development",
			"user": gin.H{
				"uuid":        "dev-uuid-12345",
				"email":       "testuser@example.com",
				"name":        "Test User",
				"picture":     "https://via.placeholder.com/150",
				"family_name": "User",
				"given_name":  "Test",
				"theme":       constants.DefaultUserTheme,
			},
		})
		return
	}

	if s.sessionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session service unavailable"})
		return
	}

	userSession, err := s.sessionService.GetUserSession(c.Request)
	if err != nil || userSession == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return
	}

	email := userSession.UserID
	userData, err := s.getUserByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	if userData == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userData = s.ensureUserUUID(c.Request.Context(), userData, email)
	updateSessionLastActive(userSession, c)
	if err := s.sessionService.ExtendUserSession(userSession, c.Request, c.Writer); err != nil {
		s.logger.Warn("failed to extend user session", slog.Any("error", err))
	}

	c.JSON(http.StatusOK, buildValidateTokenResponse(userData, "session"))
}

func (s *Service) LogoutHandler(c *gin.Context) {
	c.SetCookie(
		"auth_token",
		"",
		-1, // Expire immediately
		"/",
		"",
		s.cookieSecure,
		true,
	)

	c.SetCookie(
		"csrf_token",
		"",
		-1,
		"/",
		"",
		s.cookieSecure,
		false,
	)

	if s.sessionService != nil {
		userSession, err := s.sessionService.GetUserSession(c.Request)
		if err == nil && userSession != nil {
			if err := s.sessionService.ClearUserSession(userSession, c.Writer); err != nil {
				s.logger.Warn("failed to clear user session", slog.Any("error", err))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (s *Service) GetUserSessionsHandler(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var currentSessionID string
	if session, exists := c.Get("session"); exists {
		if userSession, ok := session.(*user.Session); ok {
			currentSessionID = userSession.ID
		}
	}

	if s.sessionStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session store unavailable"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userSessions, err := s.sessionStore.GetSessionsByUserID(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sessions"})
		return
	}

	sessionInfos := make([]SessionInfo, 0, len(userSessions))
	for _, session := range userSessions {
		sessionInfo := SessionInfo{
			SessionID: session.ID,
			ExpiresAt: session.ExpiresAt,
			Current:   session.ID == currentSessionID,
		}

		var sessionData SessionJSON
		if err := json.Unmarshal([]byte(session.JSON), &sessionData); err == nil {
			sessionInfo.UserAgent = sessionData.UserAgent
			sessionInfo.IPAddress = sessionData.IPAddress
			sessionInfo.LastActive = sessionData.LastActive
		}

		sessionInfos = append(sessionInfos, sessionInfo)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessionInfos,
		"count":    len(sessionInfos),
	})
}

func (s *Service) TerminateSessionHandler(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	if s.sessionStore == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session store unavailable"})
		return
	}

	session, err := s.sessionStore.FetchValidUserSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if session.UserID != emailStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot terminate another user's session"})
		return
	}

	err = s.sessionStore.DeleteUserSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

func (s *Service) GetWebSocketTokenHandler(c *gin.Context) {
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Generate secure WebSocket token using JWT with short expiry
	expiresAt := time.Now().Add(5 * time.Minute)
	claims := TokenClaims{
		Email: emailStr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "websocket",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		s.logger.Error("failed to sign websocket token", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      signedToken,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (s *Service) CreateDevSession(email string, c *gin.Context) error {
	if s.sessionService == nil {
		return fmt.Errorf("session service not initialized")
	}

	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return fmt.Errorf("failed to generate CSRF token: %w", err)
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
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	userSession, err := s.sessionService.IssueUserSession(email, string(jsonBytes), c.Writer)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.Set("session", userSession)
	c.Header("X-CSRF-Token", csrfToken)

	return nil
}

func (s *Service) SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userSession, err := s.sessionService.GetUserSession(c.Request)
		if err != nil {
			s.logger.Warn("error fetching session", slog.Any("error", err))
		}

		if userSession != nil {
			c.Set("session", userSession)

			if err := s.sessionService.ExtendUserSession(userSession, c.Request, c.Writer); err != nil {
				s.logger.Warn("failed to extend user session", slog.Any("error", err))
			}

			var sessionData SessionJSON
			if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
				if sessionData.CSRF != "" {
					c.Header("X-CSRF-Token", sessionData.CSRF)
				}
			}

			c.Set("email", userSession.UserID)
		}

		c.Next()
	}
}
