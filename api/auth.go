package api

import (
	"context"
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

// Replace this with a proper secret from environment variable in production
var jwtSecret = []byte(getEnvOrDefault("JWT_SECRET", "your_super_secret"))

func getEnvOrDefault(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
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

    // Create a new JWT token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "email": email,
        "exp":   time.Now().Add(time.Hour * 72).Unix(),
    })
    
    tokenString, err := token.SignedString(jwtSecret)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

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
        if tokenStr == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
            return
        }

        // Remove 'Bearer ' prefix if present
        if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
            tokenStr = tokenStr[7:]
        }

        token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })
        
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            return
        }

        // Add claims to context for later use if needed
        claims, ok := token.Claims.(jwt.MapClaims)
        if ok {
            c.Set("email", claims["email"])
        }

        c.Next()
    }
}

func ValidateTokenHandler(c *gin.Context) {
    tokenStr := c.GetHeader("Authorization")
    if tokenStr == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
        return
    }

    // Remove 'Bearer ' prefix if present
    if len(tokenStr) > 7 && tokenStr[:7] == "Bearer " {
        tokenStr = tokenStr[7:]
    }

    // Parse the token
    token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
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

    // Get claims from token
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token claims"})
        return
    }

    email, ok := claims["email"].(string)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Email claim missing"})
        return
    }

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

    // Check expiration time - refresh if less than 24 hours remaining
    expirationTime, ok := claims["exp"].(float64)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid expiration claim"})
        return
    }

    expTime := time.Unix(int64(expirationTime), 0)
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
    }

    // If token expires in less than 24 hours, generate a new one
    if remainingTime < 24*time.Hour {
        newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
            "email": email,
            "exp":   time.Now().Add(time.Hour * 72).Unix(),
        })
        
        newTokenString, err := newToken.SignedString(jwtSecret)
        if err != nil {
            // Still return the valid status with the old token
            c.JSON(http.StatusOK, response)
            return
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