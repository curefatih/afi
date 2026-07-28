#!/usr/bin/env bash
# Shared helpers for deploy Compose profile commands.
# shellcheck shell=bash

DEPLOY_DIR="${ROOT}/deploy"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.yml"
ENV_FILE="${DEPLOY_DIR}/.env"
CONFIG_FILE="${DEPLOY_DIR}/afi.yaml"

# Resolve profile list from args, PROFILE/DEPLOY_PROFILE env, or default "full".
# Usage: resolve_profiles "$@"   → sets PROFILES (bash array) and PROFILE_ARGS (--profile …)
resolve_profiles() {
  local raw=""
  if [[ "$#" -gt 0 ]]; then
    raw="$*"
  elif [[ -n "${PROFILE:-}" ]]; then
    raw="${PROFILE}"
  elif [[ -n "${DEPLOY_PROFILE:-}" ]]; then
    raw="${DEPLOY_PROFILE}"
  else
    raw="full"
  fi

  # shellcheck disable=SC2206
  PROFILES=(${raw//,/ })
  if [[ "${#PROFILES[@]}" -eq 0 ]]; then
    PROFILES=(full)
  fi

  PROFILE_ARGS=()
  local normalized=()
  local p
  for p in "${PROFILES[@]}"; do
    [[ -n "${p}" ]] || continue
    case "${p}" in
      full|infra|controlplane|dataplane|worker|web) ;;
      gateway)
        # Alias: dataplane is the gateway (data plane).
        p=dataplane
        ;;
      *)
        echo "error: unknown deploy profile '${p}'" >&2
        echo "valid: full infra controlplane dataplane worker web" >&2
        exit 1
        ;;
    esac
    normalized+=("${p}")
    PROFILE_ARGS+=(--profile "${p}")
  done
  PROFILES=("${normalized[@]}")
}

compose_args() {
  COMPOSE_ARGS=(-f "${COMPOSE_FILE}")
  if [[ -f "${ENV_FILE}" ]]; then
    COMPOSE_ARGS+=(--env-file "${ENV_FILE}")
  fi
  COMPOSE_ARGS+=("${PROFILE_ARGS[@]}")
}

require_deploy_config() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${DEPLOY_DIR}/env.example" "${ENV_FILE}"
    echo "Created ${ENV_FILE} from env.example — edit secrets before production use."
  fi
  if [[ ! -f "${CONFIG_FILE}" ]]; then
    cp "${DEPLOY_DIR}/afi.example.yaml" "${CONFIG_FILE}"
    echo "Created ${CONFIG_FILE} from afi.example.yaml — edit seed/auth before first start."
  fi

  if grep -q 'CHANGE_ME' "${ENV_FILE}"; then
    echo "error: ${ENV_FILE} still contains CHANGE_ME placeholders. Replace them before deploying." >&2
    exit 1
  fi
  if grep -q 'CHANGE_ME' "${CONFIG_FILE}"; then
    echo "error: ${CONFIG_FILE} still contains CHANGE_ME placeholders. Replace them before deploying." >&2
    exit 1
  fi
}

profile_hint() {
  local joined
  joined="$(IFS=,; echo "${PROFILES[*]}")"
  echo "profiles: ${joined}"
}
