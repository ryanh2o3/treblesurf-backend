package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/adam-hanna/sessions/user"
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
        c.Header("Pragma", "no-cache")

        // Log request details for debugging
        log.Printf("=== Auth Middleware ===")
        log.Printf("Path: %s", c.Request.URL.Path)
        log.Printf("User-Agent: %s", c.Request.UserAgent())
        log.Printf("Method: %s", c.Request.Method)
        log.Printf("Origin: %s", c.GetHeader("Origin"))
        log.Printf("Referer: %s", c.GetHeader("Referer"))

        // Only try session auth, no fallback to JWT
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            if err != nil {
                log.Printf("Session service error: %v", err)
            } else if userSession != nil {
                log.Printf("Valid session found for user: %s", userSession.UserID)
                // Session is valid
                c.Set("email", userSession.UserID)
                c.Set("session", userSession)
                
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err == nil {
                    sessionData.LastActive = time.Now()
                    
                    // Refresh CSRF token if it's getting old (older than 1 hour)
                    if time.Since(sessionData.CreatedAt) > time.Hour {
                        log.Printf("Refreshing CSRF token for user: %s", userSession.UserID)
                        newCSRFToken, err := GenerateCSRFToken()
                        if err == nil {
                            sessionData.CSRF = newCSRFToken
                            sessionData.CreatedAt = time.Now()
                            log.Printf("CSRF token refreshed successfully")
                        }
                    }
                    
                    updatedJSON, _ := json.Marshal(sessionData)
                    userSession.JSON = string(updatedJSON)

                    if sessionData.CSRF != "" {
                        c.Header("X-CSRF-Token", sessionData.CSRF)
                        log.Printf("CSRF token set in header: %s", sessionData.CSRF)
                    }
                }
                
                // Extend session
                _ = sessionService.ExtendUserSession(userSession, c.Request, c.Writer)
                
                c.Next()
                return
            } else {
                log.Printf("No valid session found")
            }
        } else {
            log.Printf("Session service is nil")
        }

        // If no valid session, deny access - don't fall back to JWT
        log.Printf("Authentication failed - denying access")
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
    }
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
        
        // Try to get token from session data first (preferred method)
        var serverToken string
        if sessionService != nil {
            userSession, err := sessionService.GetUserSession(c.Request)
            if err == nil && userSession != nil {
                // Parse session JSON to get CSRF token
                var sessionData SessionJSON
                if json.Unmarshal([]byte(userSession.JSON), &sessionData) == nil {
                    serverToken = sessionData.CSRF
                }
            }
        }
        
        // If no session token, try to get from cookie as fallback
        if serverToken == "" {
            if cookieToken, err := c.Cookie("csrf_token"); err == nil {
                serverToken = cookieToken
            }
        }
        
        // If still no server token, deny access
        if serverToken == "" {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token missing",
            })
            return
        }

        // Validate token
        if clientToken == "" || clientToken != serverToken {
            // Try to regenerate CSRF token if the token is invalid
            if clientToken != "" {
                // Try to get the user session and regenerate the token
                if sessionService != nil {
                    userSession, err := sessionService.GetUserSession(c.Request)
                    if err == nil && userSession != nil {
                        log.Printf("Attempting to regenerate CSRF token for user: %s", userSession.UserID)
                        
                        // Regenerate the CSRF token
                        if err := regenerateCSRFToken(userSession); err == nil {
                            // Parse the updated session to get the new token
                            var sessionData SessionJSON
                            if json.Unmarshal([]byte(userSession.JSON), &sessionData) == nil {
                                if sessionData.CSRF == clientToken {
                                    log.Printf("CSRF token regenerated and validated successfully")
                                    c.Next()
                                    return
                                }
                            }
                        }
                    }
                }
            }
            
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "CSRF token invalid",
            })
            return
        }

        c.Next()
    }
}

// regenerateCSRFToken regenerates a CSRF token for a user session
func regenerateCSRFToken(userSession *user.Session) error {
    if sessionService == nil {
        return fmt.Errorf("session service not available")
    }
    
    // Generate new CSRF token
    newCSRFToken, err := GenerateCSRFToken()
    if err != nil {
        return fmt.Errorf("failed to generate new CSRF token: %v", err)
    }
    
    // Parse existing session data
    var sessionData SessionJSON
    if err := json.Unmarshal([]byte(userSession.JSON), &sessionData); err != nil {
        return fmt.Errorf("failed to parse session data: %v", err)
    }
    
    // Update CSRF token and reset creation time
    sessionData.CSRF = newCSRFToken
    sessionData.CreatedAt = time.Now()
    sessionData.LastActive = time.Now()
    
    // Marshal updated data
    updatedJSON, err := json.Marshal(sessionData)
    if err != nil {
        return fmt.Errorf("failed to marshal updated session data: %v", err)
    }
    
    // Update session in storage using the session service
    userSession.JSON = string(updatedJSON)
    // Note: The session service will handle persistence when ExtendUserSession is called
    
    log.Printf("CSRF token regenerated successfully for user: %s", userSession.UserID)
    return nil
}