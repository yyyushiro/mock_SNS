package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (a *App) GetMyPostsHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	posts, err := GetMyPosts(result.Sub, a.Pool, ctx)
	if err != nil {
		log.Printf("Getting current user's posts: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(posts)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

type makePostRequest struct {
	Body string `json:"body"`
}

func (a *App) MakePostHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var req makePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("decode body: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Body == "" {
		http.Error(w, "body required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	post, err := AddPost(result.Sub, req.Body, a.Pool, ctx)
	if err != nil {
		log.Printf("creating post: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(post)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
		return
	}
	log.Printf("made post successfully: %s", req.Body)
}

func (a *App) DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	idStr := r.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	post, err := DeletePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Printf("deleting post: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(post)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (a *App) LikePostHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	like, err := LikePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Printf("Liked a post: %s", err)
		http.Error(w, "faied to like a post", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(like)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (a *App) UndoLikePostHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	like, err := UndoLikePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Printf("Undid a like: %s", err)
		http.Error(w, "faied to undo a like", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(like)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (a *App) GetPublicPostsHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userId := result.Sub
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	posts, err := GetPublicPosts(userId, a.Pool, ctx)
	if err != nil {
		log.Printf("Getting current user's public posts: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	body, err := json.Marshal(posts)
	if err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}
