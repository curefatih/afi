#!/usr/bin/env bash
# Build release binaries (linux/amd64 by default) into bin/release/.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${ROOT}/bin/release"
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
VERSION="${VERSION:-$(git -C "${ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"

mkdir -p "${OUT_DIR}"
cd "${ROOT}"

echo "Building AFI ${VERSION} for ${GOOS}/${GOARCH} → ${OUT_DIR}"

for svc in controlplane gateway worker cli; do
  out="${svc}"
  if [[ "${svc}" == "cli" ]]; then
    out="afi"
  fi
  CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
    -trimpath \
    -ldflags="-s -w" \
    -o "${OUT_DIR}/${out}" \
    "./cmd/${svc}"
  echo "  ${OUT_DIR}/${out}"
done

echo "Done. Copy ${OUT_DIR}/* and a config file to the target host."
echo "See docs/deployment/binary.md"
