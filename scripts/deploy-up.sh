#!/usr/bin/env bash
# Prepare deploy/.env + deploy/afi.yaml if missing, then build and start the stack.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT}/deploy"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
ENV_FILE="${DEPLOY_DIR}/.env"
CONFIG_FILE="${DEPLOY_DIR}/afi.yaml"

cd "${ROOT}"

if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${DEPLOY_DIR}/env.example" "${ENV_FILE}"
  echo "Created ${ENV_FILE} from env.example — edit secrets before production use."
fi

if [[ ! -f "${CONFIG_FILE}" ]]; then
  cp "${DEPLOY_DIR}/afi.example.yaml" "${CONFIG_FILE}"
  echo "Created ${CONFIG_FILE} from afi.example.yaml — edit seed/auth before first start."
fi

# Fail fast on placeholder secrets so we do not boot with known-default credentials.
if grep -q 'CHANGE_ME' "${ENV_FILE}"; then
  echo "error: ${ENV_FILE} still contains CHANGE_ME placeholders. Replace them before deploying." >&2
  exit 1
fi
if grep -q 'CHANGE_ME' "${CONFIG_FILE}"; then
  echo "error: ${CONFIG_FILE} still contains CHANGE_ME placeholders. Replace them before deploying." >&2
  exit 1
fi

echo "==> Building images"
docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" build

echo "==> Starting stack"
docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d

echo
echo "Stack is up. Useful checks:"
echo "  curl -fsS http://localhost:\${CONTROLPLANE_HOST_PORT:-8081}/healthz"
echo "  curl -fsS http://localhost:\${GATEWAY_HOST_PORT:-8080}/healthz"
echo "  open http://localhost:\${WEB_HOST_PORT:-3000}"
echo
echo "Logs: docker compose -f deploy/docker-compose.yml --env-file deploy/.env logs -f"
echo "Stop: make deploy-down"
