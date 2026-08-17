#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.yml}"
LOGGING_COMPOSE_FILE="${LOGGING_COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.logging.yml}"
MINIO_COMPOSE_FILE="${MINIO_COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.minio.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"

cd "$ROOT_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "$ENV_FILE файл байхгүй байна" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker олдсонгүй" >&2
  exit 1
fi

env_value() {
  local key="$1"
  local default="${2:-}"
  local value
  value="$(grep -E "^${key}=" "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- || true)"
  value="${value%$'\r'}"
  value="${value#\"}"
  value="${value%\"}"
  value="${value#\'}"
  value="${value%\'}"
  if [[ -n "$value" ]]; then
    printf '%s' "$value"
  else
    printf '%s' "$default"
  fi
}

DB_USER="${DB_USER:-$(env_value DB_USER postgres)}"
DB_NAME="${DB_NAME:-$(env_value DB_NAME appdb)}"
AUTH_DB_NAME="${AUTH_DB_NAME:-$(env_value AUTH_DB_NAME authdb)}"

compose_full() {
  local files=(-f "$COMPOSE_FILE")
  if [[ -f "$MINIO_COMPOSE_FILE" ]]; then
    files+=(-f "$MINIO_COMPOSE_FILE")
  fi
  if [[ -f "$LOGGING_COMPOSE_FILE" ]]; then
    files+=(-f "$LOGGING_COMPOSE_FILE")
  fi
  docker compose "${files[@]}" --env-file "$ENV_FILE" "$@"
}

ensure_network() {
  docker network inspect gov_network >/dev/null 2>&1 || docker network create gov_network >/dev/null
}

reset_database() {
  local db_name="$1"
  local pg_container="$2"

  echo "DB цэвэрлэж байна: $db_name"
  docker exec -i "$pg_container" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d postgres \
    -v db="$db_name" -v owner="$DB_USER" <<'SQL'
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = :'db'
  AND pid <> pg_backend_pid();

SELECT format('DROP DATABASE IF EXISTS %I', :'db') \gexec
SELECT format('CREATE DATABASE %I OWNER %I', :'db', :'owner') \gexec
SQL
}

verify_database() {
  local db_name="$1"
  local pg_container="$2"

  echo "Migration state: $db_name"
  docker exec -i "$pg_container" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$db_name" \
    -c "SELECT current_database(), version, dirty FROM schema_migrations;"
}

verify_main_reference_data() {
  local pg_container="$1"

  echo "Main DB лавлах/config data шалгаж байна..."
  docker exec -i "$pg_container" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" <<'SQL'
WITH checks(name, rows, min_rows) AS (
  VALUES
    ('document_type',              (SELECT COUNT(*) FROM document_type), 1),
    ('parcel_status',              (SELECT COUNT(*) FROM parcel_status), 6),
    ('construction_type',          (SELECT COUNT(*) FROM construction_type), 1),
    ('asset_spec_type',            (SELECT COUNT(*) FROM asset_spec_type), 1),
    ('asset_calc_type',            (SELECT COUNT(*) FROM asset_calc_type), 1),
    ('decision_work_type',         (SELECT COUNT(*) FROM decision_work_type), 1),
    ('decision_budget',            (SELECT COUNT(*) FROM decision_budget), 1),
    ('sdplatform.sd_system',       (SELECT COUNT(*) FROM sdplatform.sd_system WHERE system_id = 10), 1),
    ('sdplatform.sd_role',         (SELECT COUNT(*) FROM sdplatform.sd_role), 1),
    ('sdplatform.sd_permission',   (SELECT COUNT(*) FROM sdplatform.sd_permission), 1)
),
missing AS (
  SELECT string_agg(name || '=' || rows || ' < ' || min_rows, ', ') AS message
  FROM checks
  WHERE rows < min_rows
)
SELECT name, rows, min_rows
FROM checks
ORDER BY name;

DO $$
DECLARE
  missing text;
BEGIN
  WITH checks(name, rows, min_rows) AS (
    VALUES
      ('document_type',              (SELECT COUNT(*) FROM document_type), 1),
      ('parcel_status',              (SELECT COUNT(*) FROM parcel_status), 6),
      ('construction_type',          (SELECT COUNT(*) FROM construction_type), 1),
      ('asset_spec_type',            (SELECT COUNT(*) FROM asset_spec_type), 1),
      ('asset_calc_type',            (SELECT COUNT(*) FROM asset_calc_type), 1),
      ('decision_work_type',         (SELECT COUNT(*) FROM decision_work_type), 1),
      ('decision_budget',            (SELECT COUNT(*) FROM decision_budget), 1),
      ('sdplatform.sd_system',       (SELECT COUNT(*) FROM sdplatform.sd_system WHERE system_id = 10), 1),
      ('sdplatform.sd_role',         (SELECT COUNT(*) FROM sdplatform.sd_role), 1),
      ('sdplatform.sd_permission',   (SELECT COUNT(*) FROM sdplatform.sd_permission), 1)
  )
  SELECT string_agg(name || '=' || rows || ' < ' || min_rows, ', ')
    INTO missing
  FROM checks
  WHERE rows < min_rows;

  IF missing IS NOT NULL THEN
    RAISE EXCEPTION 'required main reference data missing: %', missing;
  END IF;
END $$;
SQL
}

