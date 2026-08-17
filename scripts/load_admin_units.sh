#!/usr/bin/env bash
# Засаг захиргааны нэгжийн өгөгдлийг (au1/au2/au3) backup_data/-аас appdb руу ачаална.
#
# Дампууд нь ӨӨР системээс авагдсан тул дотроо admin_units.au_levelN гэсэн нэр
# агуулдаг. Энэ script нь файлыг ӨӨРЧЛӨХГҮЙ — ачаалах урсгал дээр л нэрийг
# appdb-ийн public.auN руу хөрвүүлнэ.
#
# Хэрэглээ:
#   ./scripts/load_admin_units.sh          # хоосон хүснэгтэд ачаална
#   FORCE=1 ./scripts/load_admin_units.sh  # байгаа мөрийг УСТГААД дахин ачаална
set -Eeuo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
DATA_DIR="${DATA_DIR:-$ROOT_DIR/backup_data}"

cd "$ROOT_DIR"

env_value() {
  local key="$1" default="${2:-}" value
  value="$(grep -E "^${key}=" "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- || true)"
  value="${value%$'\r'}"; value="${value#\"}"; value="${value%\"}"
  printf '%s' "${value:-$default}"
}

DB_USER="${DB_USER:-$(env_value DB_USER postgres)}"
DB_NAME="${DB_NAME:-$(env_value DB_NAME appdb)}"

PG_CONTAINER="${PG_CONTAINER:-$(docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps -q postgres 2>/dev/null || true)}"
if [[ -z "$PG_CONTAINER" ]]; then
  echo "postgres container олдсонгүй (PG_CONTAINER=... өгч болно)" >&2
  exit 1
fi

psql_run() { docker exec -i "$PG_CONTAINER" psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" "$@"; }

# Хамгийн сүүлийн огноотой дампыг сонгоно (au_level1_YYYYMMDDHHMM.sql).
latest_dump() {
  local level="$1"
  ls -1 "$DATA_DIR"/au_level${level}_*.sql 2>/dev/null | sort | tail -1
}

if [[ "${FORCE:-0}" == "1" ]]; then
  echo "FORCE=1 — au1/au2/au3 хоослож байна"
  psql_run -q -c 'TRUNCATE public.au1, public.au2, public.au3;'
fi

for level in 1 2 3; do
  file="$(latest_dump "$level")"
  if [[ -z "$file" ]]; then
    echo "au${level}: дамп олдсонгүй ($DATA_DIR/au_level${level}_*.sql) — алгаслаа"
    continue
  fi

  rows="$(psql_run -Atc "SELECT count(*) FROM public.au${level};")"
  if [[ "$rows" != "0" ]]; then
    echo "au${level}: аль хэдийн $rows мөртэй — алгаслаа (дахин ачаалах бол FORCE=1)"
    continue
  fi

  echo "au${level}: $(basename "$file") ачаалж байна..."
  # Нэг гүйлгээгээр: алдаа гарвал бүхэлдээ буцна.
  sed -e 's/admin_units\.au_level1/public.au1/g' \
      -e 's/admin_units\.au_level2/public.au2/g' \
      -e 's/admin_units\.au_level3/public.au3/g' "$file" \
    | psql_run -q -1
  echo "au${level}: $(psql_run -Atc "SELECT count(*) FROM public.au${level};") мөр орлоо"
done

echo "Дүн:"
psql_run -c "SELECT 'au1' t, count(*) FROM public.au1 UNION ALL SELECT 'au2', count(*) FROM public.au2 UNION ALL SELECT 'au3', count(*) FROM public.au3 ORDER BY 1;"
