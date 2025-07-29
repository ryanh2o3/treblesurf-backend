package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/adam-hanna/sessions"
	"github.com/adam-hanna/sessions/auth"
	"github.com/adam-hanna/sessions/transport"
	"github.com/adam-hanna/sessions/user"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gin-gonic/gin"
)

// DynamoDBStore implements the session store interface using DynamoDB
type DynamoDBStore struct {
    db        *dynamodb.DynamoDB
    tableName string
}

// NewDynamoDBStore creates a new DynamoDB session store
func NewDynamoDBStore(db *dynamodb.DynamoDB, tableName string) *DynamoDBStore {
    return &DynamoDBStore{
        db:        db,
        tableName: tableName,
    }
}

// SessionItem represents a session stored in DynamoDB
type SessionItem struct {
    SessionID  string    `json:"session_id"`
    UserID     string    `json:"user_id"`  // Will store email as UserID
    ExpiresAt  time.Time `json:"expires_at"`
    JSON       string    `json:"json_data"`
}

// SaveUserSession saves a user session to DynamoDB
func (s *DynamoDBStore) SaveUserSession(userSession *user.Session) error {
    sessionItem := SessionItem{
        SessionID: userSession.ID,
        UserID:    userSession.UserID,
        ExpiresAt: userSession.ExpiresAt,
        JSON:      userSession.JSON,
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

// DeleteUserSession deletes a session from DynamoDB
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

// FetchValidUserSession fetches a valid session from DynamoDB
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
        return nil, nil // Session not found
    }

    sessionItem := &SessionItem{}
    err = dynamodbattribute.UnmarshalMap(result.Item, sessionItem)
    if err != nil {
        return nil, err
    }

    // Check if the session is expired
    if time.Now().After(sessionItem.ExpiresAt) {
        // Session is expired, delete it
        _ = s.DeleteUserSession(sessionID)
        return nil, nil
    }

    // Convert to user.Session
    userSession := &user.Session{
        ID:        sessionItem.SessionID,
        UserID:    sessionItem.UserID,
        ExpiresAt: sessionItem.ExpiresAt,
        JSON:      sessionItem.JSON,
    }

    return userSession, nil
}

// EnsureSessionsTable creates the Sessions table if it doesn't exist
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

// SessionJSON defines the structure for session data
type SessionJSON struct {
    CSRF string `json:"csrf"`
    // Add any other session data you want to store
}

var sessionService *sessions.Service

// InitSessionService initializes the session service with DynamoDB store
func InitSessionService() error {
    // Create DynamoDB store
    sessionStore := NewDynamoDBStore(db, "Sessions")
    
    // Ensure the Sessions table exists
    if err := sessionStore.EnsureSessionsTable(); err != nil {
        return fmt.Errorf("failed to ensure Sessions table: %w", err)
    }

    // Create auth service with your JWT secret
    // Using the same secret for consistency with your current auth
    authService, err := auth.New(auth.Options{
        Key: jwtSecret,
    })
    if err != nil {
        return err
    }

    // Create transport service (cookies)
    transportService := transport.New(transport.Options{
        HTTPOnly: true,
        Secure:   true,
        CookiePath:     "/",
        CookieName:     "session_id",
    })

    // Initialize the session service
    sessionService = sessions.New(sessionStore, authService, transportService, sessions.Options{
        ExpirationDuration: 30 * 24 * time.Hour, // Match your JWT token expiration
    })

    return nil
}

// SessionMiddleware attaches user session to the context if present
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