verify_auth_reference_data() {
  local pg_container="$1"

  echo "Auth DB лавлах/config data шалгаж байна..."
  docker exec -i "$pg_container" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$AUTH_DB_NAME" <<'SQL'
WITH checks(name, rows, min_rows) AS (
  VALUES
    ('sdplatform.sd_user',       (SELECT COUNT(*) FROM sdplatform.sd_user WHERE deleted_at IS NULL), 1),
    ('sdplatform.sd_role',       (SELECT COUNT(*) FROM sdplatform.sd_role), 1),
    ('sdplatform.sd_permission', (SELECT COUNT(*) FROM sdplatform.sd_permission), 1),
    ('sdplatform.sd_system',     (SELECT COUNT(*) FROM sdplatform.sd_system WHERE system_id = 10), 1),
    ('sdplatform.sd_menu',       (SELECT COUNT(*) FROM sdplatform.sd_menu), 1)
),
missing AS (
  SELECT string_agg(name || '=' || rows || ' < ' || min_rows, ', ') AS message
  FROM checks
  WHERE rows < min_rows
)
SELECT name, rows, min_rows
FROM checks
ORDER BY name;

DO $$
DECLARE
  missing text;
BEGIN
  WITH checks(name, rows, min_rows) AS (
    VALUES
      ('sdplatform.sd_user',       (SELECT COUNT(*) FROM sdplatform.sd_user WHERE deleted_at IS NULL), 1),
      ('sdplatform.sd_role',       (SELECT COUNT(*) FROM sdplatform.sd_role), 1),
      ('sdplatform.sd_permission', (SELECT COUNT(*) FROM sdplatform.sd_permission), 1),
      ('sdplatform.sd_system',     (SELECT COUNT(*) FROM sdplatform.sd_system WHERE system_id = 10), 1),
      ('sdplatform.sd_menu',       (SELECT COUNT(*) FROM sdplatform.sd_menu), 1)
  )
  SELECT string_agg(name || '=' || rows || ' < ' || min_rows, ', ')
    INTO missing
  FROM checks
  WHERE rows < min_rows;

  IF missing IS NOT NULL THEN
    RAISE EXCEPTION 'required auth reference data missing: %', missing;
  END IF;
END $$;
SQL
}

ensure_network

echo "API-г түр зогсоож байна..."
compose_full stop api >/dev/null 2>&1 || true

echo "Postgres асааж байна..."
compose_full up -d postgres --wait

PG_CONTAINER="$(compose_full ps -q postgres)"
if [[ -z "$PG_CONTAINER" ]]; then
  echo "postgres container олдсонгүй" >&2
  exit 1
fi

reset_database "$DB_NAME" "$PG_CONTAINER"
if [[ "$AUTH_DB_NAME" != "$DB_NAME" ]]; then
  reset_database "$AUTH_DB_NAME" "$PG_CONTAINER"
fi

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  echo "API image build хийж байна..."
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" build ${BUILD_FLAGS:---pull} api
fi

echo "API-г асааж migration + startup seeder ажиллуулж байна..."
compose_full up -d api --wait

verify_database "$DB_NAME" "$PG_CONTAINER"
if [[ "$AUTH_DB_NAME" != "$DB_NAME" ]]; then
  verify_database "$AUTH_DB_NAME" "$PG_CONTAINER"
fi
verify_main_reference_data "$PG_CONTAINER"
if [[ "$AUTH_DB_NAME" != "$DB_NAME" ]]; then
  verify_auth_reference_data "$PG_CONTAINER"
fi

echo "DB reset дууслаа. API healthy, migrations dirty=false байх ёстой."
