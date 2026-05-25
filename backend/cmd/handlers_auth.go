package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// AuthenticationURIHandler redirects users to google authorization page with making Cookie.
func (a *App) AuthenticationURIHandler(w http.ResponseWriter, r *http.Request) {
	// Add temporary parameters and construct an authentication request to Google.
	state := rand.Text()
	nonce := rand.Text()

	// Set state and nonce cookies to check validity of the access token and OpenID later.
	http.SetCookie(w, a.MakeSignedCookie("state", state, 300))
	http.SetCookie(w, a.MakeSignedCookie("nonce", nonce, 300))

	oauthURL := a.OAuth2Conf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("redirect_uri", a.OAuth2Conf.RedirectURL))
	http.Redirect(w, r, oauthURL, http.StatusFound)
}

// GetAccessTokenHandler takes the redirect URI, then gets the access token and openID.
func (a *App) GetAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stateValue, err := a.GetAndVerifyCookie(r, "state")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			log.Printf("retrieve cookie: %v", err)
			http.Error(w, "invalid session", http.StatusUnauthorized)
		}
		log.Printf("verify cookie: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("state") != stateValue {
		log.Println("the state is not identical")
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	// Makes sure this state is not reused by attackers.
	http.SetCookie(w, a.MakeDeleteCookie("state"))

	tok, err := a.OAuth2Conf.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("Exchange Authorization Code: %s", err)
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	rawOpenIdToken, ok := tok.Extra("id_token").(string)
	if !ok {
		http.Error(w, "missing id_token", http.StatusUnauthorized)
		return
	}

	openIdToken, err := a.OIDCVerifier.Verify(ctx, rawOpenIdToken)
	if err != nil {
		http.Error(w, "invalid OpenID Token", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Nonce string `json:"nonce"`
		Sub   string `json:"sub"`
	}

	if err := openIdToken.Claims(&claims); err != nil {
		log.Printf("gettin nonce: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	nonceValue, err := a.GetAndVerifyCookie(r, "nonce")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			log.Printf("retrieve cookie: %v", err)
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}
		log.Printf("verify cookie: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if nonceValue != claims.Nonce {
		log.Printf("the nonce is not valid. Cookie: %s, OpenID: %s", nonceValue, claims.Nonce)
		http.Error(w, "invalid ID token", http.StatusUnauthorized)
		return
	}

	// Make sure the attackers send the app OpneID token with the same nonce and it passes.
	http.SetCookie(w, a.MakeDeleteCookie("nonce"))

	sub := claims.Sub

	// Register the user if the user is new. Then, get the user's userId.
	var userId uuid.UUID

	// If the sub already exists, it technically updates the existing sub with the new one. However, since they are the same, it does nothing and just returns the user id.
	err = a.Pool.QueryRow(ctx, "INSERT INTO users (google_sub) VALUES ($1) ON CONFLICT (google_sub) DO UPDATE SET google_sub = EXCLUDED.google_sub RETURNING id", sub).Scan(&userId)
	if err != nil {
		log.Printf("QueryRow failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Avoid attacker's changing JWT.
	signedAccessToken, err := a.MakeSignedAccessToken(userId)
	if err != nil {
		log.Printf("making access token: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Signed Cookie prevents attacker's changing Cookie.
	http.SetCookie(w, a.MakeSignedCookie("access_token", signedAccessToken, a.AccessTokenDuration*60))

	refreshToken, refreshTokenDuration, err := a.InitRefreshToken(userId, ctx)
	if err != nil {
		log.Println(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, a.MakeSignedRefreshTokenCookie(refreshToken, refreshTokenDuration*3600))

	log.Printf("my access token: %s \n my refresh token: %s", signedAccessToken, refreshToken)

	http.Redirect(w, r, a.AppPublicURL+"/timeline", http.StatusFound)
}

func (a *App) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	// get and verify the refresh token
	refreshToken, err := a.GetAndVerifyCookie(r, "refresh_token")
	if err != nil {
		log.Printf("Getting and Verifying refresh token cookie: %s", err)
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	// verify the refresh token
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	subStr, err := a.Rdb.Get(ctx, refreshToken).Result()
	if err != nil {
		log.Printf("Getting userId from Redis: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	subUuid, err := uuid.Parse(subStr)
	if err != nil {
		log.Printf("Converting type of userId into UUID: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	newSignedAccessToken, err := a.MakeSignedAccessToken(subUuid)
	if err != nil {
		log.Printf("Making new access token: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, a.MakeSignedCookie("access_token", newSignedAccessToken, a.AccessTokenDuration*60))
}

func (a *App) LogOutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, a.MakeDeleteCookie("access_token"))

	refreshToken, err := a.GetAndVerifyCookie(r, "refresh_token")
	if err != nil {
		log.Printf("Getting refresh token: %s", err)
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = a.Rdb.Del(ctx, refreshToken).Err()
	if err != nil {
		log.Printf("Deleting refresh token from Redis: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, a.MakeDeleteRefreshTokenCookie())

	w.WriteHeader(http.StatusNoContent)
}

// RegisterHandler handles POST /api/auth/register.
// It accepts a JSON body of {email, password}, creates an unverified user account,
// and triggers a verification email. No session cookie is issued — the client must
// complete email verification before logging in.
//
// 201 Created      — account created; verification email dispatched (or best-effort).
// 400 Bad Request  — malformed JSON, invalid email format, or password rejected.
// 409 Conflict     — email address is already registered.
// 500 Internal     — unexpected DB / Redis failure.
func (a *App) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userId, verificationURL, err := a.RegisterPasswordUser(ctx, req.Email, req.Password)
	if err != nil {
		// Surface duplicate-email as 409 without logging — it is an expected user error.
		if err.Error() == "email already registered" {
			log.Println(err.Error())
			http.Error(w, "email already registered", http.StatusConflict)
			return
		}
		log.Printf("RegisterPasswordUser: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Email delivery is best-effort: the user is already persisted, so a send
	// failure should not roll back the registration. Log and move on; the user
	// can request a resend later.
	if _, err := a.SendVerificationEmail(ctx, req.Email, verificationURL); err != nil {
		log.Printf("SendVerificationEmail userId=%s: %v", userId, err)
	}

	w.WriteHeader(http.StatusCreated)
}
