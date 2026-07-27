#!/usr/bin/env bash
# Build release binaries and optionally package them with an example config.
#
# Usage:
#   bash scripts/build-release.sh
#   GOOS=linux GOARCH=arm64 bash scripts/build-release.sh
#   PACKAGE=1 bash scripts/build-release.sh          # write tarball + checksum
#   TARGETS="linux/amd64,linux/arm64,darwin/amd64,darwin/arm64" PACKAGE=1 \
#     bash scripts/build-release.sh
#
# Env:
#   GOOS / GOARCH  — single-target build (ignored when TARGETS is set)
#   TARGETS        — comma-separated goos/goarch list
#   VERSION        — embedded in CLI + archive name (default: git describe)
#   PACKAGE        — if 1, write afi-<version>-<os>-<arch>.tar.gz + .sha256
#   OUT_DIR        — binary output root (default: bin/release)
#   DIST_DIR       — package output dir (default: dist)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${ROOT}/bin/release}"
DIST_DIR="${DIST_DIR:-${ROOT}/dist}"
VERSION="${VERSION:-$(git -C "${ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"
PACKAGE="${PACKAGE:-0}"
# Strip leading v from archive names for consistency with common Go releases.
VERSION_SLUG="${VERSION#v}"

SERVICES=(controlplane gateway worker cli)

build_one() {
  local goos="$1"
  local goarch="$2"
  local dest="${OUT_DIR}/${goos}-${goarch}"

  mkdir -p "${dest}"
  echo "Building AFI ${VERSION} for ${goos}/${goarch} → ${dest}"

  local svc out ldflags
  for svc in "${SERVICES[@]}"; do
    out="${svc}"
    ldflags="-s -w"
    if [[ "${svc}" == "cli" ]]; then
      out="afi"
      ldflags="-s -w -X main.version=${VERSION}"
    fi
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build \
      -trimpath \
      -ldflags="${ldflags}" \
      -o "${dest}/${out}" \
      "./cmd/${svc}"
    echo "  ${dest}/${out}"
  done

  if [[ "${PACKAGE}" == "1" ]]; then
    package_one "${goos}" "${goarch}" "${dest}"
  fi
}

package_one() {
  local goos="$1"
  local goarch="$2"
  local dest="$3"
  local name="afi-${VERSION_SLUG}-${goos}-${goarch}"
  local stage="${DIST_DIR}/.stage/${name}"
  local archive="${DIST_DIR}/${name}.tar.gz"

  mkdir -p "${DIST_DIR}" "${stage}"
  rm -rf "${stage}"
  mkdir -p "${stage}"

  cp "${dest}/controlplane" "${dest}/gateway" "${dest}/worker" "${dest}/afi" "${stage}/"
  cp "${ROOT}/deploy/afi.example.yaml" "${stage}/afi.example.yaml"
  cat >"${stage}/README.txt" <<EOF
AFI ${VERSION} (${goos}/${goarch})

Contents:
  controlplane, gateway, worker, afi  — Go binaries (CGO disabled)
  afi.example.yaml                    — starting config (copy + edit)

Minimal run (Postgres required; Redis if you use timed quotas):

  cp afi.example.yaml afi.yaml
  # edit afi.yaml — replace every CHANGE_ME value
  export AFI_CONFIG=\$(pwd)/afi.yaml
  export AFI_DATABASE_URL='postgres://...'
  export AFI_JWT_SECRET='...'
  export AFI_INTERNAL_TOKEN='...'

  ./controlplane &
  OPENAI_API_KEY=... ./gateway &
  ./worker &

Docs: https://afi.fatihcure.com/deployment/binary/
EOF

  tar -C "${DIST_DIR}/.stage" -czf "${archive}" "${name}"
  rm -rf "${stage}"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "${DIST_DIR}" && sha256sum "$(basename "${archive}")" >"$(basename "${archive}").sha256")
  else
    (cd "${DIST_DIR}" && shasum -a 256 "$(basename "${archive}")" >"$(basename "${archive}").sha256")
  fi

  echo "  packaged ${archive}"
  echo "  checksum ${archive}.sha256"
}

cd "${ROOT}"

if [[ -n "${TARGETS:-}" ]]; then
  IFS=',' read -r -a target_list <<<"${TARGETS}"
  for target in "${target_list[@]}"; do
    target="$(echo "${target}" | tr -d '[:space:]')"
    [[ -n "${target}" ]] || continue
    goos="${target%%/*}"
    goarch="${target##*/}"
    if [[ "${goos}" == "${goarch}" || -z "${goos}" || -z "${goarch}" ]]; then
      echo "invalid TARGETS entry (want goos/goarch): ${target}" >&2
      exit 1
    fi
    build_one "${goos}" "${goarch}"
  done
else
  build_one "${GOOS:-linux}" "${GOARCH:-amd64}"
fi

rm -rf "${DIST_DIR}/.stage"

echo "Done."
if [[ "${PACKAGE}" != "1" ]]; then
  echo "Copy binaries and a config file (see deploy/afi.example.yaml) to the target host."
  echo "Set PACKAGE=1 to write dist/*.tar.gz archives that include the example config."
fi
echo "See docs/deployment/binary.md"
