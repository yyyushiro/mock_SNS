package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
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

// makeSignedAccessToken generates JWT with the given sub.
func makeSignedAccessToken(sub uuid.UUID) (string, error) {
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
	if err != nil {
		// Check if the refresh token is present.
		refreshToken, err := GetAndVerifyCookie(r, "refresh_token")
		if err != nil {
			return nil, fmt.Errorf("Verifying refresh token Cookie: %w", err)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		subStr, err := rdb.Get(ctx, refreshToken).Result()
		if err != nil {
			return nil, fmt.Errorf("Getting sub from redis: %w", err)
		}
		log.Print(subStr)
		subUuid, err := uuid.Parse(subStr)
		if err != nil {
			return nil, fmt.Errorf("Parsing sub into uuid: %w", err)
		}

		// found sub, so issue new access token. Access token is made from a sub.
		newSignedAccessToken, err := makeSignedAccessToken(subUuid)
		if err != nil {
			return nil, fmt.Errorf("Making new access token: %w", err)
		}

		// return the new access token cookie.
		return &AuthResult{
			Sub:                  subUuid,
			NewAccessTokenCookie: MakeSignedCookie("access_token", newSignedAccessToken, 900),
		}, nil
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
