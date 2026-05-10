package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// These structs are for HTTP response and different from the ones for database.
// They trim some fields such as userId.

type MakePostRequest struct {
	Body string `json:"body"`
}

type PostResponse struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	LikedByMe bool      `json:"liked_by_me"`
	LikeCount int       `json:"like_count"`
}

func postToResponse(p Post) PostResponse {
	return PostResponse{
		ID:        p.PostId,
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

	body, err := json.Marshal(out)
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
	body, err := json.Marshal(resp)
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
	out := make([]PostResponse, len(posts))
	for i := range posts {
		out[i] = postToResponse(posts[i])
	}
	body, err := json.Marshal(out)
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
	body, err := json.Marshal(out)
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

func (a *App) FollowUserHandler(w http.ResponseWriter, r *http.Request) {
	result, ok := AuthFromRequest(r)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	strPostId := r.PathValue("id")
	postID, err := uuid.Parse(strPostId)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = AddFollow(result.Sub, postID, a.Pool, ctx)
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

	strPostId := r.PathValue("id")
	postID, err := uuid.Parse(strPostId)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = DeleteFollow(result.Sub, postID, a.Pool, ctx)
	if err != nil {
		log.Print(err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
