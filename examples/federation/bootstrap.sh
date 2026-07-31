#!/usr/bin/env bash
# Bootstrap hub regions, org bindings, federation peer, and gateway deployments.
# Writes regional.secrets.env for the regional Compose profile.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${DIR}"

if [[ -f .env ]]; then
  # shellcheck disable=SC1091
  set -a && source .env && set +a
fi

HUB_BASE_URL="${HUB_BASE_URL:-http://localhost:8081}"
HUB_ADMIN_EMAIL="${HUB_ADMIN_EMAIL:-admin@afi.local}"
HUB_ADMIN_PASSWORD="${HUB_ADMIN_PASSWORD:-admin}"
HUB_REGION_SLUG="${HUB_REGION_SLUG:-us-east}"
REGIONAL_REGION_SLUG="${REGIONAL_REGION_SLUG:-eu-west}"
SECRETS_FILE="${DIR}/regional.secrets.env"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: required command not found: $1" >&2
    exit 1
  }
}
need_cmd curl
need_cmd python3

wait_hub() {
  echo "Waiting for hub control plane at ${HUB_BASE_URL}/healthz ..."
  local i
  for i in $(seq 1 90); do
    if curl -sf "${HUB_BASE_URL}/healthz" >/dev/null; then
      echo "Hub is up."
      return 0
    fi
    sleep 2
  done
  echo "error: hub control plane did not become healthy" >&2
  exit 1
}

api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local args=(-sS -X "${method}" "${HUB_BASE_URL}${path}"
    -H "Authorization: Bearer ${TOKEN}"
    -H "Content-Type: application/json")
  if [[ -n "${body}" ]]; then
    args+=(-d "${body}")
  fi
  curl "${args[@]}"
}

json_get() {
  python3 -c 'import json,sys; d=json.load(sys.stdin); print(d'"$1"')'
}

find_region_id() {
  local slug="$1"
  api GET /api/v1/platform/regions | python3 -c "
import json,sys
slug=sys.argv[1]
for r in json.load(sys.stdin):
  if r.get('slug')==slug:
    print(r['id']); break
" "${slug}"
}

ensure_region() {
  local slug="$1"
  local name="$2"
  local id
  id="$(find_region_id "${slug}" || true)"
  if [[ -n "${id}" ]]; then
    echo "Region ${slug} exists (${id})" >&2
    printf '%s' "${id}"
    return
  fi
  id="$(api POST /api/v1/platform/regions "{\"slug\":\"${slug}\",\"name\":\"${name}\"}" | json_get "['id']")"
  echo "Created region ${slug} (${id})" >&2
  printf '%s' "${id}"
}

wait_hub

echo "Logging in as ${HUB_ADMIN_EMAIL} ..."
TOKEN="$(curl -sS -X POST "${HUB_BASE_URL}/api/v1/platform/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${HUB_ADMIN_EMAIL}\",\"password\":\"${HUB_ADMIN_PASSWORD}\"}" | json_get "['token']")"
if [[ -z "${TOKEN}" || "${TOKEN}" == "None" ]]; then
  echo "error: login failed (is the hub seeded?)" >&2
  exit 1
fi

HUB_REGION_ID="$(ensure_region "${HUB_REGION_SLUG}" "US East (hub)")"
EU_REGION_ID="$(ensure_region "${REGIONAL_REGION_SLUG}" "EU West (regional CP)")"

echo "Binding all organizations to both regions ..."
api POST "/api/v1/platform/regions/${HUB_REGION_ID}/organizations/bind-all" >/dev/null
api POST "/api/v1/platform/regions/${EU_REGION_ID}/organizations/bind-all" >/dev/null

echo "Ensuring federation peer for ${REGIONAL_REGION_SLUG} ..."
EXISTING_PEER="$(api GET /api/v1/platform/federation/peers | python3 -c "
import json,sys
rid=sys.argv[1]
peers=json.load(sys.stdin) or []
for p in peers:
  if p.get('region_id')==rid and p.get('status')!='disabled':
    print(p['id']); break
" "${EU_REGION_ID}" || true)"

