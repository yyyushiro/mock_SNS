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
	mux.HandleFunc("GET /api/auth/google/start", app.authenticationURIHandler)
	mux.HandleFunc("GET /api/auth/callback/google", app.getAccessTokenHandler)
	mux.HandleFunc("POST /api/posts", app.makePostHandler)
	mux.HandleFunc("GET /api/user/me/posts", app.getMyPostsHandler)
	mux.HandleFunc("DELETE /api/posts/{id}", app.deletePostHandler)
	mux.HandleFunc("POST /api/posts/{id}/likes", app.likePostHandler)
	mux.HandleFunc("DELETE /api/posts/{id}/likes", app.undoLikePostHandler)

	if dir := strings.TrimSpace(os.Getenv("WEB_DIST_DIR")); dir != "" {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			h := spaHandler(dir)
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
