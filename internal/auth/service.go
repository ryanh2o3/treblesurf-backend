package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"treblesurf-backend/internal/constants"

	"github.com/adam-hanna/sessions"
	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/transport"
	"github.com/adam-hanna/sessions/user"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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
	CSRF       string    `json:"csrf"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	LastActive time.Time `json:"last_active"`
}

type SessionInfo struct {
	SessionID  string    `json:"session_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
	LastActive time.Time `json:"last_active,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
}

// DynamoDBStore implements the session store interface using DynamoDB
type DynamoDBStore struct {
	db        *dynamodb.DynamoDB
	tableName string
}

func NewDynamoDBStore(db *dynamodb.DynamoDB, tableName string) *DynamoDBStore {
	return &DynamoDBStore{
		db:        db,
		tableName: tableName,
	}
}

type SessionItem struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"` // Will store email as UserID
	ExpiresAt time.Time `json:"expires_at"`
	JSON      string    `json:"json_data"`
	TTL       int64     `json:"ttl"`
}

func (s *DynamoDBStore) SaveUserSession(userSession *user.Session) error {
	sessionItem := SessionItem{
		SessionID: userSession.ID,
		UserID:    userSession.UserID,
		ExpiresAt: userSession.ExpiresAt,
		JSON:      userSession.JSON,
		TTL:       userSession.ExpiresAt.Unix(),
	}

	item, err := dynamodbattribute.MarshalMap(sessionItem)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	}

	_, err = s.db.PutItem(input)
	return err
}

func (s *DynamoDBStore) DeleteUserSession(sessionID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"session_id": {
				S: aws.String(sessionID),
			},
		},
	}

	_, err := s.db.DeleteItem(input)
	return err
}

func (s *DynamoDBStore) FetchValidUserSession(sessionID string) (*user.Session, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"session_id": {
				S: aws.String(sessionID),
			},
		},
	}

	result, err := s.db.GetItem(input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var sessionItem SessionItem
	err = dynamodbattribute.UnmarshalMap(result.Item, &sessionItem)
	if err != nil {
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(sessionItem.ExpiresAt) {
		return nil, nil
	}

	return &user.Session{
		ID:        sessionItem.SessionID,
		UserID:    sessionItem.UserID,
		ExpiresAt: sessionItem.ExpiresAt,
		JSON:      sessionItem.JSON,
	}, nil
}

func (s *DynamoDBStore) EnableTTL() error {
	input := &dynamodb.UpdateTimeToLiveInput{
		TableName: aws.String(s.tableName),
		TimeToLiveSpecification: &dynamodb.TimeToLiveSpecification{
			AttributeName: aws.String("ttl"),
			Enabled:       aws.Bool(true),
		},
	}

	_, err := s.db.UpdateTimeToLive(input)
	return err
}

func (s *DynamoDBStore) GetSessionsByUserID(userID string) ([]*user.Session, error) {
	// Create a query input that filters by UserID
	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uid": {
				S: aws.String(userID),
			},
		},
	}

	result, err := s.db.Scan(input)
	if err != nil {
		return nil, err
	}

	sessions := make([]*user.Session, 0)
	now := time.Now()

	for _, item := range result.Items {
		sessionItem := &SessionItem{}
		if err := dynamodbattribute.UnmarshalMap(item, sessionItem); err != nil {
			continue
		}

		// Skip expired sessions
		if now.After(sessionItem.ExpiresAt) {
			continue
		}

		// Convert to user.Session
		userSession := &user.Session{
			ID:        sessionItem.SessionID,
			UserID:    sessionItem.UserID,
			ExpiresAt: sessionItem.ExpiresAt,
			JSON:      sessionItem.JSON,
		}

		sessions = append(sessions, userSession)
	}

	return sessions, nil
}

func (s *DynamoDBStore) EnsureSessionsTable() error {
	// Check if table exists
	tables, err := s.db.ListTables(&dynamodb.ListTablesInput{})
	if err != nil {
		return err
	}

	for _, tableName := range tables.TableNames {
		if *tableName == s.tableName {
			return nil // Table already exists
		}
	}

	// Table doesn't exist, create it
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("session_id"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("session_id"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(s.tableName),
	}

	_, err = s.db.CreateTable(input)
	return err
}

// Global variables
var jwtSecret []byte
var db *dynamodb.DynamoDB
var sessionService *sessions.Service
var sessionStoreDB *DynamoDBStore

func InitJWTSecret() {
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		log.Fatal("JWT_SECRET environment variable must be set")
	}
	jwtSecret = []byte(secretKey)
}

func SetDynamoDB(dynamoDB *dynamodb.DynamoDB) {
	db = dynamoDB
}

