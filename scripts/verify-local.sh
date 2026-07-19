#!/usr/bin/env bash
# Verifies a running local stack (control plane :8081, gateway :8080).
set -euo pipefail

CP="${AFI_CONTROLPLANE_URL:-http://localhost:8081}"
GW="${AFI_GATEWAY_URL:-http://localhost:8080}"
INTERNAL_TOKEN="${AFI_INTERNAL_TOKEN:-afi-local-internal-token}"
VIRTUAL_KEY="${AFI_VIRTUAL_API_KEY:-sk-project-local-dev-token-12345}"

echo "==> health"
curl -fsS "$CP/healthz" | grep -q '"status":"ok"'
curl -fsS "$GW/healthz" | grep -q '"status":"ok"'
echo "ok"

echo "==> login"
TOKEN=$(curl -fsS "$CP/api/v1/platform/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@afi.local","password":"admin"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
test -n "$TOKEN"
echo "ok"

echo "==> bad key rejected"
code=$(curl -s -o /dev/null -w '%{http_code}' "$GW/v1/chat/completions" \
  -H "Authorization: Bearer sk-invalid" \
  -H 'Content-Type: application/json' \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"x"}]}')
test "$code" = "401"
echo "ok"

echo "==> internal publish requires token"
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$CP/internal/v1/snapshots/publish")
if [ "$code" != "401" ]; then
  echo "expected 401 without internal token, got $code" >&2
  exit 1
fi
echo "ok"

echo "==> list models"
curl -fsS "$GW/v1/models" \
  -H "Authorization: Bearer $VIRTUAL_KEY" \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("object")=="list" and isinstance(d.get("data"), list) and any(m.get("id")=="gpt-4o-mini" for m in d["data"]), d'
echo "ok"

echo "==> snapshot publish + gateway version bump"
before=$(curl -fsS "$GW/healthz" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("snapshot_version") or 0)')
curl -fsS -X POST "$CP/internal/v1/snapshots/publish" \
  -H "X-AFI-Internal-Token: $INTERNAL_TOKEN" >/dev/null
# poll up to ~10s for hot reload
after="$before"
for _ in $(seq 1 20); do
  sleep 0.5
  after=$(curl -fsS "$GW/healthz" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("snapshot_version") or 0)')
  if [ "$after" -gt "$before" ]; then
    break
  fi
done
if [ "$after" -le "$before" ]; then
  echo "gateway snapshot version did not increase (before=$before after=$after)" >&2
  exit 1
fi
echo "ok (version $before -> $after)"

if [ -n "${OPENAI_API_KEY:-}" ]; then
  echo "==> live chat completion"
  curl -fsS "$GW/v1/chat/completions" \
    -H "Authorization: Bearer $VIRTUAL_KEY" \
    -H 'Content-Type: application/json' \
    -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}' \
    | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("choices"), d'
  echo "ok"

  if curl -fsS "$CP/api/v1/platform/organizations/org_local/usage?limit=1" \
      -H "Authorization: Bearer $TOKEN" >/dev/null 2>&1; then
    echo "==> usage worker drain (optional)"
    found=0
    for _ in $(seq 1 20); do
      sleep 0.5
      count=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/usage?limit=5" \
        -H "Authorization: Bearer $TOKEN" | python3 -c 'import sys,json; print(len(json.load(sys.stdin)))')
      if [ "${count:-0}" -ge 1 ]; then
        found=1
        break
      fi
    done
    if [ "$found" = "1" ]; then
      echo "ok (usage_events populated — worker is running)"
    else
      echo "SKIPPED (no usage_events yet — start make run-worker to drain outbox)"
    fi
  fi
else
  echo "==> live chat completion SKIPPED (OPENAI_API_KEY unset)"
  echo "==> usage worker drain SKIPPED (needs live chat)"
fi

echo "==> platform me"
curl -fsS "$CP/api/v1/platform/auth/me" -H "Authorization: Bearer $TOKEN" | grep -q admin@afi.local
echo "ok"

echo "==> quota limit 0 → 429 (no OpenAI required)"
# Clean prior verify quotas for org_local if any by creating a zero request quota on org.
curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"scope_type":"organization","scope_id":"org_local","metric":"requests","limit_value":0,"window":"total"}' >/dev/null
# wait for snapshot hot reload
for _ in $(seq 1 20); do
  sleep 0.5
  code=$(curl -s -o /dev/null -w '%{http_code}' "$GW/v1/chat/completions" \
    -H "Authorization: Bearer $VIRTUAL_KEY" \
    -H 'Content-Type: application/json' \
    -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"x"}]}')
  if [ "$code" = "429" ]; then
    break
  fi
done
if [ "$code" != "429" ]; then
  echo "expected 429 after quota limit 0, got $code" >&2
  exit 1
fi
echo "ok"

echo "verify-local: all checks passed"
