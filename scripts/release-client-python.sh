#!/usr/bin/env bash
# Build, test, and publish afi-platform to PyPI.
#
# Usage:
#   bash scripts/release-client-python.sh
#   DRY_RUN=1 bash scripts/release-client-python.sh
#   VERSION=1.2.3 bash scripts/release-client-python.sh   # force version
#
# Env:
#   DRY_RUN=1          — bump/build/test but do not upload or commit
#   SKIP_TESTS=1       — skip pytest
#   SKIP_PUBLISH=1     — build only (implies no twine upload)
#   COMMIT_BUMP=1      — commit pyproject.toml version changes (CI)
#   VERSION            — explicit version to publish (skips auto-bump)
#   TWINE_USERNAME     — usually __token__
#   TWINE_PASSWORD     — PyPI API token (or set PYPI_API_TOKEN)
#   TWINE_REPOSITORY_URL — optional (e.g. TestPyPI)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=semver.sh
source "${ROOT}/scripts/semver.sh"

PKG_DIR="${ROOT}/clients/python"
PYPROJECT="${PKG_DIR}/pyproject.toml"
PKG_NAME="afi-platform"
VENV_DIR="${PKG_DIR}/.venv"
DRY_RUN="${DRY_RUN:-0}"
SKIP_TESTS="${SKIP_TESTS:-0}"
SKIP_PUBLISH="${SKIP_PUBLISH:-0}"
COMMIT_BUMP="${COMMIT_BUMP:-0}"
PYTHON="${PYTHON:-python3}"

read_local_version() {
  "${PYTHON}" - <<'PY' "${PYPROJECT}"
import pathlib, re, sys
text = pathlib.Path(sys.argv[1]).read_text()
m = re.search(r'(?m)^version\s*=\s*"([^"]+)"', text)
if not m:
    raise SystemExit("version not found in pyproject.toml")
print(m.group(1))
PY
}

read_published_version() {
  "${PYTHON}" - <<'PY' "${PKG_NAME}"
import json, sys, urllib.error, urllib.request
name = sys.argv[1]
url = f"https://pypi.org/pypi/{name}/json"
try:
    with urllib.request.urlopen(url, timeout=30) as resp:
        data = json.load(resp)
    print(data["info"]["version"])
except urllib.error.HTTPError as e:
    if e.code == 404:
        sys.exit(0)
    raise
PY
}

set_version() {
  local v="$1"
  "${PYTHON}" - <<'PY' "${PYPROJECT}" "${v}"
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
version = sys.argv[2]
text = path.read_text()
new, n = re.subn(r'(?m)^version\s*=\s*"[^"]+"', f'version = "{version}"', text, count=1)
if n != 1:
    raise SystemExit("failed to update version in pyproject.toml")
path.write_text(new)
PY
}

cd "${PKG_DIR}"

echo "==> Python client (${PKG_NAME})"
local_v="$(read_local_version)"
published_v="$(read_published_version || true)"
if [[ -n "${VERSION:-}" ]]; then
  next_v="${VERSION}"
else
  next_v="$(semver_next_publish "${local_v}" "${published_v}")"
fi
semver_is_valid "${next_v}" || { echo "invalid VERSION=${next_v}" >&2; exit 1; }

echo "    local=${local_v} published=${published_v:-<none>} → release=${next_v}"

if [[ -n "${published_v}" && "$(semver_cmp "${next_v}" "${published_v}")" != "1" ]]; then
  echo "ERROR: release version ${next_v} is not greater than published ${published_v}" >&2
  exit 1
fi

if [[ "${next_v}" != "${local_v}" ]]; then
  echo "==> Bumping version ${local_v} → ${next_v}"
  set_version "${next_v}"
fi

echo "==> Create venv"
"${PYTHON}" -m venv "${VENV_DIR}"
# shellcheck disable=SC1091
source "${VENV_DIR}/bin/activate"
python -m pip install -q -U pip
python -m pip install -q -e ".[dev]" build twine

if [[ "${SKIP_TESTS}" != "1" ]]; then
  echo "==> Test"
  python -m pytest -q
fi

echo "==> Build"
rm -rf dist
python -m build

if [[ "${DRY_RUN}" == "1" || "${SKIP_PUBLISH}" == "1" ]]; then
  echo "==> Dry run / skip publish — checking distributions"
  python -m twine check dist/*
  echo "Done (not published)."
  exit 0
fi

if [[ -z "${TWINE_PASSWORD:-}" && -n "${PYPI_API_TOKEN:-}" ]]; then
  export TWINE_PASSWORD="${PYPI_API_TOKEN}"
fi
export TWINE_USERNAME="${TWINE_USERNAME:-__token__}"
if [[ -z "${TWINE_PASSWORD:-}" ]]; then
  echo "ERROR: set TWINE_PASSWORD or PYPI_API_TOKEN to publish" >&2
  exit 1
fi

echo "==> Publish ${PKG_NAME}==${next_v}"
python -m twine upload --non-interactive dist/*

if [[ "${COMMIT_BUMP}" == "1" ]]; then
  echo "==> Commit version bump / tag"
  git -C "${ROOT}" add "${PYPROJECT}"
  if ! git -C "${ROOT}" diff --cached --quiet; then
    git -C "${ROOT}" commit -m "chore(clients): bump python to ${next_v}"
  fi
  tag="clients-python-v${next_v}"
  if git -C "${ROOT}" rev-parse "${tag}" >/dev/null 2>&1; then
    echo "    tag ${tag} already exists"
  else
    git -C "${ROOT}" tag -a "${tag}" -m "Release ${PKG_NAME}==${next_v}"
    echo "    tagged ${tag}"
  fi
fi

echo "Done. Published ${PKG_NAME}==${next_v}"
