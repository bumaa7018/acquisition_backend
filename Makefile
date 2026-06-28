.PHONY: run dev build test test-cover lint migrate-up migrate-down migrate-auth-up migrate-auth-down migrate-main-up migrate-main-down up down logs seed seed-external-users seed-compensation-samples seed-valuation-samples seed-mock seed-mock-geometry swag deploy docker-rebuild fe-network fe-build fe-up fe-down fe-logs fe-restart all-up all-down

BINARY=server
CMD=./cmd/api
COMPOSE=docker compose -f deployments/docker-compose.yml --env-file .env
GOPATH_BIN=$(shell go env GOPATH 2>/dev/null)/bin
SWAG=$(GOPATH_BIN)/swag
GOLANGCI_LINT=$(GOPATH_BIN)/golangci-lint

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
	@command -v go >/dev/null 2>&1 || (echo "Go олдсонгүй. Ubuntu дээр Go суулгаад дахин ажиллуулна уу." && exit 1)
	@test -x "$(SWAG)" || (echo "swag суулгаж байна..." && go install github.com/swaggo/swag/cmd/swag@latest)
	CGO_ENABLED=0 "$(SWAG)" init -g cmd/api/main.go -o docs

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
	@command -v go >/dev/null 2>&1 || (echo "Go олдсонгүй. Go суулгаад дахин ажиллуулна уу." && exit 1)
	@test -x "$(GOLANGCI_LINT)" || (echo "golangci-lint суулгаж байна..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	"$(GOLANGCI_LINT)" run ./...

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
	@docker network inspect gov_network >/dev/null 2>&1 || docker network create gov_network >/dev/null
	$(COMPOSE) build --no-cache --pull
	$(COMPOSE) up -d --remove-orphans --wait

down:
	$(COMPOSE) down

reset:
	$(COMPOSE) down -v
	$(COMPOSE) build --no-cache --pull
	$(COMPOSE) up -d --remove-orphans --wait

logs:
	$(COMPOSE) logs -f

seed:
	go run $(CMD) -seed-only 2>/dev/null || go run $(CMD) &

seed-external-users:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db auth scripts/seed_external_access_users.sql

seed-compensation-samples:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_compensation_samples.sql

seed-valuation-samples:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_valuation_samples.sql

# seed_mock_data.sql + geometry нөхөлт (дараалан ажиллана)
seed-mock:
	docker exec -i $$($(COMPOSE) ps -q postgres) \
		psql -U $${DB_USER:-postgres} -d $${DB_NAME:-appdb} \
		< scripts/seed_mock_data.sql
	docker exec -i $$($(COMPOSE) ps -q postgres) \
		psql -U $${DB_USER:-postgres} -d $${DB_NAME:-appdb} \
		< scripts/patch_mock_geometry.sql

# Зөвхөн geometry нөхөлт — mock data аль хэдийн орсон үед
seed-mock-geometry:
	docker exec -i $$($(COMPOSE) ps -q postgres) \
		psql -U $${DB_USER:-postgres} -d $${DB_NAME:-appdb} \
		< scripts/patch_mock_geometry.sql

deploy:
	./scripts/deploy.sh

docker-rebuild:
	SKIP_GIT_PULL=1 ./scripts/deploy.sh

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