func InitSessionService() error {
	if db == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	// Create DynamoDB store
	sessionStore := NewDynamoDBStore(db, "Sessions")
	sessionStoreDB = sessionStore

	// Ensure the Sessions table exists
	if err := sessionStore.EnsureSessionsTable(); err != nil {
		return fmt.Errorf("failed to ensure Sessions table: %w", err)
	}

	// Enable TTL on the Sessions table
	if err := sessionStore.EnableTTL(); err != nil {
		log.Printf("Warning: Failed to enable TTL on Sessions table: %v", err)
	}

	// Create auth service with your JWT secret
	authService, err := auth.New(auth.Options{
		Key: jwtSecret,
	})
	if err != nil {
		return err
	}

	// Create transport service (cookies)
	transportService := transport.New(transport.Options{
		HTTPOnly:   true,
		Secure:     true,
		CookiePath: "/",
		CookieName: "session_id",
	})

	// Initialize the session service
	sessionService = sessions.New(sessionStore, authService, transportService, sessions.Options{
		ExpirationDuration: 30 * 24 * time.Hour, // Match your JWT token expiration
	})

	return nil
}

// Helper functions
func getUserByEmail(email string) (*User, error) {
	if db == nil {
		return nil, fmt.Errorf("DynamoDB client not initialized")
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
	}

	result, err := db.GetItem(input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil // User not found
	}

	user := &User{}
	err = dynamodbattribute.UnmarshalMap(result.Item, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func createUser(user User) error {
	if db == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	uuid, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}
	user.UUID = uuid.String()

	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = now
	user.LastLogin = now
	user.Theme = "dark" // Default theme

	item, err := dynamodbattribute.MarshalMap(user)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String("Users"),
		Item:      item,
	}

	_, err = db.PutItem(input)
	return err
}

func updateUserLastLogin(email string) error {
	if db == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
		UpdateExpression: aws.String("set last_login = :time"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":time": {
				S: aws.String(now),
			},
		},
	}

	_, err := db.UpdateItem(input)
	return err
}

func ensureUserHasUUID(email string) error {
	if db == nil {
		return fmt.Errorf("DynamoDB client not initialized")
	}

	// Get the current user to check if they have a UUID
	user, err := getUserByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return fmt.Errorf("user not found")
	}

	// If user already has a UUID, no need to update
	if user.UUID != "" {
		return nil
	}

	// Generate a new UUID for the user
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Update the user with the new UUID using expression attribute names for reserved keywords
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("Users"),
		Key: map[string]*dynamodb.AttributeValue{
			"email": {
				S: aws.String(email),
			},
		},
		UpdateExpression: aws.String("set #uuid = :uuid"),
		ExpressionAttributeNames: map[string]*string{
			"#uuid": aws.String("uuid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":uuid": {
				S: aws.String(newUUID.String()),
			},
		},
	}

	_, err = db.UpdateItem(input)
	if err != nil {
		return fmt.Errorf("failed to update user with UUID: %w", err)
	}

	log.Printf("Generated and assigned UUID %s to user %s", newUUID.String(), email)
	return nil
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

func GoogleAuthHandler(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	payload, email, name, picture, familyName, givenName := validateAndExtractGoogleClaims(c, req.IDToken)
	if payload == nil {
		return
	}

	finalUser, theme := processGoogleAuthUser(email, name, picture, familyName, givenName, c)
	if finalUser == nil {
		return
	}

	setupAuthSession(email, c)
	c.JSON(http.StatusOK, gin.H{
		"user": buildUserResponse(finalUser, email, name, picture, familyName, givenName, theme),
	})
}

func validateAndExtractGoogleClaims(
	c *gin.Context,
	idToken string,
) (*idtoken.Payload, string, string, string, string, string) {
	clientIDs, err := getGoogleClientIDs()
	if err != nil || clientIDs == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google OAuth not configured"})
		return nil, "", "", "", "", ""
	}

	payload, err := validateGoogleIDToken(idToken, clientIDs)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return nil, "", "", "", "", ""
	}

	email, name, picture, familyName, givenName, err := extractUserClaims(payload)
	if err != nil || email == "" {
		log.Printf("Missing or invalid email claim in JWT")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token: missing email"})
		return nil, "", "", "", "", ""
	}

	log.Printf("User email: %s", email)
	return payload, email, name, picture, familyName, givenName
}

func processGoogleAuthUser(email, name, picture, familyName, givenName string, c *gin.Context) (*User, string) {
	existingUser, err := getUserByEmail(email)
	if err != nil {
		log.Printf("Error checking user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return nil, ""
	}

	theme := "dark"
	var finalUser *User

	if existingUser == nil {
		finalUser, err = handleNewUser(email, name, picture, familyName, givenName)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return nil, ""
		}
	} else {
		finalUser, err = handleExistingUser(email)
		if err != nil {
			log.Printf("Error handling existing user: %v", err)
		}
		if finalUser != nil {
			theme = finalUser.Theme
		}
	}

	return finalUser, theme
}

