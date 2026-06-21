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
- PostgreSQL асна (healthcheck дамжуулсны дараа )
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
make up             # docker compose build/up + dangling cleanup
make down
make logs
make deploy         # git pull + docker build/up + dangling cleanup
```

## Auto Deploy

GitHub Actions нь `main` branch руу push/merge ороход deploy server рүү SSH
ороод `make deploy` ажиллуулна. Deploy server дээр repo
`/home/deploy/acquisition_backend` хавтаст clone хийгдсэн, `.env` файл байгаа, `deploy`
user Docker ажиллуулах эрхтэй байх ёстой.

GitHub repo дээр `Settings -> Secrets and variables -> Actions` дотор:

| Secret | Утга |
|--------|------|
| `DEPLOY_HOST` | Server IP/domain |
| `DEPLOY_USER` | `deploy` |
| `DEPLOY_PORT` | `22` |
| `DEPLOY_PATH` | `/home/deploy/acquisition_backend` |
| `DEPLOY_SSH_KEY` | Server рүү SSH орох private key |

`DEPLOY_SSH_KEY`-ийн public key нь server дээр
`/home/deploy/.ssh/authorized_keys` дотор байх ёстой. Харин server дээр
`make deploy` ажиллах үед `git pull` хийгдэх тул `deploy` user GitHub repo-г
унших эрхтэй байх хэрэгтэй. Үүнийг GitHub `Deploy keys` эсвэл deploy user-ийн
SSH key-р тохируулна.

Manual deploy:

```bash
make deploy
```

Deploy үед volume устгахгүй, бүх image устгахгүй. Docker build cache ашиглаж
image build хийгээд service-үүдийг асаана. Disk дүүрэхээс сэргийлж зөвхөн
dangling image болон ашиглагдахгүй build cache цэвэрлэнэ.

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
