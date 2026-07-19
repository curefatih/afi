#!/usr/bin/env bash
# Stop the deploy Compose stack (keeps volumes by default).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT}/deploy"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
ENV_FILE="${DEPLOY_DIR}/.env"

cd "${ROOT}"

ARGS=(-f "${COMPOSE_FILE}")
if [[ -f "${ENV_FILE}" ]]; then
  ARGS+=(--env-file "${ENV_FILE}")
fi

if [[ "${1:-}" == "--volumes" ]] || [[ "${1:-}" == "-v" ]]; then
  docker compose "${ARGS[@]}" down -v
  echo "Stack stopped and volumes removed."
else
  docker compose "${ARGS[@]}" down
  echo "Stack stopped (volumes retained). Pass --volumes to wipe Postgres/Redis data."
fi
