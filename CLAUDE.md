# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Kaima** — a mock SNS (social network service) with Google OAuth + email/password auth, posts, likes, and follows. The goal is demonstrating secure authentication patterns, not novelty.

## Commands

### Development (Docker Compose)

```bash
make dev-build   # start all services (backend, frontend, postgres, redis) with rebuild
make dev-up      # start without rebuilding images
make down        # stop all services
make clean       # stop and remove volumes
make logs        # tail all logs
make logs-backend
make logs-frontend
make rebuild-frontend  # rebuild just the frontend container
```

Ports: frontend (Vite) `5173`, API `8080`, PostgreSQL `5432`, Redis `6379`.

### Migrations

Requires the `migrate` CLI and a running PostgreSQL (via compose).

```bash
make migrate name=create_something_table   # create new migration files
make migrate-up                            # apply all pending migrations
make migrate-up steps=1                    # apply N migrations
make migrate-down steps=1                  # roll back N migrations
make migrate-down-all                      # roll back everything
make migrate-version                       # show current version
```

### Backend (standalone)

```bash
make backend-build   # go build ./... inside backend/
make backend-run     # build + run binary (reads .env from repo root)
```

### Frontend (standalone)

```bash
cd frontend
npm run dev     # Vite dev server
npm run build   # tsc + vite build
npm run lint    # eslint
```

## Architecture

### Backend (`backend/cmd/`)

All backend code is a single `main` package under `backend/cmd/`. No web framework — standard `net/http` + `ServeMux`.

**Adding a new endpoint — the pattern:**

1. Add a DB function in `database.go`. Signature convention: `func FooBar(... pool *pgxpool.Pool, ctx context.Context) (ResultType, error)`. Domain types (`Post`, `User`) are also defined here.
2. Add a handler method on `*App` in the appropriate `handlers_*.go` file. The three handler files are `handlers_auth.go`, `handlers_posts.go`, `handlers_users.go`.
3. Register the route in `main.go`, wrapping with `app.WithAuth(handler)` if auth is required.

**Auth context in handlers:**

```go
result, ok := AuthFromRequest(r)  // panics are prevented by WithAuth already running
if !ok {
    http.Error(w, "internal error", http.StatusInternalServerError)
    return
}
userID := result.Sub  // uuid.UUID — the authenticated user's ID
```

`WithAuth` (`app.go`) injects `AuthResult` into the request context. `AuthFromRequest` retrieves it. Never call `AuthFromRequest` on an unprotected handler.

**Writing JSON responses:**

```go
err = WriteJsonResponseBody(w, payload, http.StatusOK)  // http_util.go
if err != nil {
    if errors.Is(err, ErrJsonEncode) {
        http.Error(w, "internal error", http.StatusInternalServerError)
    }
    // non-ErrJsonEncode means write already started — don't send a second response
    return
}
```

Response structs (e.g. `PostResponse`) are defined in handler files, not in `database.go`. DB types (`Post`, `User`) are internal; convert them before sending.

**Nullable columns:** `users.username`, `users.email`, `users.google_sub`, `users.hashed_password` are all nullable. They scan into `sql.NullString`. Use `sqlNullStringToString(ns)` (`database.go`) to convert to a plain string.

**DB timeouts:** all queries use a `context.WithTimeout(r.Context(), 5*time.Second)` context; defer the cancel immediately.

**Migrations:** `golang-migrate` SQL files in `backend/database/migrations/`. `make migrate name=foo` creates a new up/down pair.

### Frontend (`frontend/`)

- Routes are defined in `App.tsx`. Pages live in `frontend/pages/`.
- **All authenticated API calls go through `apiFetch`** (`apis/API.ts`), never raw `fetch`. `apiFetch` adds `credentials: "include"`, and on 401 it calls `POST /api/auth/refresh` once (deduplicated) then retries. Calls to `/api/auth/refresh` and `/api/auth/logout` bypass the retry to avoid loops.
- Auth endpoints that don't require a session (`/api/auth/register`, `/api/auth/login`) use plain `fetch` directly.
- Wire types (e.g. `PostWire` with all `unknown` fields) are used to safely parse JSON; then a converter function (e.g. `postRowFromJson`) maps them to the typed export type.

### Infrastructure

- **Dev**: `docker-compose.dev.yml` — backend, Vite frontend (hot reload), PostgreSQL 18, Redis. Vite proxies `/api/**` → `backend:8080`.
- **Prod**: `fly.toml` + `Dockerfile`. Go binary serves both API and the built SPA (`GET /{path...}` via `SpaHandler`) when `WEB_DIST_DIR` is set.

## Auth Design Constraints

- **Refresh tokens are only issued** during OAuth callback (`GET /api/auth/callback/google`) and email verification (`GET /api/auth/verify-email`). `POST /api/auth/refresh` only converts an existing refresh token → new access token; it never issues new refresh tokens. Issuing a refresh token from a valid access token alone would be a privilege escalation.
- **Email verification tokens** use Redis `GETDEL` (atomic get-and-delete) to prevent replay attacks.
- The `nonce` cookie prevents ID token replay; `state` cookie prevents CSRF in the OAuth flow.
