package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/resend/resend-go/v3"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"
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

// generateVerificationToken generates a random 32-byte URL-safe token for email verification links.
func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func normalizeEmail(raw string) string {
	return strings.ToLower(norm.NFKC.String(strings.TrimSpace(raw)))
}

func validateEmail(normalized string) bool {
	_, err := mail.ParseAddress(normalized)
	return err == nil
}

func validatePassword(plain string) bool {
	return true
}

func hashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), 14)
	return string(bytes), err
}

func verifyPassword(hashed, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}

const verificationTokenTTL = 24 * time.Hour

// ErrInvalidVerificationToken is returned when the opaque token is absent
// from Redis — either expired, never existed, or already consumed.
var ErrInvalidVerificationToken = errors.New("invalid or expired verification token")

// ErrInvalidCredentials is returned when email or password does not match.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrEmailNotVerified is returned when credentials are correct but the email
// address has not yet been verified.
var ErrEmailNotVerified = errors.New("email not verified")

func verificationRedisKey(token string) string {
	return "email_verify:" + token
}

func (a *App) storeVerificationToken(ctx context.Context, userId uuid.UUID, token string) error {
	err := a.Rdb.Set(ctx, verificationRedisKey(token), userId.String(), verificationTokenTTL).Err()
	if err != nil {
		return fmt.Errorf("storing verification token: %w", err)
	}
	return nil
}

func (a *App) consumeVerificationToken(ctx context.Context, token string) (uuid.UUID, error) {
	subStr, err := a.Rdb.GetDel(ctx, verificationRedisKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.UUID{}, ErrInvalidVerificationToken
		}
		return uuid.UUID{}, fmt.Errorf("consuming verification token: %w", err)
	}

	userId, err := uuid.Parse(subStr)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parsing user id from verification token: %w", err)
	}
	return userId, nil
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

// RegisterPasswordUser is the service-layer entry point for email/password sign-up.
//
// On success it returns the new user's ID and a ready-to-use verification URL
// (AppPublicURL + /verify-email?token=…) that the caller can forward to the
// email-sending layer without needing to know the token format.
//
// Errors from InsertPasswordUser are returned unwrapped so the HTTP handler can
// inspect the "email already registered" sentinel directly.
func (a *App) RegisterPasswordUser(ctx context.Context, email, plainPassword string) (uuid.UUID, string, error) {
	normalized := normalizeEmail(email)
	if !validateEmail(normalized) {
		return uuid.UUID{}, "", fmt.Errorf("invalid email")
	}
	if !validatePassword(plainPassword) {
		return uuid.UUID{}, "", fmt.Errorf("invalid password")
	}
	hashed, err := hashPassword(plainPassword)
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("hashing password: %w", err)
	}
	userId, err := InsertPasswordUser(ctx, a.Pool, normalized, hashed)
	if err != nil {
		return uuid.UUID{}, "", err
	}
	token, err := generateVerificationToken()
	if err != nil {
		return uuid.UUID{}, "", fmt.Errorf("generating verification token: %w", err)
	}
	if err := a.storeVerificationToken(ctx, userId, token); err != nil {
		return uuid.UUID{}, "", err
	}
	verificationURL := a.AppPublicURL + "/api/auth/verify-email?token=" + token
	return userId, verificationURL, nil
}

// LoginPasswordUser validates an email/password pair and returns the user's ID.
//
// ErrInvalidCredentials is returned when no account exists for the email or
// the password does not match — deliberately indistinguishable to callers.
// ErrEmailNotVerified is returned when credentials are correct but the account
// has not yet completed email verification.
func (a *App) LoginPasswordUser(ctx context.Context, email, plainPassword string) (uuid.UUID, error) {
	normalized := normalizeEmail(email)
	id, hashedPassword, emailVerified, err := GetPasswordUserByEmail(ctx, a.Pool, normalized)
	if err != nil || !verifyPassword(hashedPassword, plainPassword) {
		// Treat "no such user" identically to a wrong password.
		return uuid.UUID{}, ErrInvalidCredentials
	}
	if !emailVerified {
		return uuid.UUID{}, ErrEmailNotVerified
	}
	return id, nil
}

// SendVerificationEmail delivers a verification link to the registrant.
// Implement this using the Resend API (or any transactional email provider).
// toEmail is the normalized address; verificationURL is the one-time link
// produced by RegisterPasswordUser.
func (a *App) SendVerificationEmail(ctx context.Context, toEmail, verificationURL string) (*resend.SendEmailResponse, error) {
	client := resend.NewClient(a.ResendApiKey)

	params := &resend.SendEmailRequest{
		From:    a.EmailFrom,
		To:      []string{toEmail},
		Html:    "<strong>Click the link below to verify your email.</strong><a href=" + verificationURL + ">click here</a>",
		Subject: "Email Verification",
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return nil, err
	}

	return sent, nil
}
