#!/usr/bin/env bash
# Prepare deploy/.env + deploy/afi.yaml if missing, then build and start selected profiles.
# Usage:
#   bash scripts/deploy-up.sh
#   bash scripts/deploy-up.sh controlplane
#   bash scripts/deploy-up.sh dataplane worker
#   PROFILE=web bash scripts/deploy-up.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=deploy-compose.sh
source "${ROOT}/scripts/deploy-compose.sh"

cd "${ROOT}"

resolve_profiles "$@"
require_deploy_config
compose_args

echo "==> Building images ($(profile_hint))"
docker compose "${COMPOSE_ARGS[@]}" build

echo "==> Starting stack ($(profile_hint))"
docker compose "${COMPOSE_ARGS[@]}" up -d

echo
echo "Stack is up ($(profile_hint)). Useful checks:"
case " ${PROFILES[*]} " in
  *" full "*|*" controlplane "*)
    echo "  curl -fsS http://localhost:\${CONTROLPLANE_HOST_PORT:-8081}/healthz"
    ;;
esac
case " ${PROFILES[*]} " in
  *" full "*|*" dataplane "*)
    echo "  curl -fsS http://localhost:\${GATEWAY_HOST_PORT:-8080}/healthz"
    ;;
esac
case " ${PROFILES[*]} " in
  *" full "*|*" web "*)
    echo "  open http://localhost:\${WEB_HOST_PORT:-3000}"
    ;;
esac
echo
echo "Logs: make deploy-logs PROFILE='${PROFILES[*]}'"
echo "Stop: make deploy-down"
