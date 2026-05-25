package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
func (a *App) MakeSignedAccessToken(sub uuid.UUID) (string, error) {
	exp := time.Now().Add(time.Duration(a.AccessTokenDuration) * time.Minute)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": sub,
			"exp": exp.Unix(),
			"iat": time.Now().Unix(),
		})

	signedAccessToken, err := accessToken.SignedString(a.JWTSecret)
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
func (a *App) verifyAccessTokenJWT(tokenString string) (*accessTokenClaims, error) {
	var claims accessTokenClaims
	tok, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected alg: %v", t.Header["alg"])
		}
		return a.JWTSecret, nil
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

// InitRefreshToken creates a refresh token and store it into Redis.
func (a *App) InitRefreshToken(userId uuid.UUID, ctx context.Context) (string, int, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return "", 0, err
	}

	err = a.Rdb.Set(ctx, refreshToken, userId.String(), time.Duration(a.RefreshTokenDuration)*time.Hour).Err()
	if err != nil {
		return "", 0, fmt.Errorf("Setting refresh token: %w", err)
	}

	return refreshToken, a.RefreshTokenDuration, nil
}

// RequireAuth verifies the user's session via access or refresh token.
func (a *App) RequireAuth(r *http.Request) (*AuthResult, error) {
	accessToken, err := a.GetAndVerifyCookie(r, "access_token")
	// Access token is invalid, not found, or has some issue.
	if err != nil {
		return nil, fmt.Errorf("Verifying access token cookie: %w", err)
	}
	claims, err := a.verifyAccessTokenJWT(accessToken)
	if err != nil {
		return nil, fmt.Errorf("Verifying access token JWT: %w", err)
	}

	subUuid, err := uuid.Parse(claims.Sub)
	if err != nil {
		return nil, fmt.Errorf("parsing string into uuid: %w", err)
	}

	return &AuthResult{Sub: subUuid, NewAccessTokenCookie: nil}, nil
}
