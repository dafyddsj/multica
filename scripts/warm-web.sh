#!/usr/bin/env bash
# HTTP-GET the verify-multica demo routes so next dev --webpack compiles them
# before the first real browser visit. Any HTTP status (including 404) means
# the module compiled; only connection failure (000 / empty) is a miss.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FRONTEND_PORT="${FRONTEND_PORT:-3000}"
FRONTEND_ORIGIN="${FRONTEND_ORIGIN:-http://127.0.0.1:${FRONTEND_PORT}}"
WARM_WEB_WAIT_SECS="${WARM_WEB_WAIT_SECS:-180}"
WARM_WEB_LISTEN_SECS="${WARM_WEB_LISTEN_SECS:-2}"
WARM_WEB_ROUTE_SECS="${WARM_WEB_ROUTE_SECS:-120}"
SLUG="${WARM_WEB_SLUG:-_warmup}"

paths=(
  /
  /login
  /workspaces/new
  "/${SLUG}/issues"
  "/${SLUG}/inbox"
  "/${SLUG}/agents"
  "/${SLUG}/settings"
  "/${SLUG}/issues/${SLUG}"
)

http_code() {
  local url=$1 timeout=$2 code
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time "$timeout" "$url" || true)"
  printf '%s' "$code"
}

connected() {
  local code
  code="$(http_code "$1" "$2")"
  [ -n "$code" ] && [ "$code" != 000 ]
}

ready=0
deadline=$((SECONDS + WARM_WEB_WAIT_SECS))
while [ "$SECONDS" -lt "$deadline" ]; do
  if connected "${FRONTEND_ORIGIN}/" "$WARM_WEB_LISTEN_SECS"; then
    ready=1
    break
  fi
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "warm-web: Next never listened on ${FRONTEND_ORIGIN}/" >&2
  exit 1
fi

for path in "${paths[@]}"; do
  url="${FRONTEND_ORIGIN}${path}"
  code="$(http_code "$url" "$WARM_WEB_ROUTE_SECS")"
  if [ -z "$code" ] || [ "$code" = 000 ]; then
    echo "warm-web: could not connect to ${url}" >&2
    exit 1
  fi
  echo "warm-web: ${code} ${url}"
done
