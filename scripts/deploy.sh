#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.yml}"
LOGGING_COMPOSE_FILE="${LOGGING_COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.logging.yml}"
# Файлын систем (MinIO) — зөвхөн ЭНЭ script-ээр асна. Хасах бол compose файлыг
# устгахад болно: дараагийн deploy-д minio нь orphan болж цэвэрлэгдэж, api-гийн
# S3 тохиргоо/depends_on мөн арилна (дэлгэрэнгүй тайлбар нь файл дотор).
MINIO_COMPOSE_FILE="${MINIO_COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.minio.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
GIT_REMOTE="${GIT_REMOTE:-origin}"

cd "$ROOT_DIR"

if ! command -v git >/dev/null 2>&1; then
  echo "git олдсонгүй" >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker олдсонгүй" >&2
  exit 1
fi

if [[ ! -f "$ENV_FILE" ]]; then
  echo "$ENV_FILE файл байхгүй байна" >&2
  exit 1
fi

compose() {
  docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" "$@"
}

# api container нь docker/docker-compose CLI-гүй хөнгөн image дотор ажилладаг
# тул дотроосоо Loki/Promtail/Grafana-г асааж чадахгүй (internal/pkg/logstack
# зөвхөн локал dev орчинд air-г host дээр шууд ажиллуулах үед л ажиллана).
# Тиймээс production deploy бүрд ЭНД, host дээр шалгаж асаана.
#
# АНХААР: Loki/Promtail/Grafana болон MinIO мөн "deployments" compose project-д
# багтдаг тул зөвхөн $COMPOSE_FILE-ээр --remove-orphans ажиллуулбал compose
# тэдгээрийг "orphan" гэж үзэж УСТГАНА. Тиймээс бүх файлыг ХАМТ, НЭГ дуудлагаар
# up хийнэ — эс тэгвээс эхний (зөвхөн main) up дуудлага logging stack/MinIO-г
# устгаад, дараа нь дахин үүсгэх хэрэгтэй болно (хэрэглээгүй ч устгал/үүсгэлт
# давхардана, эсвэл дараагийн алхам амжилтгүй бол тэдгээр устсан хэвээр үлдэнэ).
compose_full() {
  local files=(-f "$COMPOSE_FILE")
  # Файл байхгүй бол тэр сервис оролцохгүй — хасах гэдэг нь файлыг устгах.
  # `if` хэлбэрээр бичсэн нь санамсаргүй биш: `[[ ]] && ...` нь set -e-тэй
  # хамт уншихад хоёрдмол утгатай (файл байхгүй үед script унах эсэх нь
  # bash-ийн хувилбар/нөхцлөөс хамаарч мэдэгддэггүй).
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

current_branch="$(git rev-parse --abbrev-ref HEAD)"
GIT_BRANCH="${GIT_BRANCH:-$current_branch}"

if [[ "${SKIP_GIT_PULL:-0}" != "1" ]]; then
  if [[ -n "$(git status --porcelain)" ]]; then
    echo "Working tree dirty байна. Deploy user дээр commit хийгдээгүй өөрчлөлт үлдсэн тул pull хийхгүй." >&2
    git status --short >&2
    exit 1
  fi

  old_sha="$(git rev-parse HEAD)"

  echo "Git update шалгаж байна: $GIT_REMOTE/$GIT_BRANCH"
  git fetch --prune "$GIT_REMOTE"
  git pull --ff-only "$GIT_REMOTE" "$GIT_BRANCH"

  new_sha="$(git rev-parse HEAD)"
  if [[ "$old_sha" != "$new_sha" && "${DEPLOY_REEXECED:-0}" != "1" ]]; then
    echo "Repo шинэчлэгдсэн тул deploy script-ийг шинэ хувилбараар дахин ажиллуулж байна..."
    exec env SKIP_GIT_PULL=1 DEPLOY_REEXECED=1 "$ROOT_DIR/scripts/deploy.sh"
  fi
fi

ensure_network

echo "Docker image build хийж байна..."
compose build ${BUILD_FLAGS:---pull}

echo "Docker service-үүдийг шинээр асааж байна..."
compose_full up -d --remove-orphans --pull missing --wait

compose_full ps

# Frontend container байвал restart хийнэ — backend шинэчлэгдсэний дараа
# DNS cache цэвэрлэгдэж, шинэ api container-тай дахин холбогдоно
FRONTEND=$(docker ps --filter "label=com.docker.compose.service=frontend" --format "{{.Names}}" 2>/dev/null | head -1)
if [[ -n "$FRONTEND" ]]; then
  echo "Frontend container ($FRONTEND) restart хийж байна..."
  docker restart "$FRONTEND"
fi
