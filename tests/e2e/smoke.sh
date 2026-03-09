#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

check() {
  local endpoint="$1"
  local key="$2"
  local body
  body="$(curl -fsS "$BASE_URL$endpoint")"
  echo "$body" | grep -q "$key" || {
    echo "Expected key '$key' in $endpoint response, got: $body"
    exit 1
  }
}

check "/health" '"success":true'
check "/live" '"status":"live"'
check "/ready" '"status":"ready"'
check "/api/metrics/latest" '"cpu_usage_percent"'
check "/api/system/info" '"hostname"'
check "/api/alerts/current" '"triggered"'

echo "Smoke test passed"
