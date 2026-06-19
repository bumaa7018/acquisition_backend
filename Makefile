.PHONY: run dev build test test-cover lint migrate-up migrate-down migrate-auth-up migrate-auth-down migrate-main-up migrate-main-down up down logs seed swag fe-network fe-build fe-up fe-down fe-logs fe-restart all-up all-down

BINARY=server
CMD=./cmd/api

# Үндсэн DB
DB_HOST     ?= $(shell grep '^DB_HOST'     .env 2>/dev/null | cut -d= -f2)
DB_PORT     ?= $(shell grep '^DB_PORT'     .env 2>/dev/null | cut -d= -f2)
DB_USER     ?= $(shell grep '^DB_USER'     .env 2>/dev/null | cut -d= -f2)
DB_PASS     ?= $(shell grep '^DB_PASSWORD' .env 2>/dev/null | cut -d= -f2)
DB_NAME     ?= $(shell grep '^DB_NAME'     .env 2>/dev/null | cut -d= -f2)
DB_SSL      ?= $(shell grep '^DB_SSLMODE'  .env 2>/dev/null | cut -d= -f2)
MAIN_DSN     = postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL)

# Auth DB
AUTH_DB_HOST ?= $(shell grep '^AUTH_DB_HOST'     .env 2>/dev/null | cut -d= -f2)
AUTH_DB_PORT ?= $(shell grep '^AUTH_DB_PORT'     .env 2>/dev/null | cut -d= -f2)
AUTH_DB_USER ?= $(shell grep '^AUTH_DB_USER'     .env 2>/dev/null | cut -d= -f2)
AUTH_DB_PASS ?= $(shell grep '^AUTH_DB_PASSWORD' .env 2>/dev/null | cut -d= -f2)
AUTH_DB_NAME ?= $(shell grep '^AUTH_DB_NAME'     .env 2>/dev/null | cut -d= -f2)
AUTH_DB_SSL  ?= $(shell grep '^AUTH_DB_SSLMODE'  .env 2>/dev/null | cut -d= -f2)
AUTH_DSN      = postgres://$(AUTH_DB_USER):$(AUTH_DB_PASS)@$(AUTH_DB_HOST):$(AUTH_DB_PORT)/$(AUTH_DB_NAME)?sslmode=$(AUTH_DB_SSL)

swag:
	CGO_ENABLED=0 $(shell go env GOPATH)/bin/swag init -g cmd/api/main.go -o docs

run: swag
	CGO_ENABLED=0 go run $(CMD)

dev:
	@which air > /dev/null 2>&1 || (echo "air суулгаж байна..." && go install github.com/air-verse/air@latest)
	CGO_ENABLED=0 $(shell go env GOPATH)/bin/air -c .air.toml

build:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY) $(CMD)

test:
	CGO_ENABLED=0 go test ./... -v -count=1

test-cover:
	CGO_ENABLED=0 go test ./... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run ./...

# Хоёр DB-д зэрэг migration
migrate-up: migrate-main-up migrate-auth-up

migrate-down: migrate-main-down migrate-auth-down

# Үндсэн DB migration
migrate-main-up:
	migrate -path ./migrations/main -database "$(MAIN_DSN)" up

migrate-main-down:
	migrate -path ./migrations/main -database "$(MAIN_DSN)" down 1

# Auth DB migration
migrate-auth-up:
	migrate -path ./migrations/auth -database "$(AUTH_DSN)" up

migrate-auth-down:
	migrate -path ./migrations/auth -database "$(AUTH_DSN)" down 1

up:
	docker compose -f deployments/docker-compose.yml --env-file .env up --build -d

down:
	docker compose -f deployments/docker-compose.yml down

logs:
	docker compose -f deployments/docker-compose.yml logs -f

seed:
	go run $(CMD) -seed-only 2>/dev/null || go run $(CMD) &

# ─── Frontend ──────────────────────────────────────────────────────────────────
FE_DIR ?= ../government-frontend

fe-network:
	@docker network inspect gov_network >/dev/null 2>&1 || \
		(echo "▶ gov_network үүсгэж байна..." && docker network create gov_network)

fe-build:
	@echo "▶ Frontend image бэлтгэж байна..."
	@docker compose -f $(FE_DIR)/docker-compose.yml build

fe-up: fe-network
	@echo "▶ Frontend асааж байна..."
	@docker compose -f $(FE_DIR)/docker-compose.yml up -d
	@echo "✓ Frontend: http://localhost:3000"

fe-down:
	@docker compose -f $(FE_DIR)/docker-compose.yml down

fe-logs:
	@docker compose -f $(FE_DIR)/docker-compose.yml logs -f

fe-restart: fe-down fe-up

# Бүгдийг (backend + frontend) хамт асаах
all-up: up fe-up
	@echo "✓ Backend + Frontend бэлэн"

all-down: fe-down down
