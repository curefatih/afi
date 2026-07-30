#!/usr/bin/env bash
# Release platform clients whose paths changed since BASE_REF.
#
# Usage:
#   bash scripts/release-clients.sh
#   BASE_REF=origin/main bash scripts/release-clients.sh
#   CLIENTS=typescript DRY_RUN=1 bash scripts/release-clients.sh
#
# Env:
#   BASE_REF   — git ref to diff against (default: HEAD~1, or origin/main if set)
#   CLIENTS    — comma list: typescript,python,java,all (default: auto-detect from diff)
#   FORCE=1    — release even when the path did not change
#   DRY_RUN, SKIP_TESTS, SKIP_PUBLISH, COMMIT_BUMP, VERSION — passed through
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

BASE_REF="${BASE_REF:-}"
CLIENTS="${CLIENTS:-}"
FORCE="${FORCE:-0}"

if [[ -z "${BASE_REF}" ]]; then
  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    BASE_REF="HEAD~1"
  else
    BASE_REF="$(git rev-list --max-parents=0 HEAD)"
  fi
fi

path_changed() {
  local path="$1"
  git diff --name-only "${BASE_REF}" HEAD -- "${path}" | grep -q .
}

want_typescript=0
want_python=0
want_java=0

if [[ -n "${CLIENTS}" && "${CLIENTS}" != "all" ]]; then
  IFS=',' read -r -a list <<<"${CLIENTS}"
  for c in "${list[@]}"; do
    c="$(echo "${c}" | tr -d '[:space:]')"
    case "${c}" in
      typescript|ts) want_typescript=1 ;;
      python|py) want_python=1 ;;
      java) want_java=1 ;;
      *) echo "unknown CLIENTS entry: ${c}" >&2; exit 1 ;;
    esac
  done
elif [[ "${CLIENTS}" == "all" || "${FORCE}" == "1" ]]; then
  want_typescript=1
  want_python=1
  want_java=1
else
  echo "==> Detecting client changes since ${BASE_REF}"
  if path_changed "clients/typescript"; then
    want_typescript=1
    echo "    typescript: changed"
  else
    echo "    typescript: unchanged"
  fi
  if path_changed "clients/python"; then
    want_python=1
    echo "    python: changed"
  else
    echo "    python: unchanged"
  fi
  if path_changed "clients/java"; then
    want_java=1
    echo "    java: changed"
  else
    echo "    java: unchanged"
  fi
fi

if [[ "${want_typescript}" != "1" && "${want_python}" != "1" && "${want_java}" != "1" ]]; then
  echo "No client path changes — nothing to release."
  exit 0
fi

pass_env=( )
for key in DRY_RUN SKIP_TESTS SKIP_PUBLISH COMMIT_BUMP VERSION NODE_AUTH_TOKEN NPM_TOKEN \
  NPM_TRUSTED_PUBLISHING TWINE_USERNAME TWINE_PASSWORD PYPI_API_TOKEN TWINE_REPOSITORY_URL \
  GITHUB_TOKEN GITHUB_ACTOR GITHUB_REPOSITORY PYTHON MVN; do
  if [[ -n "${!key:-}" ]]; then
    pass_env+=("${key}=${!key}")
  fi
done

if [[ "${want_typescript}" == "1" ]]; then
  echo
  env "${pass_env[@]}" bash "${ROOT}/scripts/release-client-typescript.sh"
fi

if [[ "${want_python}" == "1" ]]; then
  echo
  env "${pass_env[@]}" bash "${ROOT}/scripts/release-client-python.sh"
fi

if [[ "${want_java}" == "1" ]]; then
  echo
  env "${pass_env[@]}" bash "${ROOT}/scripts/release-client-java.sh"
fi

echo
echo "Client release finished."
