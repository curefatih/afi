#!/usr/bin/env bash
# Poll until GET <url> returns JSON with "status":"ok".
# Usage: wait-for-health.sh <health-url> [timeout_seconds]
set -euo pipefail

URL="${1:?usage: wait-for-health.sh <health-url> [timeout_seconds]}"
TIMEOUT="${2:-60}"
INTERVAL="${WAIT_INTERVAL:-0.25}"

echo "Waiting for ${URL} (timeout ${TIMEOUT}s)..."
deadline=$((SECONDS + TIMEOUT))
while (( SECONDS < deadline )); do
  if curl -fsS --max-time 2 "${URL}" 2>/dev/null | grep -q '"status":"ok"'; then
    echo "ok"
    exit 0
  fi
  sleep "${INTERVAL}"
done

echo "timed out waiting for healthy response from ${URL}" >&2
exit 1
