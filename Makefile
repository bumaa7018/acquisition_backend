.PHONY: run dev build test test-e2e test-cover lint migrate-up migrate-down migrate-auth-up migrate-auth-down migrate-main-up migrate-main-down up down reset reset-db logs seed seed-admin-example seed-external-users seed-compensation-samples seed-valuation-samples seed-parcel-workflow seed-parcel-samples seed-mock seed-mock-geometry swag deploy docker-rebuild middleware-run middleware-docker-build middleware-docker-up middleware-docker-down middleware-docker-logs middleware-docker-restart fe-network fe-build fe-up fe-down fe-logs fe-restart all-up all-down

# GOROOT-ыг Go өөрөө тодорхойлно. gvm (~/.zshrc) нь GOROOT=/usr/local/go гэж
# тавьдаг ба тэр нь go.mod-ийн шаардаж татдаг toolchain-аас (GOTOOLCHAIN=auto)
# ЗӨРВӨЛ std сан "олдохгүй" болно: go build "package crypto/mlkem is not in
# std" гэж унана, golangci-lint нь go/packages дээр timeout-д хүрнэ. Тиймээс
# recipe-үүд рүү GOROOT дамжуулахгүй.
unexport GOROOT

BINARY=server
CMD=./cmd/api
COMPOSE=docker compose -f deployments/docker-compose.yml --env-file .env
MW_DIR ?= ../goverment-middleware
# Файлын систем (MinIO) — хасах бол compose файлыг устгахад болно, тиймээс
# байгаа эсэхийг wildcard-аар шалгана (scripts/deploy.sh-ийн адил).
MINIO_FILE=deployments/docker-compose.minio.yml
MINIO_ARG=$(if $(wildcard $(MINIO_FILE)),-f $(MINIO_FILE))
# up/down/reset-д ашиглана — MinIO болон Loki/Promtail/Grafana мөн адил
# "deployments" project-д багтдаг тул зөвхөн үндсэн compose файлаар
# --remove-orphans ажиллуулбал тэдгээрийг "orphan" гэж үзэж УСТГАЖ ЧАДНА. Бүх
# файлыг ХАМТ дамжуулснаар compose бүх зорилтот сервисийг таньж, буруу
# устгахаас сэргийлнэ. Мөн api-гийн S3 тохиргоо (S3_ENDPOINT=minio:9000) нь
# minio файл дотор байдаг тул түүнгүйгээр api хадгалалт бэлдэж чадалгүй зогсоно.
FULL_COMPOSE=docker compose -f deployments/docker-compose.yml $(MINIO_ARG) -f deployments/docker-compose.logging.yml --env-file .env
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

# E2E — БОДИТ PostgreSQL шаардана (make up + make migrate-main-up).
# .env-ийн DB_* хувьсагчаар холбогдоно; DB байхгүй бол тест skip болно.
test-e2e:
	E2E_DATABASE_URL="$(MAIN_DSN)" CGO_ENABLED=0 go test -tags e2e ./... -v -count=1 -run E2E

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
	CGO_ENABLED=0 go run ./cmd/seed-sql -db auth migrations/auth/000001_authdb_schema.up.sql

migrate-auth-down:
	@echo "authdb schema down disabled: sdplatform-owned tables must not be dropped"

up:
	@docker network inspect gov_network >/dev/null 2>&1 || docker network create gov_network >/dev/null
	$(COMPOSE) build --no-cache --pull
	$(FULL_COMPOSE) up -d --remove-orphans --pull missing --wait

down:
	$(FULL_COMPOSE) down

# АНХААР: -v нь minio_data volume-ыг МӨН устгана — файлын системд байршуулсан
# бүх файл (баримт, зураг, дроны ортофото) DB-тэй хамт цэвэрлэгдэнэ.
#
# ХОРИГЛОХ ТОХИОЛДОЛ: postgres volume-ыг устгах тул AUTH_DB (sdplatform) сан ч
# бүрэн арчигдана. Жинхэнэ sdplatform өгөгдлийн санд холбогдсон үед энэ target-ыг
# ОГТ хэрэглэж БОЛОХГҮЙ. Аппын санг л шинэчлэх бол `make reset-db` — тэр нь
# authdb-д хүрдэггүй (байхгүй бол л үүсгэнэ).
reset:
	$(FULL_COMPOSE) down -v
	$(COMPOSE) build --no-cache --pull
	$(FULL_COMPOSE) up -d --remove-orphans --pull missing --wait

reset-db:
	./scripts/reset_databases.sh

logs:
	$(FULL_COMPOSE) logs -f

seed:
	go run $(CMD) -seed-only 2>/dev/null || go run $(CMD) &

seed-admin-example:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db auth scripts/seed_admin_example_admin_role.sql

seed-external-users:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db auth scripts/seed_external_access_users.sql

seed-compensation-samples:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_compensation_samples.sql

seed-valuation-samples:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_valuation_samples.sql

seed-parcel-workflow:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_parcel_workflow.sql

# Нэгж талбарын ЖИШЭЭ өгөгдөл: өмчлөгчийн нэр + байршлын давхардал.
# Эзэмшигч нь ГУС-аас татагддаг, давхардал нь татах мөчид тооцогддог тул
# аль хэдийн бүртгэгдсэн өгөгдөл дээр хоёулаа хоосон байдаг — дэлгэц дээр
# харагдуулахын тулд үүнийг ажиллуулна. Idempotent.
seed-parcel-samples:
	CGO_ENABLED=0 go run ./cmd/seed-sql -db main scripts/seed_parcel_samples.sql

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

# ─── Middleware service ───────────────────────────────────────────────────────
middleware-run:
	$(MAKE) -C $(MW_DIR) run

middleware-docker-build:
	$(MAKE) -C $(MW_DIR) docker-build

middleware-docker-up:
	$(MAKE) -C $(MW_DIR) docker-up

middleware-docker-down:
	$(MAKE) -C $(MW_DIR) docker-down

middleware-docker-logs:
	$(MAKE) -C $(MW_DIR) docker-logs

middleware-docker-restart:
	$(MAKE) -C $(MW_DIR) docker-restart

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
