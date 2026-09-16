#!/bin/sh
set -eu

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
FORMULA_SOURCE="${YKC_FORMULA_SOURCE:-$ROOT/dist/homebrew/yunku-cli.rb}"
TAP_REPO_URL="${YKC_TAP_REPO_URL:-}"
TAP_BRANCH="${YKC_TAP_BRANCH:-main}"
TAP_FORMULA_PATH="${YKC_TAP_FORMULA_PATH:-Formula/yunku-cli.rb}"
COMMIT_MESSAGE="${YKC_TAP_COMMIT_MESSAGE:-chore: update yunku-cli formula}"
GIT_NAME="${YKC_GIT_NAME:-Yunku Release Bot}"
GIT_EMAIL="${YKC_GIT_EMAIL:-ykc-release-bot@example.com}"

say() {
  printf '%s\n' "$*"
}

err() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || err "missing required command: $1"
}

need_file() {
  [ -f "$1" ] || err "required file not found: $1"
}

need_env() {
  name="$1"
  value="$2"
  [ -n "$value" ] || err "missing required environment variable: $name"
}

checkout_tap_branch() {
  repo_url="$1"
  branch="$2"
  target_dir="$3"

  if git clone --branch "$branch" "$repo_url" "$target_dir" >/dev/null 2>&1; then
    return
  fi

  git clone "$repo_url" "$target_dir" >/dev/null 2>&1
  (
    cd "$target_dir"
    if git show-ref --verify --quiet "refs/remotes/origin/$branch"; then
      git checkout -B "$branch" "origin/$branch" >/dev/null 2>&1
      exit 0
    fi
    if git show-ref --verify --quiet "refs/heads/$branch"; then
      git checkout "$branch" >/dev/null 2>&1
      exit 0
    fi
    git checkout --orphan "$branch" >/dev/null 2>&1
  )
}

need_cmd git
need_env "YKC_TAP_REPO_URL" "$TAP_REPO_URL"
need_file "$FORMULA_SOURCE"

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/ykc-homebrew-publish-XXXXXX")"
cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT INT TERM

TAP_DIR="$TMP_ROOT/tap"
checkout_tap_branch "$TAP_REPO_URL" "$TAP_BRANCH" "$TAP_DIR"

DEST_PATH="$TAP_DIR/$TAP_FORMULA_PATH"
mkdir -p "$(dirname "$DEST_PATH")"
cp "$FORMULA_SOURCE" "$DEST_PATH"

(
  cd "$TAP_DIR"
  if [ -z "$(git status --short -- "$TAP_FORMULA_PATH")" ]; then
    say "No formula changes to publish."
    exit 0
  fi

  git config user.name "$GIT_NAME"
  git config user.email "$GIT_EMAIL"
  git add "$TAP_FORMULA_PATH"
  git commit -m "$COMMIT_MESSAGE" >/dev/null
  git push origin "HEAD:$TAP_BRANCH" >/dev/null
)

say "Published Homebrew formula to $TAP_REPO_URL ($TAP_BRANCH)"
