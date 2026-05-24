package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// generateRefreshToken generates a random 32-byte base64 token for a refresh token.
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// MakeSignedAccessToken generates JWT with the given sub.
func MakeSignedAccessToken(sub uuid.UUID) (string, error) {
	accessTokenDuration, err := strconv.Atoi(os.Getenv("ACCESS_TOKEN_DURATION"))
	if err != nil {
		return "", err
	}

	exp := time.Now().Add(time.Duration(accessTokenDuration) * time.Minute)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": sub,
			"exp": exp.Unix(),
			"iat": time.Now().Unix(),
		})

	signedAccessToken, err := accessToken.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}
	return signedAccessToken, nil
}

type accessTokenClaims struct {
	Sub string `json:"sub"`
	jwt.RegisteredClaims
}

// verifyAccessTokenJWT verifies the given JWT and decode it.
func verifyAccessTokenJWT(tokenString string, secret []byte) (*accessTokenClaims, error) {
	var claims accessTokenClaims
	tok, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected alg: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return &claims, nil
}

// You can expand this struct if you want to get more information from authorization later.
type AuthResult struct {
	Sub                  uuid.UUID
	NewAccessTokenCookie *http.Cookie
}

// Empty struct is often used as a key because it avoids conflicts, and it is comparable.
type authContextKey struct{}

// ContextWithAuth attaches an AuthResult to r for handlers wrapped by WithAuth.
// Handlers need the AuthResult such as for manipulating the database.
func ContextWithAuth(r *http.Request, ar *AuthResult) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authContextKey{}, ar))
}

// AuthFromRequest returns the AuthResult set by WithAuth, or false if missing.
func AuthFromRequest(r *http.Request) (*AuthResult, bool) {
	ar, ok := r.Context().Value(authContextKey{}).(*AuthResult)
	if !ok || ar == nil {
		return nil, false
	}
	return ar, true
}

// RequireAuth verifies the user's session via access or refresh token.
func RequireAuth(r *http.Request, rdb *redis.Client) (*AuthResult, error) {
	accessToken, err := GetAndVerifyCookie(r, "access_token")
	// Access token is invalid, not found, or has some issue.
	if err != nil {
		return nil, fmt.Errorf("Verifying access token cookie: %w", err)
	}
	accessTokenClaims, err := verifyAccessTokenJWT(accessToken, []byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return nil, fmt.Errorf("Verifying access token JWT: %w", err)
	}

	subUuid, err := uuid.Parse(accessTokenClaims.Sub)
	if err != nil {
		return nil, fmt.Errorf("parsing string into uuid: %w", err)
	}

	return &AuthResult{Sub: subUuid, NewAccessTokenCookie: nil}, nil
}
