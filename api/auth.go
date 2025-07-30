package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
)

type TokenClaims struct {
    Email string `json:"email"`
    jwt.RegisteredClaims
}

// Replace this with a proper secret from environment variable in production
var jwtSecret []byte

func init() {
    secretKey := os.Getenv("JWT_SECRET")
    if secretKey == "" {
        log.Fatal("JWT_SECRET environment variable must be set")
    }
    jwtSecret = []byte(secretKey)
}

// TokenRequest defines the structure for incoming token validation requests
type TokenRequest struct {
    IDToken string `json:"id_token"`
}

// GoogleAuthHandler validates Google ID tokens and issues JWT tokens
func GoogleAuthHandler(c *gin.Context) {
    var req TokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    // Replace with your actual Google Client ID
    clientID := "667754725634-pmth2mlieh3jfebl8ujkjji7nqqih3tf.apps.googleusercontent.com"
    iosClientID := "667754725634-9tck0kii14dm6d1e0u05preefmfppp5b.apps.googleusercontent.com"

	clientIDs := map[string]bool{
		clientID: true,
	}

	clientIDs[iosClientID] = true
	clientIDs[clientID] = true

    var payload *idtoken.Payload
    var err error
    
    for id := range clientIDs {
        payload, err = idtoken.Validate(context.Background(), req.IDToken, id)
        if err == nil {
            break
        }
    }
    
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
        return
    }

    email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	picture := payload.Claims["picture"].(string)
	familyName := payload.Claims["family_name"].(string)
	givenName := payload.Claims["given_name"].(string)
	log.Printf("User email: %s", email)
	theme := "dark"
	// Check if user exists in database
    existingUser, err := getUserByEmail(email)
    if err != nil {
        log.Printf("Error checking user: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
        return
    }

    if existingUser == nil {
        // Create new user
        newUser := User{
            Email:      email,
            Name:       name,
            Picture:    picture,
            FamilyName: familyName,
            GivenName:  givenName,
        }
        
        if err := createUser(newUser); err != nil {
            log.Printf("Error creating user: %v", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
            return
        }
        
        log.Printf("Created new user: %s", email)
    } else {
        // Update last login time
        if err := updateUserLastLogin(email); err != nil {
            log.Printf("Error updating last login: %v", err)
            // Continue anyway, not a critical error
        }
		theme = existingUser.Theme
        
        log.Printf("User logged in: %s", email)
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "treblesurf-api",
			Subject:   email,
		},
	})
    
    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

	// In GoogleAuthHandler, after setting the auth cookie:
	csrfToken, err := GenerateCSRFToken()
	if err == nil {
		c.SetCookie(
			"csrf_token",
			csrfToken,
			int(24*time.Hour.Seconds()),
			"/",
			"",
			true,  // Secure
			false, // Not HTTP-only (JS needs to access it)
		)
		// Include it in response for SPA to use
		c.Header("X-CSRF-Token", csrfToken)
	}

	c.SetCookie(
        "auth_token",
        tokenString,
        int(24*time.Hour.Seconds()),
        "/",
        "",
        true,  // Secure (HTTPS only)
        true,  // HTTP-only
    )

    if sessionService != nil {
    // Generate a session with CSRF token
    sessionData := SessionJSON{
        CSRF: csrfToken,
        UserAgent: c.Request.UserAgent(),
        IPAddress: getClientIP(c),
        CreatedAt: time.Now(),
        LastActive: time.Now(),
    }
    jsonBytes, err := json.Marshal(sessionData)
    err = nil
        _, err = sessionService.IssueUserSession(email, string(jsonBytes), c.Writer)
        if err != nil {
            // Just log the error, don't fail the request
            log.Printf("Error creating session: %v", err)
        }
    }

    c.JSON(http.StatusOK, gin.H{
        "token": tokenString,
        "user": gin.H{
            "email": email,
            "name": name,
			"picture": picture,
			"family_name": familyName,
			"given_name": givenName,
			"theme": theme,
        },
    })
}

func ClientTypeMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Detect client type based on headers or user agent
        clientType := c.GetHeader("X-Client-Type")
        userAgent := c.Request.UserAgent()
        isAppClient := clientType == "app" || 
            (userAgent != "" && strings.Contains(strings.ToLower(userAgent), "app"))

        // Set client type in context for downstream use
        c.Set("isAppClient", isAppClient)
        c.Next()
    }
}

// AdaptiveAuthMiddleware uses either app or web auth based on client type
func AdaptiveAuthMiddleware() gin.HandlerFunc {
    // Create instances of both middleware handlers
    webAuth := WebAuthMiddleware()
    appAuth := AuthMiddleware()
    
    // Return a handler that will choose between them
    return func(c *gin.Context) {
        // Get client type from context
        isAppClient, exists := c.Get("isAppClient")
        if !exists {
            // If no client type is set, default to web
            webAuth(c)
            return
        }
        
        // Use appropriate auth middleware based on client type
        if isAppClient.(bool) {
            appAuth(c)
        } else {
            webAuth(c)
        }
    }
}

