#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
COMPOSE_FILE="${COMPOSE_FILE:-$ROOT_DIR/deployments/docker-compose.yml}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
GIT_REMOTE="${GIT_REMOTE:-origin}"
DB_VOLUME_NAME="${DB_VOLUME_NAME:-postgres_data}"

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

detect_project_name() {
  local container volume

  container="$(compose ps -q postgres 2>/dev/null || true)"
  if [[ -n "$container" ]]; then
    docker inspect "$container" --format '{{ index .Config.Labels "com.docker.compose.project" }}' 2>/dev/null || true
    return
  fi

  volume="$(docker volume ls -q --filter "label=com.docker.compose.volume=$DB_VOLUME_NAME" | head -n 1)"
  if [[ -n "$volume" ]]; then
    docker volume inspect "$volume" --format '{{ index .Labels "com.docker.compose.project" }}' 2>/dev/null || true
  fi
}

current_branch="$(git rev-parse --abbrev-ref HEAD)"
GIT_BRANCH="${GIT_BRANCH:-$current_branch}"

if [[ "${SKIP_GIT_PULL:-0}" != "1" ]]; then
  if [[ -n "$(git status --porcelain)" ]]; then
    echo "Working tree dirty байна. Deploy user дээр commit хийгдээгүй өөрчлөлт үлдсэн тул pull хийхгүй." >&2
    git status --short >&2
    exit 1
  fi

  echo "Git update шалгаж байна: $GIT_REMOTE/$GIT_BRANCH"
  git fetch --prune "$GIT_REMOTE"
  git pull --ff-only "$GIT_REMOTE" "$GIT_BRANCH"
fi

project_name="$(detect_project_name)"

echo "Docker container/image цэвэрлэж байна..."
compose down --remove-orphans --rmi local

if [[ -n "$project_name" ]]; then
  echo "Compose project-ийн DB биш volume-уудыг устгаж байна: $project_name"
  while IFS= read -r volume; do
    [[ -n "$volume" ]] || continue
    compose_volume="$(docker volume inspect "$volume" --format '{{ index .Labels "com.docker.compose.volume" }}' 2>/dev/null || true)"
    if [[ "$compose_volume" != "$DB_VOLUME_NAME" ]]; then
      docker volume rm "$volume" >/dev/null || true
    fi
  done < <(docker volume ls -q --filter "label=com.docker.compose.project=$project_name")
else
  echo "Compose project name олдсонгүй. Volume cleanup алгаслаа."
fi

echo "Unused Docker image/build cache цэвэрлэж байна..."
docker image prune -af
docker builder prune -af

echo "Docker image шинээр build хийж байна..."
compose build --pull --no-cache

echo "Docker service-үүдийг шинээр асааж байна..."
compose up -d --force-recreate --remove-orphans

compose ps
