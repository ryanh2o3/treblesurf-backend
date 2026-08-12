package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/repository/mock"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestValidateAppleIdentityToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	const kid = "test-kid"
	cache := &appleKeyCache{
		keysByKid: map[string]*rsa.PublicKey{kid: &privateKey.PublicKey},
		fetchedAt: time.Now(),
		fetchFn: func(context.Context) (appleJWKSet, error) {
			return appleJWKSet{Keys: []appleJWK{rsaPublicKeyToJWK(kid, &privateKey.PublicKey)}}, nil
		},
	}

	audiences := map[string]bool{"treble.TrebleSurf": true}

	t.Run("valid token", func(t *testing.T) {
		token := signAppleTestToken(t, privateKey, kid, appleIDTokenClaims{
			Email: "relay@privaterelay.appleid.apple.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    appleIssuer,
				Subject:   "apple-sub-123",
				Audience:  jwt.ClaimStrings{"treble.TrebleSurf"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

		claims, err := validateAppleIdentityToken(context.Background(), token, audiences, cache)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Sub != "apple-sub-123" {
			t.Fatalf("expected sub apple-sub-123, got %s", claims.Sub)
		}
		if claims.Email != "relay@privaterelay.appleid.apple.com" {
			t.Fatalf("unexpected email: %s", claims.Email)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token := signAppleTestToken(t, privateKey, kid, appleIDTokenClaims{
			Email: "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    appleIssuer,
				Subject:   "apple-sub-123",
				Audience:  jwt.ClaimStrings{"treble.TrebleSurf"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		})

		_, err := validateAppleIdentityToken(context.Background(), token, audiences, cache)
		if err == nil {
			t.Fatal("expected error for expired token")
		}
		if !errors.Is(err, errAppleInvalidToken) {
			t.Fatalf("expected errAppleInvalidToken, got %v", err)
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		token := signAppleTestToken(t, privateKey, kid, appleIDTokenClaims{
			Email: "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    appleIssuer,
				Subject:   "apple-sub-123",
				Audience:  jwt.ClaimStrings{"com.other.app"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

		_, err := validateAppleIdentityToken(context.Background(), token, audiences, cache)
		if !errors.Is(err, errAppleWrongAudience) {
			t.Fatalf("expected errAppleWrongAudience, got %v", err)
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token := signAppleTestToken(t, privateKey, kid, appleIDTokenClaims{
			Email: "user@example.com",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "https://evil.example",
				Subject:   "apple-sub-123",
				Audience:  jwt.ClaimStrings{"treble.TrebleSurf"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

		_, err := validateAppleIdentityToken(context.Background(), token, audiences, cache)
		if err == nil {
			t.Fatal("expected error for wrong issuer")
		}
	})
}

func TestAppleAuthHandler_NewAndReturningUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	const kid = "test-kid"
	const appleSub = "001234.abcdef"
	const email = "relay@privaterelay.appleid.apple.com"

	users := map[string]*model.User{}
	userRepo := &mock.UserRepo{
		GetByAppleIDFn: func(_ context.Context, appleID string) (*model.User, error) {
			for _, u := range users {
				if u.AppleID == appleID {
					copied := *u
					return &copied, nil
				}
			}
			return nil, repository.ErrNotFound
		},
		GetByEmailFn: func(_ context.Context, gotEmail string) (*model.User, error) {
			u, ok := users[gotEmail]
			if !ok {
				return nil, repository.ErrNotFound
			}
			copied := *u
			return &copied, nil
		},
		CreateFn: func(_ context.Context, user *model.User) error {
			if user.UUID == "" {
				user.UUID = "test-uuid"
			}
			if user.CreatedAt == "" {
				user.CreatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			if user.LastLogin == "" {
				user.LastLogin = user.CreatedAt
			}
			if user.Theme == "" {
				user.Theme = "system"
			}
			copied := *user
			users[user.Email] = &copied
			return nil
		},
		UpdateFn: func(_ context.Context, user *model.User) error {
			copied := *user
			users[user.Email] = &copied
			return nil
		},
		UpdateLastLoginFn: func(_ context.Context, gotEmail string, at time.Time) error {
			u, ok := users[gotEmail]
			if !ok {
				return repository.ErrNotFound
			}
			u.LastLogin = at.UTC().Format(time.RFC3339)
			return nil
		},
	}

	svc := &Service{
		userRepo:       userRepo,
		appleClientIDs: buildClientIDMap([]string{"treble.TrebleSurf"}),
		cookieSecure:   false,
		isDevelopment:  true,
		appleKeyCache: &appleKeyCache{
			keysByKid: map[string]*rsa.PublicKey{kid: &privateKey.PublicKey},
			fetchedAt: time.Now(),
			fetchFn: func(context.Context) (appleJWKSet, error) {
				return appleJWKSet{Keys: []appleJWK{rsaPublicKeyToJWK(kid, &privateKey.PublicKey)}}, nil
			},
		},
	}

	makeToken := func(includeEmail bool) string {
		claims := appleIDTokenClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    appleIssuer,
				Subject:   appleSub,
				Audience:  jwt.ClaimStrings{"treble.TrebleSurf"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		if includeEmail {
			claims.Email = email
		}
		return signAppleTestToken(t, privateKey, kid, claims)
	}

	t.Run("new user", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/apple", strings.NewReader(
			`{"identity_token":"`+makeToken(true)+`"}`,
		))
		c.Request.Header.Set("Content-Type", "application/json")

		svc.AppleAuthHandler(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		created, ok := users[email]
		if !ok {
			t.Fatal("expected user to be created")
		}
		if created.AppleID != appleSub {
			t.Fatalf("expected apple_id %s, got %s", appleSub, created.AppleID)
		}
		if created.Picture != "" {
			t.Fatalf("expected empty picture, got %q", created.Picture)
		}
		if w.Header().Get("X-CSRF-Token") == "" {
			t.Fatal("expected X-CSRF-Token header")
		}
	})

	t.Run("returning user without email claim", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/apple", strings.NewReader(
			`{"identity_token":"`+makeToken(false)+`"}`,
		))
		c.Request.Header.Set("Content-Type", "application/json")

		svc.AppleAuthHandler(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}
		if len(users) != 1 {
			t.Fatalf("expected single user, got %d", len(users))
		}
	})

	t.Run("bad token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/apple", strings.NewReader(
			`{"identity_token":"not-a-jwt"}`,
		))
		c.Request.Header.Set("Content-Type", "application/json")

		svc.AppleAuthHandler(c)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("wrong audience", func(t *testing.T) {
		token := signAppleTestToken(t, privateKey, kid, appleIDTokenClaims{
			Email: email,
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    appleIssuer,
				Subject:   appleSub,
				Audience:  jwt.ClaimStrings{"com.other.app"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/auth/apple", strings.NewReader(
			`{"identity_token":"`+token+`"}`,
		))
		c.Request.Header.Set("Content-Type", "application/json")

		svc.AppleAuthHandler(c)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestConfigAppleClientIDsDefault(t *testing.T) {
	t.Setenv("GO_ENV", "development")
	t.Setenv("JWT_SECRET", "test")
	t.Setenv("APPLE_CLIENT_IDS", "")
	t.Setenv("APPLE_BUNDLE_ID", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(cfg.Auth.AppleClientIDs) != 1 || cfg.Auth.AppleClientIDs[0] != "treble.TrebleSurf" {
		t.Fatalf("expected default treble.TrebleSurf, got %#v", cfg.Auth.AppleClientIDs)
	}
}

func signAppleTestToken(t *testing.T, key *rsa.PrivateKey, kid string, claims appleIDTokenClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func rsaPublicKeyToJWK(kid string, pub *rsa.PublicKey) appleJWK {
	return appleJWK{
		Kty: "RSA",
		Kid: kid,
		Use: "sig",
		Alg: "RS256",
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}
