#!/usr/bin/env bash
# Build, test, and publish @afi-ai/platform-client to npm.
#
# Usage:
#   bash scripts/release-client-typescript.sh
#   DRY_RUN=1 bash scripts/release-client-typescript.sh
#   VERSION=1.2.3 bash scripts/release-client-typescript.sh   # force version
#
# Env:
#   DRY_RUN=1       — bump/build/test but do not publish or commit
#   SKIP_TESTS=1    — skip npm test
#   SKIP_PUBLISH=1  — build only (implies no npm publish)
#   COMMIT_BUMP=1   — commit package.json version changes (CI)
#   VERSION         — explicit version to publish (skips auto-bump)
#   NODE_AUTH_TOKEN — npm token fallback (granular + Bypass 2FA). Optional when
#                     GitHub Actions OIDC trusted publishing is configured.
#   NPM_TRUSTED_PUBLISHING=1 — prefer OIDC; unset NODE_AUTH_TOKEN for publish
#                              so a bad classic token cannot force E403.
set -euo pipefail


ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=semver.sh
source "${ROOT}/scripts/semver.sh"

PKG_DIR="${ROOT}/clients/typescript"
PKG_JSON="${PKG_DIR}/package.json"
PKG_NAME="$(node -p "require('${PKG_JSON}').name")"
DRY_RUN="${DRY_RUN:-0}"
SKIP_TESTS="${SKIP_TESTS:-0}"
SKIP_PUBLISH="${SKIP_PUBLISH:-0}"
COMMIT_BUMP="${COMMIT_BUMP:-0}"

read_local_version() {
  node -p "require('${PKG_JSON}').version"
}

read_published_version() {
  npm view "${PKG_NAME}" version 2>/dev/null || true
}

set_version() {
  local v="$1"
  node -e "
    const fs = require('fs');
    const p = process.argv[1];
    const v = process.argv[2];
    const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
    pkg.version = v;
    fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
  " "${PKG_JSON}" "${v}"
}

cd "${PKG_DIR}"

echo "==> TypeScript client (${PKG_NAME})"
local_v="$(read_local_version)"
published_v="$(read_published_version)"
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

echo "==> Install"
npm install --no-fund --no-audit

if [[ "${SKIP_TESTS}" != "1" ]]; then
  echo "==> Test"
  npm test
fi

echo "==> Build"
npm run build

if [[ "${DRY_RUN}" == "1" || "${SKIP_PUBLISH}" == "1" ]]; then
  echo "==> Dry run / skip publish — packing only"
  npm pack --dry-run
  echo "Done (not published)."
  exit 0
fi

# GitHub Actions exposes ACTIONS_ID_TOKEN_REQUEST_URL when id-token: write is set.
oidc_ready=0
if [[ -n "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" && -n "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}" ]]; then
  oidc_ready=1
fi

# Prefer OIDC whenever GitHub can mint an ID token. A leftover NODE_AUTH_TOKEN
# (from setup-node / secrets) forces token auth and yields E403/E404 under the
# GAT bypass-2FA restrictions — scrub it before publish.
if [[ "${oidc_ready}" == "1" && "${NPM_FORCE_TOKEN:-0}" != "1" ]]; then
  unset NODE_AUTH_TOKEN NPM_TOKEN || true
  # setup-node writes //registry…:_authToken=${NODE_AUTH_TOKEN}; remove so npm
  # does not try token auth with an empty/stale value.
  for rc in "${PKG_DIR}/.npmrc" "${HOME}/.npmrc"; do
    if [[ -f "${rc}" ]] && grep -q '_authToken' "${rc}"; then
      # Drop only auth lines; keep registry= etc.
      tmp="$(mktemp)"
      grep -v '_authToken' "${rc}" >"${tmp}" && mv "${tmp}" "${rc}"
    fi
  done
  echo "==> Publish ${PKG_NAME}@${next_v} (OIDC trusted publishing)"
  echo "    Ensure npm Trusted Publisher is set for this package:"
  echo "    https://www.npmjs.com/package/${PKG_NAME}/access"
  echo "    → Trusted Publisher → GitHub Actions → workflow release-clients.yml"
elif [[ -n "${NODE_AUTH_TOKEN:-}" || -n "${NPM_TOKEN:-}" ]]; then
  export NODE_AUTH_TOKEN="${NODE_AUTH_TOKEN:-${NPM_TOKEN}}"
  echo "==> Publish ${PKG_NAME}@${next_v} (token)"
else
  echo "ERROR: set NODE_AUTH_TOKEN (granular access token with Bypass 2FA)," >&2
  echo "       or configure npm Trusted Publisher + id-token: write for OIDC." >&2
  exit 1
fi

npm_major="$(npm --version | cut -d. -f1)"
npm_minor="$(npm --version | cut -d. -f2)"
if [[ "${oidc_ready}" == "1" && ( "${npm_major}" -lt 11 || ( "${npm_major}" -eq 11 && "${npm_minor}" -lt 5 ) ) ]]; then
  echo "ERROR: npm $(npm --version) is too old for trusted publishing (need >= 11.5.1)" >&2
  exit 1
fi

if ! npm publish --access public; then
  echo >&2
  echo "ERROR: npm publish failed." >&2
  echo "If you saw E404 with OIDC: the package exists but Trusted Publisher is" >&2
  echo "missing or mismatches (repo curefatih/afi, workflow release-clients.yml)." >&2
  echo "Configure it at: https://www.npmjs.com/package/${PKG_NAME}/access" >&2
  exit 1
fi

if [[ "${COMMIT_BUMP}" == "1" ]]; then
  echo "==> Commit version bump / tag"
  git -C "${ROOT}" add "${PKG_JSON}"
  if ! git -C "${ROOT}" diff --cached --quiet; then
    git -C "${ROOT}" commit -m "chore(clients): bump typescript to ${next_v}"
  fi
  tag="clients-typescript-v${next_v}"
  if git -C "${ROOT}" rev-parse "${tag}" >/dev/null 2>&1; then
    echo "    tag ${tag} already exists"
  else
    git -C "${ROOT}" tag -a "${tag}" -m "Release ${PKG_NAME}@${next_v}"
    echo "    tagged ${tag}"
  fi
fi

echo "Done. Published ${PKG_NAME}@${next_v}"
