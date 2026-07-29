#!/usr/bin/env bash
# Build, test, and publish ai.afi:platform-client to Maven Central.
#
# Usage:
#   bash scripts/release-client-java.sh
#   DRY_RUN=1 bash scripts/release-client-java.sh
#   VERSION=1.2.3 bash scripts/release-client-java.sh
#
# Env:
#   DRY_RUN=1 / SKIP_TESTS=1 / SKIP_PUBLISH=1 / COMMIT_BUMP=1 / VERSION
#   MAVEN_USERNAME / MAVEN_PASSWORD — Sonatype Central Portal user token
#   Or set in ~/.m2/settings.xml under <server><id>central</id>…
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=semver.sh
source "${ROOT}/scripts/semver.sh"

PKG_DIR="${ROOT}/clients/java"
POM="${PKG_DIR}/pom.xml"
GROUP_PATH="ai/afi"
ARTIFACT="platform-client"
DRY_RUN="${DRY_RUN:-0}"
SKIP_TESTS="${SKIP_TESTS:-0}"
SKIP_PUBLISH="${SKIP_PUBLISH:-0}"
COMMIT_BUMP="${COMMIT_BUMP:-0}"
MVN="${MVN:-mvn}"

read_local_version() {
  # shellcheck disable=SC2016
  sed -n 's|^[[:space:]]*<version>\([^<]*\)</version>.*|\1|p' "${POM}" | head -n1
}

read_published_version() {
  local meta url
  url="https://repo1.maven.org/maven2/${GROUP_PATH}/${ARTIFACT}/maven-metadata.xml"
  meta="$(curl -fsSL "${url}" 2>/dev/null || true)"
  if [[ -z "${meta}" ]]; then
    return 0
  fi
  printf '%s\n' "${meta}" | sed -n 's|.*<latest>\([^<]*\)</latest>.*|\1|p' | head -n1
}

set_version() {
  local v="$1"
  # Update only the project version (first <version> after <artifactId>).
  python3 - <<'PY' "${POM}" "${v}"
import pathlib, re, sys
path = pathlib.Path(sys.argv[1])
version = sys.argv[2]
text = path.read_text()
# Replace the first non-parent <version> under the project coordinates.
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

cd "${PKG_DIR}"

echo "==> Java client (ai.afi:${ARTIFACT})"
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

if [[ -z "${MAVEN_PASSWORD:-}" && -z "${MAVEN_CENTRAL_TOKEN:-}" ]]; then
  echo "ERROR: set MAVEN_PASSWORD (or MAVEN_CENTRAL_TOKEN) for Sonatype Central Portal" >&2
  echo "       and MAVEN_USERNAME (token username). See clients/README.md." >&2
  exit 1
fi
export MAVEN_PASSWORD="${MAVEN_PASSWORD:-${MAVEN_CENTRAL_TOKEN}}"

echo "==> Publish ai.afi:${ARTIFACT}:${next_v}"
# Uses the 'release' profile (GPG + central-publishing-maven-plugin).
# Expects ~/.m2/settings.xml server id 'central' or -Denv credentials.
"${MVN}" -B -Prelease \
  -DskipTests \
  deploy \
  ${MAVEN_USERNAME:+-Denv.MAVEN_USERNAME="${MAVEN_USERNAME}"} \
  ${MAVEN_PASSWORD:+-Denv.MAVEN_PASSWORD="${MAVEN_PASSWORD}"}

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
