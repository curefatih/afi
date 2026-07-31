#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${DIR}"

ENV_ARGS=()
[[ -f .env ]] && ENV_ARGS+=(--env-file .env)
[[ -f regional.secrets.env ]] && ENV_ARGS+=(--env-file regional.secrets.env)

docker compose "${ENV_ARGS[@]}" -f docker-compose.yml --profile hub --profile regional down "$@"
