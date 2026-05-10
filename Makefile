COMPOSE_FILE := docker-compose.dev.yml
COMPOSE := docker compose -f $(COMPOSE_FILE)

BACKEND_DIR := backend
BACKEND_BIN := $(CURDIR)/bin/backend

# Dev DB published by docker-compose.dev.yml (host → container).
MIGRATE_DB_ADDR ?= 127.0.0.1:5432

.PHONY: dev-up dev-build down logs clean ps migrate-up migrate-down migrate-down-all migrate-version backend-build backend-run rebuild-frontend

dev-up:
	$(COMPOSE) up -d

dev-build:
	$(COMPOSE) up -d --build

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f

logs-backend:
	$(COMPOSE) logs -f backend

logs-frontend:
	$(COMPOSE) logs -f frontend

rebuild-frontend:
	$(COMPOSE) rm -sf frontend
	$(COMPOSE) up -d --build frontend

logs-db:
	$(COMPOSE) logs -f db

ps:
	$(COMPOSE) ps

clean:
	$(COMPOSE) down -v --remove-orphans

# make migrate name=create_whatever_table
migrate:
	migrate create -ext sql -dir $(CURDIR)/$(BACKEND_DIR)/database/migrations -seq $(name)

# Optional steps=N (see migrate -help). Up with no N applies all pending; down with no N asks to drop ALL (use steps=1 or migrate-down-all).
migrate-up:
	set -a && . "$(CURDIR)/.env" && set +a && migrate -path $(CURDIR)/$(BACKEND_DIR)/database/migrations -database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(MIGRATE_DB_ADDR)/$$POSTGRES_DB?sslmode=disable" up $(steps)

migrate-down:
	set -a && . "$(CURDIR)/.env" && set +a && migrate -path $(CURDIR)/$(BACKEND_DIR)/database/migrations -database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(MIGRATE_DB_ADDR)/$$POSTGRES_DB?sslmode=disable" down $(steps)

migrate-down-all:
	set -a && . "$(CURDIR)/.env" && set +a && migrate -path $(CURDIR)/$(BACKEND_DIR)/database/migrations -database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(MIGRATE_DB_ADDR)/$$POSTGRES_DB?sslmode=disable" down -all

migrate-version:
	set -a && . "$(CURDIR)/.env" && set +a && migrate -path $(CURDIR)/$(BACKEND_DIR)/database/migrations -database "postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@$(MIGRATE_DB_ADDR)/$$POSTGRES_DB?sslmode=disable" version

# Run from repository root (kaima/) so process cwd stays the repo root (e.g. .env next to Makefile).
backend-build:
	go build -C $(BACKEND_DIR) ./...

backend-run: backend-build
	mkdir -p $(dir $(BACKEND_BIN))
	go build -C $(BACKEND_DIR) -o $(BACKEND_BIN) ./cmd
	$(BACKEND_BIN)
