package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims represents JWT token claims
type TokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GetAudience implements jwt.Claims interface
func (c *TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.RegisteredClaims.GetAudience()
}

// GetExpirationTime implements jwt.Claims interface
func (c *TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetExpirationTime()
}

// GetIssuedAt implements jwt.Claims interface
func (c *TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetIssuedAt()
}

// GetIssuer implements jwt.Claims interface
func (c *TokenClaims) GetIssuer() (string, error) {
	return c.RegisteredClaims.GetIssuer()
}

// GetNotBefore implements jwt.Claims interface
func (c *TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.RegisteredClaims.GetNotBefore()
}

// GetSubject implements jwt.Claims interface
func (c *TokenClaims) GetSubject() (string, error) {
	return c.RegisteredClaims.GetSubject()
}
