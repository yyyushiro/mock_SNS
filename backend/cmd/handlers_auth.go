package main

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	http.SetCookie(w, MakeSignedCookie("state", state, 300))
	http.SetCookie(w, MakeSignedCookie("nonce", nonce, 300))

	oauthURL := a.OAuth2Conf.AuthCodeURL(state, oauth2.SetAuthURLParam("nonce", nonce), oauth2.SetAuthURLParam("redirect_uri", os.Getenv("REDIRECT_URI")))
	http.Redirect(w, r, oauthURL, http.StatusFound)
}

// GetAccessTokenHandler takes the redirect URI, then gets the access token and openID.
func (a *App) GetAccessTokenHandler(w http.ResponseWriter, r *http.Request) {
	stateValue, err := GetAndVerifyCookie(r, "state")
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
	stateDeleteCookie := MakeDeleteCookie("state")
	http.SetCookie(w, stateDeleteCookie)

	tok, err := a.OAuth2Conf.Exchange(context.TODO(), r.URL.Query().Get("code"))
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

	openIdToken, err := a.OIDCVerifier.Verify(r.Context(), rawOpenIdToken)
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

	nonceValue, err := GetAndVerifyCookie(r, "nonce")
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
	nonceDeleteCookie := MakeDeleteCookie("nonce")
	http.SetCookie(w, nonceDeleteCookie)

	sub := claims.Sub

	// Register the user if the user is new. Then, get the user's userId.
	var userId uuid.UUID
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = a.Pool.QueryRow(ctx, "INSERT INTO users (google_sub) VALUES ($1) ON CONFLICT (google_sub) DO UPDATE SET google_sub = EXCLUDED.google_sub RETURNING id", sub).Scan(&userId)
	if err != nil {
		log.Printf("QueryRow failed: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Avoid attacker's changing JWT.
	signedAccessToken, err := makeSignedAccessToken(userId)
	if err != nil {
		log.Printf("making access token: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// This cookie is needed for authorization of user in each request.
	accessTokenCookie := MakeSignedCookie("access_token", signedAccessToken, 900)
	http.SetCookie(w, accessTokenCookie)

	// Store refresh token into Redis and Cookie for getting new access token when expired.
	refreshToken, err := generateRefreshToken()
	if err != nil {
		log.Printf("generating refresh token: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	refreshTokenDuration, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_DURATION"))
	if err != nil {
		log.Printf("parsing refresh token duration: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	ctx, cancel = context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = a.Rdb.Set(ctx, refreshToken, sub, time.Duration(refreshTokenDuration)*time.Hour).Err()
	if err != nil {
		log.Printf("storing refresh token into Redis: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	refreshTokenCookie := MakeSignedCookie("refresh_token", refreshToken, 604800)
	http.SetCookie(w, refreshTokenCookie)

	log.Printf("my access token: %s \n my refresh token: %s", signedAccessToken, refreshToken)

	base := strings.TrimSpace(os.Getenv("APP_PUBLIC_URL"))
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	http.Redirect(w, r, base+"/timeline", http.StatusFound)
}

func (a *App) LogOutHandler(w http.ResponseWriter, r *http.Request) {
	_, err := RequireAuth(r, a.Rdb)
	if err != nil {
		log.Printf("Authorization: %s", err)
		http.Error(w, "invalid session", http.StatusUnauthorized)
		return
	}
	// delete access token and refresh token.
	accessTokenDeleteCookie := MakeDeleteCookie("access_token")
	http.SetCookie(w, accessTokenDeleteCookie)

	refreshToken, err := GetAndVerifyCookie(r, "refresh_token")
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

	refreshTokenDeleteCookie := MakeDeleteCookie("refresh_token")
	http.SetCookie(w, refreshTokenDeleteCookie)

	w.WriteHeader(http.StatusNoContent)
}
