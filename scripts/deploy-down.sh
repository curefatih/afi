#!/usr/bin/env bash
# Stop the deploy Compose stack (keeps volumes by default).
# Usage:
#   bash scripts/deploy-down.sh
#   bash scripts/deploy-down.sh --volumes
#   PROFILE=dataplane bash scripts/deploy-down.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=deploy-compose.sh
source "${ROOT}/scripts/deploy-compose.sh"

cd "${ROOT}"

REMOVE_VOLUMES=0
PROFILE_CLI=()
for arg in "$@"; do
  case "${arg}" in
    --volumes|-v) REMOVE_VOLUMES=1 ;;
    *) PROFILE_CLI+=("${arg}") ;;
  esac
done

# `docker compose down` stops the whole project; profiles mainly matter for messaging.
# Still resolve so operators can pass PROFILE=… consistently with deploy-up.
if [[ "${#PROFILE_CLI[@]}" -gt 0 ]]; then
  resolve_profiles "${PROFILE_CLI[@]}"
else
  resolve_profiles
fi
compose_args

if [[ "${REMOVE_VOLUMES}" -eq 1 ]]; then
  docker compose "${COMPOSE_ARGS[@]}" down -v
  echo "Stack stopped and volumes removed ($(profile_hint))."
else
  docker compose "${COMPOSE_ARGS[@]}" down
  echo "Stack stopped (volumes retained). Pass --volumes to wipe Postgres/Redis data."
fi
