#!/usr/bin/env bash
# Prove a running API is on native auth: Clerk overlay off.
# Fail if /api/config advertises a publishable key or if native
# send-code / verify-code are hidden (404).
#
# Usage: scripts/check-clerk-overlay-off.sh [api-base]
# Default api-base is http://127.0.0.1:8080
set -euo pipefail

API="${1:-http://127.0.0.1:8080}"

cfg="$(curl -fsS "${API}/api/config")"
if printf '%s' "$cfg" | grep -q '"clerk_publishable_key"'; then
	printf 'FAIL: /api/config still advertises clerk_publishable_key\n' >&2
	exit 1
fi

probe() {
	local path="$1"
	local body="$2"
	local tmp
	tmp="$(mktemp)"
	local code
	code="$(curl -sS -o "$tmp" -w '%{http_code}' \
		-X POST "${API}${path}" \
		-H 'Content-Type: application/json' \
		-d "$body")"
	rm -f "$tmp"
	printf '%s' "$code"
}

send_code="$(probe /auth/send-code '{"email":"dev@localhost"}')"
if [ "$send_code" = "404" ]; then
	printf 'FAIL: POST /auth/send-code returned 404 (native auth hidden)\n' >&2
	exit 1
fi
if [ "$send_code" != "200" ] && [ "$send_code" != "429" ]; then
	printf 'FAIL: POST /auth/send-code returned %s (want 200 or 429)\n' "$send_code" >&2
	exit 1
fi

verify_code="$(probe /auth/verify-code '{"email":"dev@localhost","code":"000000"}')"
if [ "$verify_code" = "404" ]; then
	printf 'FAIL: POST /auth/verify-code returned 404 (native auth hidden)\n' >&2
	exit 1
fi
if [ "$verify_code" = "401" ] || [ "$verify_code" = "400" ]; then
	:
elif [ "$verify_code" != "200" ]; then
	printf 'FAIL: POST /auth/verify-code returned %s (want a native handler status)\n' "$verify_code" >&2
	exit 1
fi

printf 'OK: overlay off (send-code %s, verify-code %s)\n' "$send_code" "$verify_code"
