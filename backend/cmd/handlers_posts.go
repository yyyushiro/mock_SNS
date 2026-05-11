package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// PostResponse mirrors the fields exposed from Post for JSON.

type MakePostRequest struct {
	Body string `json:"body"`
}

type PostResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	LikedByMe bool      `json:"liked_by_me"`
	LikeCount int       `json:"like_count"`
}

func postToResponse(p Post) PostResponse {
	return PostResponse{
		ID:        p.Id,
		UserID:    p.UserId,
		Username:  p.Username,
		Body:      p.Body,
		CreatedAt: p.CreatedAt,
		LikedByMe: p.LikedByMe,
		LikeCount: p.LikeCount,
	}
}

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

	out := make([]PostResponse, len(posts))
	for i := range posts {
		out[i] = postToResponse(posts[i])
	}

	err = WriteJsonResponseBody(w, out, http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing my posts response: %v", err)
		return
	}
}

func (a *App) MakePostHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var req MakePostRequest
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

	resp := postToResponse(*post)
	err = WriteJsonResponseBody(w, resp, http.StatusCreated)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing create post response: %v", err)
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

	err = DeletePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

	err = LikePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Printf("Liked a post: %s", err)
		http.Error(w, "faied to like a post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
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

	err = UndoLikePost(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Printf("Undid a like: %s", err)
		http.Error(w, "faied to undo a like", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	out := make([]PostResponse, len(posts))
	for i := range posts {
		out[i] = postToResponse(posts[i])
	}
	err = WriteJsonResponseBody(w, out, http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing public posts response: %v", err)
		return
	}
}

func (a *App) GetFollowingPostsHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	userId := result.Sub
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	posts, err := GetFollowingPosts(userId, a.Pool, ctx)
	if err != nil {
		log.Printf("Getting current user's public posts: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	out := make([]PostResponse, len(posts))
	for i := range posts {
		out[i] = postToResponse(posts[i])
	}

	err = WriteJsonResponseBody(w, out, http.StatusOK)
	if err != nil {
		if errors.Is(err, ErrJsonEncode) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		log.Printf("Writing following posts response: %v", err)
		return
	}
}
