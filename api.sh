#!/usr/bin/env bash
# ============================================================
# Go API Template — curl examples
# Usage: bash api.sh
# ============================================================

BASE_URL="http://localhost:8080"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="Admin123!"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

header() { echo -e "\n${CYAN}══ $1 ══${NC}"; }
label()  { echo -e "${YELLOW}→ $1${NC}"; }

# ─── Health ──────────────────────────────────────────────────
header "HEALTH"

label "Liveness"
curl -s "$BASE_URL/health" | python3 -m json.tool

label "Readiness"
curl -s "$BASE_URL/health/ready" | python3 -m json.tool

# ─── Auth ────────────────────────────────────────────────────
header "AUTH"

label "Register"
curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "Password123!",
    "first_name": "New",
    "last_name": "User"
  }' | python3 -m json.tool

label "Login (admin)"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}")
echo "$RESPONSE" | python3 -m json.tool

ACCESS_TOKEN=$(echo "$RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['access_token'])" 2>/dev/null)
REFRESH_TOKEN=$(echo "$RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['refresh_token'])" 2>/dev/null)
echo -e "${GREEN}Access token saved.${NC}"

label "Refresh Token"
curl -s -X POST "$BASE_URL/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | python3 -m json.tool

label "Forgot Password"
curl -s -X POST "$BASE_URL/api/v1/auth/forgot-password" \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@example.com"}' | python3 -m json.tool

label "Logout"
curl -s -X POST "$BASE_URL/api/v1/auth/logout" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

# ─── Users ───────────────────────────────────────────────────
header "USERS"

label "GET /users/me"
curl -s "$BASE_URL/api/v1/users/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "GET /users (paginated)"
curl -s "$BASE_URL/api/v1/users?page=1&page_size=10" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "GET /users (search)"
curl -s "$BASE_URL/api/v1/users?search=admin" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "GET /users (filter active)"
curl -s "$BASE_URL/api/v1/users?is_active=true" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "POST /users"
USER_RESP=$(curl -s -X POST "$BASE_URL/api/v1/users" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "created@example.com",
    "password": "Password123!",
    "first_name": "Created",
    "last_name": "ByAdmin",
    "is_active": true
  }')
echo "$USER_RESP" | python3 -m json.tool
USER_ID=$(echo "$USER_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['ID'])" 2>/dev/null)

if [ -n "$USER_ID" ]; then
  label "GET /users/:id"
  curl -s "$BASE_URL/api/v1/users/$USER_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

  label "PUT /users/:id"
  curl -s -X PUT "$BASE_URL/api/v1/users/$USER_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "first_name": "Updated",
      "last_name": "Name",
      "is_active": true
    }' | python3 -m json.tool

  label "DELETE /users/:id (soft delete)"
  curl -s -X DELETE "$BASE_URL/api/v1/users/$USER_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" -w "\nHTTP %{http_code}\n"
fi

label "Unauthorized (no token) — expect 401"
curl -s "$BASE_URL/api/v1/users" | python3 -m json.tool

# ─── Roles ───────────────────────────────────────────────────
header "ROLES"

label "GET /roles"
curl -s "$BASE_URL/api/v1/roles" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "POST /roles"
ROLE_RESP=$(curl -s -X POST "$BASE_URL/api/v1/roles" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "editor",
    "description": "Content editor role"
  }')
echo "$ROLE_RESP" | python3 -m json.tool
ROLE_ID=$(echo "$ROLE_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['ID'])" 2>/dev/null)

if [ -n "$ROLE_ID" ]; then
  label "GET /roles/:id"
  curl -s "$BASE_URL/api/v1/roles/$ROLE_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

  label "PUT /roles/:id"
  curl -s -X PUT "$BASE_URL/api/v1/roles/$ROLE_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "editor",
      "description": "Updated editor description"
    }' | python3 -m json.tool
fi

# ─── Permissions ─────────────────────────────────────────────
header "PERMISSIONS"

label "GET /permissions"
curl -s "$BASE_URL/api/v1/permissions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | python3 -m json.tool

label "POST /permissions"
PERM_RESP=$(curl -s -X POST "$BASE_URL/api/v1/permissions" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "reports:read",
    "description": "Read reports",
    "resource": "reports",
    "action": "read"
  }')
echo "$PERM_RESP" | python3 -m json.tool
PERM_ID=$(echo "$PERM_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['ID'])" 2>/dev/null)

if [ -n "$ROLE_ID" ] && [ -n "$PERM_ID" ]; then
  label "POST /roles/:id/permissions — assign permission to role"
  curl -s -X POST "$BASE_URL/api/v1/roles/$ROLE_ID/permissions" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"permission_id\":\"$PERM_ID\"}" | python3 -m json.tool

  label "DELETE /roles/:id/permissions/:permission_id — remove permission"
  curl -s -X DELETE "$BASE_URL/api/v1/roles/$ROLE_ID/permissions/$PERM_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" -w "\nHTTP %{http_code}\n"

  label "DELETE /permissions/:id"
  curl -s -X DELETE "$BASE_URL/api/v1/permissions/$PERM_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" -w "\nHTTP %{http_code}\n"
fi

if [ -n "$ROLE_ID" ]; then
  label "DELETE /roles/:id"
  curl -s -X DELETE "$BASE_URL/api/v1/roles/$ROLE_ID" \
    -H "Authorization: Bearer $ACCESS_TOKEN" -w "\nHTTP %{http_code}\n"
fi

echo -e "\n${GREEN}Done.${NC}"
