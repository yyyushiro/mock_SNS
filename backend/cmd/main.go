package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel() makes sure the memory of ctx is released before main() ends.
	defer cancel()

	pool, err := connectDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	rdb, err := newRedisClient()
	if err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	oauth2Config, oidcVerifier, err := setUpOAuth2AndOIDC(ctx)
	if err != nil {
		log.Fatalf("failed to configure OAuth2 and OIDC: %v", err)
	}

	app := &App{
		Pool:         pool,
		Rdb:          rdb,
		OAuth2Conf:   oauth2Config,
		OIDCVerifier: oidcVerifier,
	}

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("GET /api/auth/google/start", app.AuthenticationURIHandler)
	mux.HandleFunc("GET /api/auth/callback/google", app.GetAccessTokenHandler)
	mux.HandleFunc("POST /api/auth/refresh", app.RefreshTokenHandler)
	mux.HandleFunc("POST /api/auth/logout", app.WithAuth(app.LogOutHandler))

	// Posts
	mux.HandleFunc("POST /api/posts", app.WithAuth(app.MakePostHandler))
	mux.HandleFunc("DELETE /api/posts/{id}", app.WithAuth(app.DeletePostHandler))
	mux.HandleFunc("POST /api/posts/{id}/likes", app.WithAuth(app.LikePostHandler))
	mux.HandleFunc("DELETE /api/posts/{id}/likes", app.WithAuth(app.UndoLikePostHandler))

	// Users — me
	mux.HandleFunc("GET /api/user/me", app.WithAuth(app.GetMyInfoHandler))
	mux.HandleFunc("PATCH /api/user/me", app.WithAuth(app.PatchMyUsernameHandler))
	mux.HandleFunc("GET /api/user/me/posts", app.WithAuth(app.GetMyPostsHandler))
	mux.HandleFunc("GET /api/user/me/posts/public", app.WithAuth(app.GetPublicPostsHandler))
	mux.HandleFunc("GET /api/user/me/posts/following", app.WithAuth(app.GetFollowingPostsHandler))

	// Users — by id
	mux.HandleFunc("GET /api/user/{id}", app.WithAuth(app.GetOtherUserInfoHandler))
	mux.HandleFunc("POST /api/user/{id}/follow", app.WithAuth(app.FollowUserHandler))
	mux.HandleFunc("DELETE /api/user/{id}/follow", app.WithAuth(app.UnfollowUserHandler))

	// Make sure the directory is valid.
	if dir := strings.TrimSpace(os.Getenv("WEB_DIST_DIR")); dir != "" {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			h := SpaHandler(dir)
			mux.Handle("GET /{path...}", h)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
