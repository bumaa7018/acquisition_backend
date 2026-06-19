#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
LOCK_FILE="${LOCK_FILE:-/tmp/government-auto-deploy.lock}"
GIT_REMOTE="${GIT_REMOTE:-origin}"

exec 9>"$LOCK_FILE"
if ! flock -n 9; then
  echo "Deploy аль хэдийн ажиллаж байна. Алгаслаа."
  exit 0
fi

cd "$ROOT_DIR"

current_branch="$(git rev-parse --abbrev-ref HEAD)"
GIT_BRANCH="${GIT_BRANCH:-$current_branch}"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "Working tree dirty байна. Автомат deploy хийхгүй." >&2
  git status --short >&2
  exit 1
fi

git fetch --prune "$GIT_REMOTE"

local_sha="$(git rev-parse HEAD)"
remote_sha="$(git rev-parse "$GIT_REMOTE/$GIT_BRANCH")"

if [[ "$local_sha" == "$remote_sha" ]]; then
  echo "Шинэ commit алга: $local_sha"
  exit 0
fi

echo "Шинэ commit илэрлээ: $local_sha -> $remote_sha"
exec "$ROOT_DIR/scripts/deploy.sh"
