#!/usr/bin/env bash
# Build, test, and publish ai.afi:platform-client to GitHub Packages.
#
# Usage:
#   bash scripts/release-client-java.sh
#   DRY_RUN=1 bash scripts/release-client-java.sh
#   VERSION=1.2.3 bash scripts/release-client-java.sh
#
# Env:
#   DRY_RUN=1 / SKIP_TESTS=1 / SKIP_PUBLISH=1 / COMMIT_BUMP=1 / VERSION
#   GITHUB_TOKEN — required to publish (and to read latest published version)
#   GITHUB_ACTOR — GitHub username (default: github-actions[bot] in CI)
#   GITHUB_REPOSITORY — owner/repo (default: curefatih/afi)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=semver.sh
source "${ROOT}/scripts/semver.sh"

PKG_DIR="${ROOT}/clients/java"
POM="${PKG_DIR}/pom.xml"
ARTIFACT="platform-client"
# GitHub Packages Maven package name is groupId.artifactId
GH_PACKAGE="ai.afi.platform-client"
GITHUB_REPOSITORY="${GITHUB_REPOSITORY:-curefatih/afi}"
GH_OWNER="${GITHUB_REPOSITORY%%/*}"
DRY_RUN="${DRY_RUN:-0}"
SKIP_TESTS="${SKIP_TESTS:-0}"
SKIP_PUBLISH="${SKIP_PUBLISH:-0}"
COMMIT_BUMP="${COMMIT_BUMP:-0}"
MVN="${MVN:-mvn}"

read_local_version() {
  sed -n 's|^[[:space:]]*<version>\([^<]*\)</version>.*|\1|p' "${POM}" | head -n1
}

read_published_version() {
  local token="${GITHUB_TOKEN:-}"
  local url
  url="https://api.github.com/users/${GH_OWNER}/packages/maven/${GH_PACKAGE}/versions?per_page=1"
  if [[ -z "${token}" ]]; then
    # Anonymous metadata fetch often fails for GitHub Packages; treat as unpublished.
    return 0
  fi
  python3 - <<'PY' "${url}" "${token}"
import json, sys, urllib.request, urllib.error
url, token = sys.argv[1], sys.argv[2]
req = urllib.request.Request(
    url,
    headers={
        "Accept": "application/vnd.github+json",
        "Authorization": f"Bearer {token}",
        "X-GitHub-Api-Version": "2022-11-28",
    },
)
try:
    with urllib.request.urlopen(req, timeout=30) as resp:
        data = json.load(resp)
except urllib.error.HTTPError as e:
    if e.code in (404, 401, 403):
        sys.exit(0)
    raise
if not data:
    sys.exit(0)
# Prefer non-deleted versions; API returns newest first.
for item in data:
    name = item.get("name")
    if name and not item.get("deleted_at"):
        print(name)
        break
PY
}

set_version() {
  local v="$1"
  python3 - <<'PY' "${POM}" "${v}"
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
version = sys.argv[2]
text = path.read_text()
new, n = re.subn(
    r"(<artifactId>platform-client</artifactId>\s*\n\s*<version>)[^<]+(</version>)",
    rf"\g<1>{version}\g<2>",
    text,
    count=1,
)
if n != 1:
    raise SystemExit("failed to update version in pom.xml")
path.write_text(new)
PY
}

write_settings() {
  local settings="$1"
  local user="${GITHUB_ACTOR:-github-actions[bot]}"
  local pass="${GITHUB_TOKEN:?GITHUB_TOKEN required for GitHub Packages}"
  cat >"${settings}" <<EOF
<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0 https://maven.apache.org/xsd/settings-1.2.0.xsd">
  <servers>
    <server>
      <id>github</id>
      <username>${user}</username>
      <password>${pass}</password>
    </server>
  </servers>
</settings>
EOF
}

cd "${PKG_DIR}"

echo "==> Java client (ai.afi:${ARTIFACT}) → GitHub Packages"
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

if [[ "${SKIP_TESTS}" != "1" ]]; then
  echo "==> Test"
  "${MVN}" -B -q test
else
  echo "==> Package (skip tests)"
  "${MVN}" -B -q -DskipTests package
fi

echo "==> Package"
"${MVN}" -B -q -DskipTests package

if [[ "${DRY_RUN}" == "1" || "${SKIP_PUBLISH}" == "1" ]]; then
  echo "==> Dry run / skip publish"
  ls -la target/*.jar
  echo "Done (not published)."
  exit 0
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "ERROR: set GITHUB_TOKEN to publish to GitHub Packages" >&2
  exit 1
fi

SETTINGS="$(mktemp)"
trap 'rm -f "${SETTINGS}"' EXIT
write_settings "${SETTINGS}"

echo "==> Publish ai.afi:${ARTIFACT}:${next_v} to GitHub Packages"
"${MVN}" -B -q -s "${SETTINGS}" -DskipTests deploy

if [[ "${COMMIT_BUMP}" == "1" ]]; then
  echo "==> Commit version bump / tag"
  git -C "${ROOT}" add "${POM}"
  if ! git -C "${ROOT}" diff --cached --quiet; then
    git -C "${ROOT}" commit -m "chore(clients): bump java to ${next_v} [skip release]"
  fi
  tag="clients-java-v${next_v}"
  if git -C "${ROOT}" rev-parse "${tag}" >/dev/null 2>&1; then
    echo "    tag ${tag} already exists"
  else
    git -C "${ROOT}" tag -a "${tag}" -m "Release ai.afi:${ARTIFACT}:${next_v}"
    echo "    tagged ${tag}"
  fi
fi

echo "Done. Published ai.afi:${ARTIFACT}:${next_v}"
echo "    https://github.com/${GITHUB_REPOSITORY}/packages"