func setupAuthSession(email string, c *gin.Context) {
	csrfToken := setupCSRFToken(c)
	createSession(email, csrfToken, c)
	setAuthCookie(c)
}

func ValidateTokenHandler(c *gin.Context) {
	setCacheControlHeaders(c)

	if os.Getenv("GO_ENV") == constants.EnvDevelopment {
		log.Println("Development mode: returning mock user for validation")
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
				"theme":       "dark",
			},
		})
		return
	}

	if sessionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session service unavailable"})
		return
	}

	userSession, err := sessionService.GetUserSession(c.Request)
	if err != nil || userSession == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
		return
	}

	email := userSession.UserID
	user, err := getUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	user = ensureUserUUID(user, email)
	updateSessionLastActive(userSession, c)
	_ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)

	c.JSON(http.StatusOK, buildValidateTokenResponse(user, "session"))
}

func LogoutHandler(c *gin.Context) {
	// Clear auth cookie
	c.SetCookie(
		"auth_token",
		"",
		-1, // Expire immediately
		"/",
		"",
		true,
		true,
	)

	// Clear CSRF cookie
	c.SetCookie(
		"csrf_token",
		"",
		-1,
		"/",
		"",
		true,
		false,
	)

	if sessionService != nil {
		userSession, err := sessionService.GetUserSession(c.Request)
		if err == nil && userSession != nil {
			_ = sessionService.ClearUserSession(userSession, c.Writer)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func GetUserSessionsHandler(c *gin.Context) {
	// Get current user's email from the context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get the current session ID (if any)
	var currentSessionID string
	if session, exists := c.Get("session"); exists {
		if userSession, ok := session.(*user.Session); ok {
			currentSessionID = userSession.ID
		}
	}

	// Get all sessions for this user from DynamoDB
	if sessionStoreDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session store unavailable"})
		return
	}

	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	userSessions, err := sessionStoreDB.GetSessionsByUserID(emailStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sessions"})
		return
	}

	// Create a list of session info objects (with only the data we want to expose)
	sessionInfos := make([]SessionInfo, 0, len(userSessions))
	for _, session := range userSessions {
		// Parse JSON to get additional session metadata
		sessionInfo := SessionInfo{
			SessionID:  session.ID,
			ExpiresAt:  session.ExpiresAt,
			Current:    session.ID == currentSessionID,
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

func TerminateSessionHandler(c *gin.Context) {
	// Get current user's email from context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get the session ID to terminate
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

	if sessionStoreDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Session store unavailable"})
		return
	}

	// Get the session first to verify it belongs to this user
	session, err := sessionStoreDB.FetchValidUserSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Verify the session belongs to the current user
	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	if session.UserID != emailStr {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot terminate another user's session"})
		return
	}

	// Delete the session
	err = sessionStoreDB.DeleteUserSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

func GetWebSocketTokenHandler(c *gin.Context) {
	// Get current user's email from context
	email, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Generate a simple token for WebSocket authentication
	// In production, you might want to use JWT or a more secure method
	emailStr, ok := email.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	
	token := fmt.Sprintf("ws_%s_%d", emailStr, time.Now().Unix())

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	})
}

func CreateDevSession(email string, c *gin.Context) error {
	if sessionService == nil {
		return fmt.Errorf("session service not initialized")
	}

	// Generate CSRF token
	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	// Create session data
	sessionData := SessionJSON{
		CSRF:       csrfToken,
		UserAgent:  c.Request.UserAgent(),
		IPAddress:  getClientIP(c),
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	// Marshal the session data to JSON
	jsonBytes, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Create a session and attach it to the response
	userSession, err := sessionService.IssueUserSession(email, string(jsonBytes), c.Writer)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Set the session in the context so ValidateTokenHandler can find it
	c.Set("session", userSession)

	// Set the CSRF token in the header
	c.Header("X-CSRF-Token", csrfToken)

	return nil
}

func SessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userSession, err := sessionService.GetUserSession(c.Request)
		if err != nil {
			// Log the error but continue
			fmt.Printf("Error fetching session: %v\n", err)
		}

		if userSession != nil {
			// Store the session in the context
			c.Set("session", userSession)

			// Extend the session if needed
			_ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)

			// Parse session JSON to get CSRF token
			var sessionData SessionJSON
			if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
				if sessionData.CSRF != "" {
					c.Header("X-CSRF-Token", sessionData.CSRF)
				}
			}

			// Set email in context for compatibility with existing code
			c.Set("email", userSession.UserID)
		}

		c.Next()
	}
}