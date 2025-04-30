package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
    
    payload, err := idtoken.Validate(context.Background(), req.IDToken, clientID)
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
        
        log.Printf("User logged in: %s", email)
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
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
			int(72*time.Hour.Seconds()),
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
        int(72*time.Hour.Seconds()),
        "/",
        "",
        true,  // Secure (HTTPS only)
        true,  // HTTP-only
    )

    c.JSON(http.StatusOK, gin.H{
        "token": tokenString,
        "user": gin.H{
            "email": email,
            "name": name,
			"picture": picture,
			"family_name": familyName,
			"given_name": givenName,
        },
    })
}

// AuthMiddleware checks if the request has a valid JWT token
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenStr := c.GetHeader("Authorization")

		// If not in header, check cookie
        if tokenStr == "" {
            tokenCookie, err := c.Cookie("auth_token")
            if err == nil {
                tokenStr = tokenCookie
            }
        }

        if tokenStr == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
            return
        }

        // Remove 'Bearer ' prefix if present
        if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
            tokenStr = tokenStr[7:]
        }

        // Parse token with claims directly - removing the duplicate parsing
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

		// Check token expiration time
        if claims.ExpiresAt != nil {
            expTime := claims.ExpiresAt.Time
            remainingTime := time.Until(expTime)
            
            // If token expires in less than 24 hours, generate a new one
            if remainingTime < 24*time.Hour {
                newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, TokenClaims{
                    Email: claims.Email,
                    RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                        NotBefore: jwt.NewNumericDate(time.Now()),
                        Issuer:    "treblesurf-api",
                        Subject:   claims.Email,
                    },
                })
                
                newTokenString, err := newToken.SignedString(jwtSecret)
                if err == nil {
                    // Set the new auth token cookie
                    c.SetCookie(
                        "auth_token",
                        newTokenString,
                        int(72*time.Hour.Seconds()),
                        "/",
                        "",
                        true,  // Secure (HTTPS only)
                        true,  // HTTP-only
                    )
                    
                    // Also refresh the CSRF token
                    csrfToken, err := GenerateCSRFToken()
                    if err == nil {
                        c.SetCookie(
                            "csrf_token",
                            csrfToken,
                            int(72*time.Hour.Seconds()),
                            "/",
                            "",
                            true,  // Secure
                            false, // Not HTTP-only (JS needs to access it)
                        )
                        // Include it in response for SPA to use
                        c.Header("X-CSRF-Token", csrfToken)
                    }
                }
            }
        }

        c.Set("email", claims.Email)
        c.Next()
    }
}

func ValidateTokenHandler(c *gin.Context) {
    tokenStr := c.GetHeader("Authorization")

	// Check cookie if header is empty - ADD THIS
    if tokenStr == "" {
        tokenCookie, err := c.Cookie("auth_token")
        if err == nil {
            tokenStr = tokenCookie
        }
    }
    if tokenStr == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
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

    // Check if token is valid
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "details": err.Error()})
        return
    }

    if !token.Valid {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Token validation failed"})
        return
    }

    // Get email directly from structured claims
    email := claims.Email

    // Get user data
    user, err := getUserByEmail(email)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
        return
    }

    if user == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    if claims.ExpiresAt == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Missing expiration claim"})
		return
	}
	expTime := claims.ExpiresAt.Time
	remainingTime := time.Until(expTime)

    response := gin.H{
        "valid": true,
        "user": gin.H{
            "email":       user.Email,
            "name":        user.Name,
            "picture":     user.Picture,
            "family_name": user.FamilyName,
            "given_name":  user.GivenName,
        },
        "expires_in": int(remainingTime.Seconds()),
		"token": tokenStr,
    }

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
        
        newTokenString, err := newToken.SignedString(jwtSecret)
        if err != nil {
            // Still return the valid status with the old token
            c.JSON(http.StatusOK, response)
            return
        }
        
        // Set the new token in cookie - ADD THIS
        c.SetCookie(
            "auth_token",
            newTokenString,
            int(72*time.Hour.Seconds()),
            "/",
            "",
            true,  // Secure (HTTPS only)
            true,  // HTTP-only
        )
        
        // Generate new CSRF token too - ADD THIS
        csrfToken, err := GenerateCSRFToken()
        if err == nil {
            c.SetCookie(
                "csrf_token",
                csrfToken,
                int(72*time.Hour.Seconds()),
                "/",
                "",
                true,  // Secure
                false, // Not HTTP-only (JS needs to access it)
            )
            // Include it in response for SPA to use
            c.Header("X-CSRF-Token", csrfToken)
        }
        
        response["token"] = newTokenString
        response["token_refreshed"] = true
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
    
    c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}