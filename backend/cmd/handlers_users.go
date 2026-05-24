package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type MyInfoResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type OtherUserInfoResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func userToMyInfoResponse(u *User) MyInfoResponse {
	name := ""
	if u.Username.Valid {
		name = u.Username.String
	}
	return MyInfoResponse{ID: u.Id, Username: name, CreatedAt: u.CreatedAt}
}

func userToOtherUserInfoResponse(u *User) OtherUserInfoResponse {
	name := ""
	if u.Username.Valid {
		name = u.Username.String
	}
	return OtherUserInfoResponse{ID: u.Id, Username: name, CreatedAt: u.CreatedAt}
}

func (a *App) FollowUserHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	strFolloweeId := r.PathValue("id")
	followeeId, err := uuid.Parse(strFolloweeId)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = AddFollow(result.Sub, followeeId, a.Pool, ctx)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) UnfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	strFolloweeId := r.PathValue("id")
	followeeId, err := uuid.Parse(strFolloweeId)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = DeleteFollow(result.Sub, followeeId, a.Pool, ctx)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *App) GetMyInfoHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	user, err := GetUserInfo(result.Sub, a.Pool, ctx)
	if err != nil {
		log.Printf("Getting My Info: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = WriteJsonResponseBody(w, userToMyInfoResponse(user), http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing GetMyInfo response: %v", err)
		return
	}
}

func (a *App) GetOtherUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	strOtherID := r.PathValue("id")
	otherID, err := uuid.Parse(strOtherID)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	user, err := GetUserInfo(otherID, a.Pool, ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("Getting other user info: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = WriteJsonResponseBody(w, userToOtherUserInfoResponse(user), http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing GetOtherUserInfo response: %v", err)
		return
	}
}

type PatchMyUsernameRequest struct {
	Username string `json:"username"`
}

func (a *App) PatchMyUsernameHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var req PatchMyUsernameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("PatchMyUsername decode: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	trimmed := strings.TrimSpace(req.Username)
	if trimmed == "" || len(trimmed) > 255 {
		http.Error(w, "invalid username", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := UpdateMyUsername(result.Sub, trimmed, a.Pool, ctx)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "username taken", http.StatusConflict)
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println(err)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("PatchMyUsername: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := GetUserInfo(result.Sub, a.Pool, ctx)
	if err != nil {
		log.Printf("PatchMyUsername reload user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	err = WriteJsonResponseBody(w, userToMyInfoResponse(user), http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing PatchMyUsername response: %v", err)
		return
	}
}