JOIN_TOKEN=""
PEER_ID=""
if [[ -n "${EXISTING_PEER}" ]]; then
  PEER_ID="${EXISTING_PEER}"
  echo "Peer already exists (${PEER_ID}); rotating join token ..."
  RESP="$(api POST "/api/v1/platform/federation/peers/${PEER_ID}/rotate-join-token")"
  JOIN_TOKEN="$(echo "${RESP}" | json_get "['join_token']")"
else
  RESP="$(api POST /api/v1/platform/federation/peers \
    "{\"name\":\"EU regional CP\",\"region_id\":\"${EU_REGION_ID}\",\"base_url\":\"http://regional-controlplane:8081\"}")"
  PEER_ID="$(echo "${RESP}" | json_get "['peer']['id']")"
  JOIN_TOKEN="$(echo "${RESP}" | json_get "['join_token']")"
  echo "Registered peer ${PEER_ID}"
fi

register_deployment() {
  local region_id="$1"
  local name="$2"
  local public_url="$3"
  local existing
  existing="$(api GET "/api/v1/platform/regions/${region_id}/deployments" | python3 -c "
import json,sys
name=sys.argv[1]
for d in json.load(sys.stdin) or []:
  if d.get('name')==name:
    print(d['id']); break
" "${name}" || true)"
  if [[ -n "${existing}" ]]; then
    echo "Deployment ${name} exists (${existing}); rotating join token ..." >&2
    RESP="$(api POST "/api/v1/platform/regions/${region_id}/deployments/${existing}/rotate-join-token")"
    echo "${RESP}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['deployment']['id']); print(d['join_token'])"
    return
  fi
  RESP="$(api POST "/api/v1/platform/regions/${region_id}/deployments" \
    "{\"name\":\"${name}\",\"public_base_url\":\"${public_url}\"}")"
  echo "Registered deployment ${name}" >&2
  echo "${RESP}" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['deployment']['id']); print(d['join_token'])"
}

echo "Registering hub gateway deployment ..."
HUB_DEP_OUT="$(register_deployment "${HUB_REGION_ID}" "hub-gateway-us" "http://localhost:8080")"
HUB_DEPLOYMENT_ID="$(echo "${HUB_DEP_OUT}" | sed -n '1p')"
HUB_DEPLOYMENT_JOIN_TOKEN="$(echo "${HUB_DEP_OUT}" | sed -n '2p')"

echo "Registering regional gateway deployment (heartbeats to hub) ..."
REG_DEP_OUT="$(register_deployment "${EU_REGION_ID}" "regional-gateway-eu" "http://localhost:8180")"
REGIONAL_DEPLOYMENT_ID="$(echo "${REG_DEP_OUT}" | sed -n '1p')"
REGIONAL_DEPLOYMENT_JOIN_TOKEN="$(echo "${REG_DEP_OUT}" | sed -n '2p')"

# Publish snapshots so export embeds a compiled regional blob.
echo "Publishing hub snapshots ..."
curl -sS -X POST "${HUB_BASE_URL}/internal/v1/snapshots/publish" \
  -H "X-AFI-Internal-Token: ${AFI_INTERNAL_TOKEN:-afi-federation-internal-token}" >/dev/null || true

cat > "${SECRETS_FILE}" <<EOF
# Generated by bootstrap.sh — do not commit.
AFI_FEDERATION_JOIN_TOKEN=${JOIN_TOKEN}
HUB_DEPLOYMENT_ID=${HUB_DEPLOYMENT_ID}
HUB_DEPLOYMENT_JOIN_TOKEN=${HUB_DEPLOYMENT_JOIN_TOKEN}
REGIONAL_DEPLOYMENT_ID=${REGIONAL_DEPLOYMENT_ID}
REGIONAL_DEPLOYMENT_JOIN_TOKEN=${REGIONAL_DEPLOYMENT_JOIN_TOKEN}
HUB_REGION_ID=${HUB_REGION_ID}
EU_REGION_ID=${EU_REGION_ID}
FEDERATION_PEER_ID=${PEER_ID}
EOF

echo
echo "Wrote ${SECRETS_FILE}"
echo "  federation peer:  ${PEER_ID}"
echo "  hub deployment:   ${HUB_DEPLOYMENT_ID}"
echo "  eu deployment:    ${REGIONAL_DEPLOYMENT_ID}"
echo
echo "Next: start regional services with ./up.sh (or compose --profile regional)."
