package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

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