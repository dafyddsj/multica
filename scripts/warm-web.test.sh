#!/usr/bin/env bash
# Stub-driven tests for scripts/warm-web.sh. Never hits a real Next server.
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

fake_bin="$tmp_dir/bin"
mkdir -p "$fake_bin"

write_curl_stub() {
  cat >"$fake_bin/curl" <<'EOF'
#!/usr/bin/env bash
url=""
for arg in "$@"; do
  case "$arg" in
    http*) url="$arg" ;;
  esac
done
printf '%s\n' "$url" >>"${WARM_WEB_CURL_LOG}"

count=0
if [ -f "${WARM_WEB_CURL_COUNT}" ]; then
  count="$(cat "${WARM_WEB_CURL_COUNT}")"
fi
count=$((count + 1))
printf '%s\n' "$count" >"${WARM_WEB_CURL_COUNT}"

if [ -n "${WARM_WEB_FAIL_UNTIL:-}" ] && [ "$count" -le "${WARM_WEB_FAIL_UNTIL}" ]; then
  printf '000'
  exit 0
fi

if [ "${WARM_WEB_ALWAYS_CODE:-}" = 000 ]; then
  printf '000'
  exit 0
fi

if [ -n "${WARM_WEB_ROUTE_000:-}" ] && [ "$url" = "${WARM_WEB_ROUTE_000}" ]; then
  printf '000'
  exit 0
fi

printf '%s' "${WARM_WEB_HTTP_CODE:-200}"
EOF
  chmod +x "$fake_bin/curl"
}

write_curl_stub

run_warm() {
  PATH="$fake_bin:/usr/bin:/bin" \
    FRONTEND_PORT="${FRONTEND_PORT:-3000}" \
    FRONTEND_ORIGIN="${FRONTEND_ORIGIN:-http://127.0.0.1:3000}" \
    WARM_WEB_SLUG="${WARM_WEB_SLUG:-_warmup}" \
    WARM_WEB_CURL_LOG="$tmp_dir/curl.log" \
    WARM_WEB_CURL_COUNT="$tmp_dir/curl.count" \
    bash "$root_dir/scripts/warm-web.sh"
}

reset_stub_state() {
  : >"$tmp_dir/curl.log"
  printf '0\n' >"$tmp_dir/curl.count"
}

expected_paths=(
  /
  /login
  /workspaces/new
  /_warmup/issues
  /_warmup/inbox
  /_warmup/agents
  /_warmup/settings
  /_warmup/issues/_warmup
)

# ---------------------------------------------------------------------------
# Wait loop retries until the stub reports a listener, then warms every path.
# ---------------------------------------------------------------------------
reset_stub_state
out="$tmp_dir/retry.out"
err="$tmp_dir/retry.err"
status=0
WARM_WEB_FAIL_UNTIL=2 WARM_WEB_WAIT_SECS=10 WARM_WEB_ROUTE_SECS=2 \
  run_warm >"$out" 2>"$err" || status=$?
[ "$status" -eq 0 ] || fail "retry-until-listen exited $status, want 0"

for path in "${expected_paths[@]}"; do
  url="http://127.0.0.1:3000${path}"
  if ! grep -Fq "$url" "$tmp_dir/curl.log"; then
    fail "did not GET $url"
  fi
  if ! grep -Fq "warm-web: 200 $url" "$out"; then
    fail "missing success log for $url"
  fi
done
attempts="$(cat "$tmp_dir/curl.count")"
[ "$attempts" -gt 2 ] || fail "wait loop did not retry (attempts=$attempts)"

# ---------------------------------------------------------------------------
# Exits non-zero when the listener never appears.
# ---------------------------------------------------------------------------
reset_stub_state
status=0
WARM_WEB_ALWAYS_CODE=000 WARM_WEB_WAIT_SECS=1 WARM_WEB_ROUTE_SECS=1 \
  run_warm >"$tmp_dir/never.out" 2>"$tmp_dir/never.err" || status=$?
[ "$status" -ne 0 ] || fail "listener-never-appears exited 0"

# ---------------------------------------------------------------------------
# HTTP 404 is success: webpack compiled the module.
# ---------------------------------------------------------------------------
reset_stub_state
status=0
WARM_WEB_HTTP_CODE=404 WARM_WEB_WAIT_SECS=5 WARM_WEB_ROUTE_SECS=2 \
  run_warm >"$tmp_dir/404.out" 2>"$tmp_dir/404.err" || status=$?
[ "$status" -eq 0 ] || fail "HTTP 404 warmup exited $status, want 0"
if ! grep -Fq "warm-web: 404 http://127.0.0.1:3000/login" "$tmp_dir/404.out"; then
  fail "404 was not logged as success"
fi

# ---------------------------------------------------------------------------
# A route that returns curl 000 after the listener is up fails the script.
# ---------------------------------------------------------------------------
reset_stub_state
status=0
WARM_WEB_ROUTE_000="http://127.0.0.1:3000/login" WARM_WEB_WAIT_SECS=5 WARM_WEB_ROUTE_SECS=2 \
  run_warm >"$tmp_dir/miss.out" 2>"$tmp_dir/miss.err" || status=$?
[ "$status" -ne 0 ] || fail "route 000 after listen exited 0"

echo "✓ warm-web.sh behaviour verified"
