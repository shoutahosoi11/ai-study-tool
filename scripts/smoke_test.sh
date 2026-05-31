#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-}"
CONFIRM_PRODUCTION_SMOKE="${CONFIRM_PRODUCTION_SMOKE:-false}"

if [[ -z "$BASE_URL" ]]; then
  echo "BASE_URL is required" >&2
  exit 2
fi

BASE_URL="${BASE_URL%/}"
LOWER_BASE_URL="$(printf '%s' "$BASE_URL" | tr '[:upper:]' '[:lower:]')"
if [[ "$LOWER_BASE_URL" == *"ai-study-tool.com"* || "$LOWER_BASE_URL" == *"run.app"* ]]; then
  if [[ "$CONFIRM_PRODUCTION_SMOKE" != "true" ]]; then
    echo "Refusing production-looking BASE_URL without CONFIRM_PRODUCTION_SMOKE=true" >&2
    exit 2
  fi
fi

curl_status() {
  local method="$1"
  local path="$2"
  shift 2
  curl -sS -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$path" "$@"
}

expect_status() {
  local name="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" != "$want" ]]; then
    echo "FAIL $name: got HTTP $got, want $want" >&2
    exit 1
  fi
  echo "ok $name"
}

health_status="$(curl_status GET /health)"
expect_status "health" "$health_status" "200"

ready_status="$(curl_status GET /ready)"
expect_status "ready" "$ready_status" "200"

unauthorized_status="$(curl_status GET /api/v1/users/me)"
expect_status "unauthorized users/me" "$unauthorized_status" "401"

pairing_status="$(curl_status POST /api/v1/extension/pairing/start -H 'Content-Type: application/json' --data '{}')"
case "$pairing_status" in
  200|201|202|400|429) echo "ok extension pairing start controlled with HTTP $pairing_status" ;;
  *)
    echo "FAIL extension pairing start: unexpected HTTP $pairing_status" >&2
    exit 1
    ;;
esac

headers="$(curl -fsSI "$BASE_URL/health")"
printf '%s\n' "$headers" | rg -qi '^x-content-type-options: *nosniff' || {
  echo "FAIL security headers: missing X-Content-Type-Options nosniff" >&2
  exit 1
}
printf '%s\n' "$headers" | rg -qi '^content-security-policy:' || {
  echo "FAIL security headers: missing Content-Security-Policy" >&2
  exit 1
}
echo "ok security headers"

echo "smoke test complete"

