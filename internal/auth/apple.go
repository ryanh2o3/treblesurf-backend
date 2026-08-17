package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleIssuer       = "https://appleid.apple.com"
	appleJWKSURL      = "https://appleid.apple.com/auth/keys"
	appleJWKSCacheTTL = time.Hour
)

var (
	errAppleInvalidToken  = errors.New("invalid apple identity token")
	errAppleWrongAudience = errors.New("apple identity token audience mismatch")
)

// AppleTokenRequest is the body for POST /auth/apple.
type AppleTokenRequest struct {
	IdentityToken     string `json:"identity_token"`
	AuthorizationCode string `json:"authorization_code"`
}

// AppleClaims are the verified claims we care about from Apple's identity token.
type AppleClaims struct {
	Email string
	Sub   string
}

type appleIDTokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type appleJWKSet struct {
	Keys []appleJWK `json:"keys"`
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type appleKeyCache struct {
	fetchedAt time.Time
	keysByKid map[string]*rsa.PublicKey
	fetchFn   func(ctx context.Context) (appleJWKSet, error)
	mu        sync.RWMutex
}

func newAppleKeyCache() *appleKeyCache {
	return &appleKeyCache{
		keysByKid: make(map[string]*rsa.PublicKey),
		fetchFn:   fetchAppleJWKS,
	}
}

func fetchAppleJWKS(ctx context.Context) (appleJWKSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, http.NoBody)
	if err != nil {
		return appleJWKSet{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return appleJWKSet{}, fmt.Errorf("fetching apple jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		if readErr != nil {
			return appleJWKSet{}, fmt.Errorf("apple jwks unexpected status %d", resp.StatusCode)
		}
		return appleJWKSet{}, fmt.Errorf("apple jwks unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var set appleJWKSet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return appleJWKSet{}, fmt.Errorf("decoding apple jwks: %w", err)
	}
	return set, nil
}

func (c *appleKeyCache) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if key := c.cachedKey(kid); key != nil {
		return key, nil
	}
	if err := c.refresh(ctx, true); err != nil {
		return nil, err
	}
	if key := c.cachedKey(kid); key != nil {
		return key, nil
	}
	return nil, fmt.Errorf("%w: unknown kid %s", errAppleInvalidToken, kid)
}

func (c *appleKeyCache) cachedKey(kid string) *rsa.PublicKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.fetchedAt.IsZero() || time.Since(c.fetchedAt) > appleJWKSCacheTTL {
		return nil
	}
	return c.keysByKid[kid]
}

func (c *appleKeyCache) refresh(ctx context.Context, force bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Another goroutine may have refreshed while we waited for the lock.
	if !force && !c.fetchedAt.IsZero() && time.Since(c.fetchedAt) <= appleJWKSCacheTTL && len(c.keysByKid) > 0 {
		return nil
	}

	set, err := c.fetchFn(ctx)
	if err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" {
			continue
		}
		pub, err := jwkToRSAPublicKey(&jwk)
		if err != nil {
			continue
		}
		keys[jwk.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("apple jwks contained no usable RSA keys")
	}

	c.keysByKid = keys
	c.fetchedAt = time.Now()
	return nil
}

func jwkToRSAPublicKey(jwk *appleJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decoding jwk n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decoding jwk e: %w", err)
	}

	var eInt int
	for _, b := range eBytes {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, errors.New("invalid jwk exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}, nil
}

// validateAppleIdentityToken verifies Apple's identity token signature and claims.
func validateAppleIdentityToken(
	ctx context.Context,
	identityToken string,
	audiences map[string]bool,
	keys *appleKeyCache,
) (*AppleClaims, error) {
	if identityToken == "" {
		return nil, errAppleInvalidToken
	}
	if len(audiences) == 0 {
		return nil, errors.New("apple client ids not configured")
	}
	if keys == nil {
		return nil, errors.New("apple key cache not configured")
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	token, err := parser.ParseWithClaims(identityToken, &appleIDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, fmt.Errorf("%w: missing kid", errAppleInvalidToken)
		}
		return keys.getKey(ctx, kid)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errAppleInvalidToken, err)
	}

	claims, ok := token.Claims.(*appleIDTokenClaims)
	if !ok || !token.Valid {
		return nil, errAppleInvalidToken
	}

	if claims.Issuer != appleIssuer {
		return nil, fmt.Errorf("%w: unexpected issuer", errAppleInvalidToken)
	}
	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: missing sub", errAppleInvalidToken)
	}
	if !audienceAllowed(claims.Audience, audiences) {
		return nil, errAppleWrongAudience
	}

	return &AppleClaims{
		Email: claims.Email,
		Sub:   claims.Subject,
	}, nil
}

func audienceAllowed(tokenAud jwt.ClaimStrings, allowed map[string]bool) bool {
	for _, aud := range tokenAud {
		if allowed[aud] {
			return true
		}
	}
	return false
}
