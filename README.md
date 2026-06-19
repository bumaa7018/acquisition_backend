# Go REST API Template

Production-ready Go REST API with Clean Architecture — Gin, PostgreSQL, JWT, RBAC.

## Quick Start

```bash
git clone <repo>
cd go-api-template
cp .env.example .env
make up
```

`make up` дуусмагц:
- PostgreSQL асна (healthcheck дамжуулсны дараа)
- Migration автоматаар хийгдэнэ
- Admin хэрэглэгч болон permission-ууд seed хийгдэнэ
- API `http://localhost:8080` дээр ажиллана

## Stack

| Layer | Technology |
|-------|-----------|
| HTTP | Gin |
| Database | PostgreSQL 16 + pgx/v5 |
| Migration | golang-migrate |
| Auth | JWT (access + refresh) + bcrypt |
| Config | caarlos0/env + godotenv |
| Validation | go-playground/validator |
| Logging | slog (structured JSON) |
| Testing | testify + httptest |

## Architecture

```
cmd/api/          — entrypoint, DI wiring, graceful shutdown
internal/
  config/         — env loading
  domain/         — entities, domain errors
  repository/     — interfaces + postgres implementations
  service/        — business logic
  handler/http/   — Gin handlers, DTOs
  middleware/     — JWT auth, RBAC, logging, recovery, CORS
  pkg/jwt/        — token issue/verify
  pkg/hash/       — bcrypt wrapper
  pkg/response/   — unified API response
migrations/       — *.up.sql / *.down.sql
deployments/      — Dockerfile, docker-compose.yml
```

## Endpoints

### Auth (public)
```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
POST /api/v1/auth/forgot-password
POST /api/v1/auth/reset-password
```

### Users (protected)
```
GET    /api/v1/users          — list (permission: users:read)
GET    /api/v1/users/me       — own profile (JWT required)
GET    /api/v1/users/:id      — (users:read)
POST   /api/v1/users          — (users:create)
PUT    /api/v1/users/:id      — (users:update)
DELETE /api/v1/users/:id      — soft delete (users:delete)
```

### Roles & Permissions (protected)
```
GET/POST/PUT/DELETE /api/v1/roles
POST   /api/v1/roles/:id/permissions
DELETE /api/v1/roles/:id/permissions/:permission_id
GET/POST/DELETE     /api/v1/permissions
```

### Health
```
GET /health        — liveness
GET /health/ready  — readiness (DB ping)
```

## Makefile Commands

```bash
make run            # локалаар ажиллуулах
make build          # бинар build
make test           # бүх unit тест
make test-cover     # coverage тайлантай
make lint           # golangci-lint
make migrate-up     # migration
make migrate-down
make up             # docker-compose up --build -d
make down
make logs
make deploy         # git pull + docker clean rebuild/up
make deploy-check   # remote commit өөрчлөгдсөн үед deploy хийх
```

## Auto Deploy

Deploy server дээр repo-г `deploy` хэрэглэгчийн `/home/deploy/government`
хавтаст clone хийсэн гэж үзнэ. GitHub/Git remote дээр `main` branch-д шинэ commit
орж ирэхэд systemd timer 30 секунд тутам шалгаад pull хийж Docker-ийг шинээр
build/recreate хийнэ.

```bash
sudo cp deployments/systemd/government-auto-deploy.service /etc/systemd/system/
sudo cp deployments/systemd/government-auto-deploy.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now government-auto-deploy.timer
```

Manual deploy:

```bash
make deploy
```

Deploy үед `postgres_data` volume устахгүй. Compose project-ийн бусад volume,
хуучин local image, unused image болон build cache цэвэрлэгдээд image дахин
build хийгдэнэ.

## curl Examples

```bash
# Health check
curl http://localhost:8080/health

# Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user@example.com","password":"Password123!","first_name":"John","last_name":"Doe"}'

# Login (admin)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"Admin123!"}'

# Use token
TOKEN=<access_token_from_login>

# List users
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN"

# Get own profile
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"

# Refresh token
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<your_refresh_token>"}'

# Forgot password (returns token in response for dev; in prod — email хаяглана)
curl -X POST http://localhost:8080/api/v1/auth/forgot-password \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com"}'
```

## Response Format

```json
// Success
{ "data": { ... } }

// Paginated
{ "data": [...], "total": 100, "page": 1, "page_size": 20, "total_pages": 5 }

// Error
{ "error": "not found" }
```

## Local Development

```bash
# .env файл үүсгэх (DB_HOST=localhost болгох)
cp .env.example .env
# DB_HOST=localhost гэж өөрчил

# Migrate
make migrate-up

# Run
make run
```

## Testing

```bash
make test
# эсвэл
CGO_ENABLED=0 go test ./... -v
```

> **Note:** macOS Tahoe (26.x) дээр Go 1.22-той `CGO_ENABLED=0` шаардлагатай (dyld LC_UUID issue).
