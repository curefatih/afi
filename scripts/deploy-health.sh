#!/usr/bin/env bash
# Smoke-check a deployed stack (Compose or binary). Override URLs via env.
set -euo pipefail

CP="${AFI_CONTROLPLANE_URL:-http://localhost:8081}"
GW="${AFI_GATEWAY_URL:-http://localhost:8080}"
WEB="${AFI_WEB_URL:-}"

echo "==> control plane health (${CP})"
curl -fsS "${CP}/healthz" | grep -q '"status":"ok"'
echo "ok"

echo "==> gateway health (${GW})"
curl -fsS "${GW}/healthz" | grep -q '"status":"ok"'
echo "ok"

if [[ -n "${WEB}" ]]; then
  echo "==> web (${WEB})"
  code=$(curl -s -o /dev/null -w '%{http_code}' "${WEB}/healthz" || true)
  if [[ "${code}" != "200" ]]; then
    code=$(curl -s -o /dev/null -w '%{http_code}' "${WEB}/" || true)
  fi
  test "${code}" = "200"
  echo "ok"
fi

echo "deploy-health: passed"
