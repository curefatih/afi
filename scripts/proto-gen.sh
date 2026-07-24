#!/usr/bin/env bash
# Generate Go stubs from proto/afi/extension/v1.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PROTOC="${PROTOC:-protoc}"
command -v "$PROTOC" >/dev/null || {
  echo "ERROR: protoc not found (install protobuf / set PROTOC)" >&2
  exit 1
}
command -v protoc-gen-go >/dev/null || {
  echo "ERROR: protoc-gen-go not found — go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
  exit 1
}
command -v protoc-gen-go-grpc >/dev/null || {
  echo "ERROR: protoc-gen-go-grpc not found — go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" >&2
  exit 1
}

INCLUDE_FLAGS=()
for d in /opt/homebrew/include /usr/local/include /usr/include; do
  if [ -d "$d/google/protobuf" ]; then
    INCLUDE_FLAGS+=(--proto_path="$d")
    break
  fi
done

mkdir -p gen/proto
"$PROTOC" \
  --proto_path=proto \
  "${INCLUDE_FLAGS[@]}" \
  --go_out=gen/proto --go_opt=module=github.com/curefatih/afi/gen/proto \
  --go-grpc_out=gen/proto --go-grpc_opt=module=github.com/curefatih/afi/gen/proto \
  proto/afi/extension/v1/*.proto

echo "Generated gen/proto/afi/extension/v1/"