// WebAuthMiddleware is for web clients - uses sessions ONLY, no JWT fallback
func WebAuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        c.Header("Pragma", "no-cache")

        // Only try session auth, no fallback to JWT
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            if err == nil && userSession != nil {
                // Session is valid
                c.Set("email", userSession.UserID)
                c.Set("session", userSession)
                
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
                    sessionData.LastActive = time.Now()
                    updatedJSON, _ := json.Marshal(sessionData)
                    userSession.JSON = string(updatedJSON)

                    if sessionData.CSRF != "" {
                        c.Header("X-CSRF-Token", sessionData.CSRF)
                    }
                }
                
                // Extend session
                _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)
                
                c.Next()
                return
            }
        }

        // If no valid session, deny access - don't fall back to JWT
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
    }
}

// AuthMiddleware checks if the request has a valid JWT token (for apps only)
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenStr := c.GetHeader("Authorization")

        // For apps, we ONLY accept Authorization header, no cookies
        if tokenStr == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
            return
        }

        // Remove 'Bearer ' prefix if present
        if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
            tokenStr = tokenStr[7:]
        }

        // Parse token with claims
        claims := &TokenClaims{}
        token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            return
        }

        c.Set("email", claims.Email)
        c.Next()
    }
}

// Add a new middleware that tries both JWT and session authentication

// CombinedAuthMiddleware checks for either valid JWT or valid session
// func CombinedAuthMiddleware() gin.HandlerFunc {
//     return func(c *gin.Context) {
//         c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
//         c.Header("Pragma", "no-cache")

//         // Try session auth first
//         if sessionService != nil {
//             userSession, err := sessionService.GetUserSession(c.Request)
//             if err == nil && userSession != nil {
//                 // Session is valid
//                 c.Set("email", userSession.UserID)
//                 c.Set("session", userSession)
                
//                 // Parse session JSON to get CSRF token
//                 var sessionData SessionJSON
//                 if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
//                     sessionData.LastActive = time.Now()
//                     updatedJSON, _ := json.Marshal(sessionData)
//                     userSession.JSON = string(updatedJSON)

//                     if sessionData.CSRF != "" {
//                         c.Header("X-CSRF-Token", sessionData.CSRF)
//                     }
//                 }
                
//                 // Extend session
//                 _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)
                
//                 c.Next()
//                 return
//             }
//         }

//         tokenStr := c.GetHeader("Authorization")

//         // If not in header, check cookie
//         if tokenStr == "" {
//             tokenCookie, err := c.Cookie("auth_token")
//             if err == nil {
//                 tokenStr = tokenCookie
//             }
//         }

//         if tokenStr != "" {
//             // Remove 'Bearer ' prefix if present
//             if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
//                 tokenStr = tokenStr[7:]
//             }

//             // Parse token with claims
//             claims := &TokenClaims{}
//             token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
//                 if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
//                     return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
//                 }
//                 return jwtSecret, nil
//             })
            
//             if err == nil && token.Valid {
//                 // JWT is valid, set email in context
//                 c.Set("email", claims.Email)
                
//                 // Check token expiration time for renewal
//                 if claims.ExpiresAt != nil {
//                     // Same token renewal logic as your AuthMiddleware
//                     expTime := claims.ExpiresAt.Time
//                     remainingTime := time.Until(expTime)
                    
//                     if remainingTime < 24*time.Hour {
//                         // Your existing token renewal logic
//                         // ...
//                     }
//                 }
                
//                 c.Next()
//                 return
//             }
//         }

//         // Both auth methods failed
//         c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
//     }
// }

