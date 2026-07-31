#!/usr/bin/env bash
# Bootstrap local-only deploy/.env + deploy/afi.yaml and start the full Docker Compose stack.
# See docs/guides/quick-start.md
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
CONFIG_FILE="${DEPLOY_DIR}/afi.yaml"
QS_ENV="${DEPLOY_DIR}/quickstart.env"
QS_CONFIG="${DEPLOY_DIR}/quickstart.afi.yaml"

cd "${ROOT}"

install_file() {
  local src="$1"
  local dest="$2"
  local label="$3"

  if [[ ! -f "${dest}" ]]; then
    cp "${src}" "${dest}"
    echo "Created ${dest} from ${label}."
    return
  fi

  if grep -q 'CHANGE_ME' "${dest}"; then
    cp "${src}" "${dest}"
    echo "Replaced ${dest} (still had CHANGE_ME placeholders) with ${label}."
    return
  fi

  echo "Keeping existing ${dest}."
}

install_file "${QS_ENV}" "${ENV_FILE}" "quickstart.env"
install_file "${QS_CONFIG}" "${CONFIG_FILE}" "quickstart.afi.yaml"

if [[ -z "${OPENAI_API_KEY:-}" ]] && ! grep -qE '^OPENAI_API_KEY=.+' "${ENV_FILE}"; then
  echo
  echo "Note: OPENAI_API_KEY is empty. The stack will start, but chat inference needs a key."
  echo "      Add it to deploy/.env or export OPENAI_API_KEY before re-running."
  echo
fi

# If the caller exported provider keys, merge them into deploy/.env for Compose.
merge_env_key() {
  local key="$1"
  local value="${!key:-}"
  [[ -n "${value}" ]] || return 0
  if grep -qE "^${key}=" "${ENV_FILE}"; then
    # Portable in-place replace without relying on GNU sed -i.
    local tmp
    tmp="$(mktemp)"
    awk -v k="${key}" -v v="${value}" '
      BEGIN { done=0 }
      index($0, k "=") == 1 && !done { print k "=" v; done=1; next }
      { print }
      END { if (!done) print k "=" v }
    ' "${ENV_FILE}" > "${tmp}"
    mv "${tmp}" "${ENV_FILE}"
  else
    printf '\n%s=%s\n' "${key}" "${value}" >> "${ENV_FILE}"
  fi
  echo "Wrote ${key} into ${ENV_FILE}."
}

merge_env_key OPENAI_API_KEY
merge_env_key ANTHROPIC_API_KEY
merge_env_key GEMINI_API_KEY

echo "==> Starting AFI via Docker Compose (first build may take several minutes)"
bash "${ROOT}/scripts/deploy-up.sh"

echo
echo "Quick start is ready:"
echo "  Web UI:         http://localhost:3000  (admin@afi.local / admin)"
echo "  Control plane:  http://localhost:8081/healthz"
echo "  Gateway:        http://localhost:8080/healthz"
echo "  Virtual API key: sk-project-local-dev-token-12345"
echo
echo "Health: make deploy-health"
echo "Logs:   make deploy-logs"
echo "Stop:   make deploy-down"
