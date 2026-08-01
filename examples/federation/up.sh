#!/usr/bin/env bash
# Bring up the federation lab: hub → bootstrap → regional (+ restart hub gateway with tokens).
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${DIR}"

if [[ ! -f .env ]]; then
  cp env.example .env
  echo "Created .env from env.example"
fi

COMPOSE=(docker compose --env-file .env -f docker-compose.yml)

echo "==> Building images (first run can take a few minutes) ..."
"${COMPOSE[@]}" --profile hub --profile regional build

echo "==> Starting hub stack ..."
"${COMPOSE[@]}" --profile hub up -d

./bootstrap.sh

# shellcheck disable=SC1091
set -a && source .env && source regional.secrets.env && set +a

echo "==> Starting regional stack + refreshing hub gateway with deployment tokens ..."
"${COMPOSE[@]}" --env-file .env --env-file regional.secrets.env --profile hub --profile regional up -d

echo
echo "Federation lab is up."
echo
echo "  Hub UI / API:        http://localhost:${WEB_HOST_PORT:-3000}  (API http://localhost:${HUB_CONTROLPLANE_HOST_PORT:-8081})"
echo "  Hub gateway:         http://localhost:${HUB_GATEWAY_HOST_PORT:-8080}"
echo "  Regional UI:         http://localhost:${REGIONAL_WEB_HOST_PORT:-3100}  (API http://localhost:${REGIONAL_CONTROLPLANE_HOST_PORT:-8181})"
echo "  Regional gateway:    http://localhost:${REGIONAL_GATEWAY_HOST_PORT:-8180}"
echo
echo "  Hub admin:           ${HUB_ADMIN_EMAIL:-admin@afi.local} / ${HUB_ADMIN_PASSWORD:-admin}"
echo "  Hub virtual API key: sk-federation-lab-hub-key"
echo
echo "  Login to the hub UI → Regions / Federation to inspect peers and deployments."
echo "  After ~${AFI_FEDERATION_PULL_INTERVAL:-10s}, regional CP has pulled memberships + snapshot."
echo
echo "Smoke test (hub key against both gateways):"
echo "  curl -s http://localhost:8080/healthz"
echo "  curl -s http://localhost:8180/healthz"
echo "  curl -s http://localhost:8080/v1/models -H 'Authorization: Bearer sk-federation-lab-hub-key' | head"
echo "  curl -s http://localhost:8180/v1/models -H 'Authorization: Bearer sk-federation-lab-hub-key' | head"