func ValidateTokenHandler(c *gin.Context) {
    // Add cache control headers to prevent browser caching
    c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
    c.Header("Pragma", "no-cache")
    c.Header("Expires", "0")

    // Determine client type (app or web)
    clientType := c.GetHeader("X-Client-Type")
    userAgent := c.Request.UserAgent()
    isAppClient := clientType == "app" || 
        (userAgent != "" && strings.Contains(strings.ToLower(userAgent), "app"))

    // Handle web client authentication (sessions only)
    if !isAppClient {
        if sessionService == nil {
            c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Session service unavailable"})
            return
        }

        userSession, err := sessionService.GetUserSession(c.Request)
        if err != nil || userSession == nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
            return
        }
        
        // Session is valid, get user data
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

        // Get CSRF token from session and update last active time
        var sessionData SessionJSON
        if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
            sessionData.LastActive = time.Now()
            updatedJSON, _ := json.Marshal(sessionData)
            userSession.JSON = string(updatedJSON)
            
            if sessionData.CSRF != "" {
                c.Header("X-CSRF-Token", sessionData.CSRF)
            }
        }

        // Extend session validity
        _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)

        // Return user data for web client
        c.JSON(http.StatusOK, gin.H{
            "valid": true,
            "auth_type": "session",
            "user": gin.H{
                "email":       user.Email,
                "name":        user.Name,
                "picture":     user.Picture,
                "family_name": user.FamilyName,
                "given_name":  user.GivenName,
                "theme":       user.Theme,
            },
        })
        return
    }
    
    // Handle app client authentication (JWT only)
    tokenStr := c.GetHeader("Authorization")
    if tokenStr == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing Authorization header"})
        return
    }

    // Remove 'Bearer ' prefix if present
    if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
        tokenStr = tokenStr[7:]
    }

    // Parse the token with claims
    claims := &TokenClaims{}
    token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return jwtSecret, nil
    })

    // Check token validity
    if err != nil || !token.Valid {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
        return
    }
    
    // Token is valid, get user data
    email := claims.Email
    user, err := getUserByEmail(email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
        return
    }

    if user == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    // Handle token renewal if needed
    var tokenRefreshed bool
    var newTokenString string
    
    if claims.ExpiresAt != nil {
        expTime := claims.ExpiresAt.Time
        remainingTime := time.Until(expTime)

        // If token expires in less than 24 hours, generate a new one
        if remainingTime < 24*time.Hour {
            newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
                Email: email,
                RegisteredClaims: jwt.RegisteredClaims{
                    ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
                    IssuedAt:  jwt.NewNumericDate(time.Now()),
                    NotBefore: jwt.NewNumericDate(time.Now()),
                    Issuer:    "treblesurf-api",
                    Subject:   email,
                },
            })
            
            newTokenString, err = newToken.SignedString(jwtSecret)
            if err == nil {
                tokenRefreshed = true
            }
        }
    }

    // Return response for app client
    response := gin.H{
        "valid": true,
        "auth_type": "jwt",
        "user": gin.H{
            "email":       user.Email,
            "name":        user.Name,
            "picture":     user.Picture,
            "family_name": user.FamilyName,
            "given_name":  user.GivenName,
            "theme":       user.Theme,
        },
    }
    
    if tokenRefreshed {
        response["token"] = newTokenString
        response["token_refreshed"] = true
    } else {
        response["token"] = tokenStr
    }
    
    c.JSON(http.StatusOK, response)
}

type User struct {
    Email      string `json:"email"`
    Name       string `json:"name"`
    Picture    string `json:"picture"`
    FamilyName string `json:"family_name"`
    GivenName  string `json:"given_name"`
    CreatedAt  string `json:"created_at"`
    LastLogin  string `json:"last_login"`
	Theme    string `json:"theme"`
}

// getUserByEmail checks if a user exists in the Users table
func getUserByEmail(email string) (*User, error) {
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

// createUser creates a new user in the Users table
func createUser(user User) error {
    now := time.Now().UTC().Format(time.RFC3339)
    user.CreatedAt = now
    user.LastLogin = now

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

// updateUserLastLogin updates the last login time for a user
func updateUserLastLogin(email string) error {
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

// GenerateCSRFToken creates a secure random token for CSRF protection
func GenerateCSRFToken() (string, error) {
    bytes := make([]byte, 32)
    _, err := rand.Read(bytes)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(bytes), nil
}

// CSRFMiddleware adds CSRF protection to routes that modify state
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only check POST, PUT, DELETE requests
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
            c.Next()
            return
        }

        // Get token from header
        clientToken := c.GetHeader("X-CSRF-Token")
        
        // Get token from session/cookie
        serverToken, err := c.Cookie("csrf_token")
        if err != nil {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token missing",
            })
            return
        }

        // Validate token
        if clientToken == "" || clientToken != serverToken {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token invalid",
            })
            return
        }

        c.Next()
    }
}

// LogoutHandler clears authentication cookies
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

type SessionInfo struct {
    SessionID  string    `json:"session_id"`
    ExpiresAt  time.Time `json:"expires_at"`
    Current    bool      `json:"current"`
    LastActive time.Time `json:"last_active,omitempty"`
    UserAgent  string    `json:"user_agent,omitempty"`
    IPAddress  string    `json:"ip_address,omitempty"`
}

// GetUserSessionsHandler retrieves all active sessions for the current user
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

    userSessions, err := sessionStoreDB.GetSessionsByUserID(email.(string))
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

// TerminateSessionHandler terminates a specific session
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

    // If session not found or doesn't belong to current user
    if session == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
        return
    }

    // Verify ownership
    if session.UserID != email.(string) {
        c.JSON(http.StatusForbidden, gin.H{"error": "You can only terminate your own sessions"})
        return
    }

    // Check if this is the current session
    currentSession, exists := c.Get("session")
    if exists {
        if userSession, ok := currentSession.(*user.Session); ok && userSession.ID == sessionID {
            // This is the current session, use ClearUserSession which handles cookies too
            err = sessionService.ClearUserSession(session, c.Writer)
        } else {
            // Just delete from database
            err = sessionStoreDB.DeleteUserSession(sessionID)
        }
    } else {
        // Just delete from database
        err = sessionStoreDB.DeleteUserSession(sessionID)
    }

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to terminate session"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Session terminated successfully"})
}

func getClientIP(c *gin.Context) string {
    // Try CloudFront-specific header first
    if ip := c.Request.Header.Get("CloudFront-Viewer-Address"); ip != "" {
        return strings.Split(ip, ":")[0] 
    }

    return c.ClientIP() 
}