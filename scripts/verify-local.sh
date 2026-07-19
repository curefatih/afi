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
  | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert d.get("object")=="list" and isinstance(d.get("data"), list), d
assert any(m.get("id")=="gpt-4o-mini" for m in d["data"]), d
assert all("supports_streaming" in m for m in d["data"]), d
assert all("supports_tts" in m and "supports_stt" in m for m in d["data"]), d
has_tts = any(m.get("id")=="tts-1" and m.get("supports_tts") for m in d["data"])
has_stt = any(m.get("id")=="whisper-1" and m.get("supports_stt") for m in d["data"])
assert has_tts and has_stt, d
assert any(m.get("id")=="echo-demo" for m in d["data"]), d
'
echo "ok"

echo "==> echo extension + demo hook (no OpenAI)"
# Clear leftover request quotas with limit 0 from prior verify runs.
while read -r qid; do
  [ -n "$qid" ] || continue
  curl -fsS -X DELETE "$CP/api/v1/platform/quotas/$qid" -H "Authorization: Bearer $TOKEN" >/dev/null || true
done < <(curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
  -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
for q in json.load(sys.stdin):
  if q.get("metric")=="requests" and int(q.get("limit_value",-1))==0:
    print(q["id"])
')
# wait for snapshot after deletes
for _ in $(seq 1 10); do
  sleep 0.3
  code=$(curl -s -o /dev/null -w '%{http_code}' "$GW/v1/chat/completions" \
    -H "Authorization: Bearer $VIRTUAL_KEY" \
    -H 'Content-Type: application/json' \
    -d '{"model":"echo-demo","messages":[{"role":"user","content":"warmup"}]}')
  if [ "$code" != "429" ]; then
    break
  fi
done
curl -fsS "$GW/healthz" | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert "echo" in d.get("provider_types",[]), d
hooks=d.get("hooks",[])
names=[]
for h in hooks:
  if isinstance(h, dict):
    names.append(h.get("name"))
    if h.get("name")=="demo_tag":
      assert h.get("before_chat") and h.get("after_chat"), h
  else:
    names.append(h)
assert "demo_tag" in names, d
'
curl -fsS "$GW/v1/chat/completions" \
  -H "Authorization: Bearer $VIRTUAL_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"model":"echo-demo","messages":[{"role":"user","content":"ping"}]}' \
  | python3 -c '
import sys,json
d=json.load(sys.stdin)
content=d["choices"][0]["message"]["content"]
assert "echo:" in content and "[hook:demo]" in content and "ping" in content, content
'
echo "ok"

if [ -n "${OPENAI_API_KEY:-}" ]; then
  echo "==> TTS speech (optional live)"
  code=$(curl -s -o /tmp/afi-verify-tts.bin -w '%{http_code}' "$GW/v1/audio/speech" \
    -H "Authorization: Bearer $VIRTUAL_KEY" \
    -H 'Content-Type: application/json' \
    -d '{"model":"tts-1","input":"afi verify","voice":"alloy"}')
  if [ "$code" = "200" ]; then
    test -s /tmp/afi-verify-tts.bin
    echo "ok"
  else
    echo "SKIPPED (speech HTTP $code — check OPENAI_API_KEY / route)"
  fi
else
  echo "==> TTS speech SKIPPED (OPENAI_API_KEY unset)"
fi

if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
  has_claude=$(curl -fsS "$GW/v1/models" -H "Authorization: Bearer $VIRTUAL_KEY" \
    | python3 -c 'import sys,json; d=json.load(sys.stdin); print(any(m.get("id")=="claude-sonnet" for m in d.get("data",[])))')
  if [ "$has_claude" = "True" ]; then
    echo "==> native /v1/messages (Anthropic)"
    curl -fsS "$GW/v1/messages" \
      -H "Authorization: Bearer $VIRTUAL_KEY" \
      -H 'Content-Type: application/json' \
      -H 'anthropic-version: 2023-06-01' \
      -d '{"model":"claude-sonnet","max_tokens":32,"messages":[{"role":"user","content":"ping"}]}' \
      | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("type")=="message" or d.get("content"), d'
    echo "ok"
  else
    echo "==> native /v1/messages SKIPPED (no claude-sonnet route in models)"
  fi
else
  echo "==> native /v1/messages SKIPPED (ANTHROPIC_API_KEY unset)"
fi

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
      curl -fsS "$CP/api/v1/platform/organizations/org_local/usage?limit=1" \
        -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
e=json.load(sys.stdin)[0]
assert "modality" in e and "metrics" in e, e
assert "key_kind" in e or e.get("api_key_id"), e
'
      curl -fsS "$CP/api/v1/platform/organizations/org_local/usage/summary?group_by=modality" \
        -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert isinstance(d, list), d
'
      echo "ok (usage_events + summary — worker is running)"
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

echo "==> provider health"
curl -fsS "$CP/api/v1/platform/organizations/org_local/providers/health" \
  -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
d=json.load(sys.stdin)
assert isinstance(d, list) and len(d)>=1, d
assert all("status" in x and "provider_id" in x for x in d), d
'
echo "ok"

echo "==> personal API key"
PERSONAL_KEY_JSON=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"verify-personal","kind":"personal"}')
echo "$PERSONAL_KEY_JSON" | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("kind")=="personal" and d.get("owner_user_id") and not d.get("project_id"), d'
curl -fsS "$CP/api/v1/platform/organizations/org_local/keys" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json; keys=json.load(sys.stdin); assert any(k.get("kind")=="personal" for k in keys), keys'
USER_ID=$(echo "$PERSONAL_KEY_JSON" | python3 -c 'import sys,json; print(json.load(sys.stdin)["owner_user_id"])')
curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"scope_type\":\"user\",\"scope_id\":\"$USER_ID\",\"metric\":\"tokens\",\"limit_value\":999999,\"window\":\"total\"}" \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); assert d.get("scope_type")=="user", d'
echo "ok"

echo "==> quota limit 0 → 429 (no OpenAI required)"
# Most-specific scope wins: pin the seed virtual key so org/project quotas cannot override.
KEY_ID=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/keys" \
  -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
keys=json.load(sys.stdin)
for k in keys:
  if k.get("key_prefix","").startswith("sk-project-local") or k.get("name")=="local-dev":
    print(k["id"]); break
else:
  print(keys[0]["id"])
')
curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"scope_type\":\"api_key\",\"scope_id\":\"$KEY_ID\",\"metric\":\"requests\",\"limit_value\":0,\"window\":\"total\"}" >/dev/null
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
# Remove the zero quota so the next verify (and local demos) are not stuck at 429.
ZERO_IDS=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
  -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
for q in json.load(sys.stdin):
  if q.get("metric")=="requests" and int(q.get("limit_value",-1))==0 and q.get("scope_id")=="'"$KEY_ID"'":
    print(q["id"])
')
for qid in $ZERO_IDS; do
  curl -fsS -X DELETE "$CP/api/v1/platform/quotas/$qid" -H "Authorization: Bearer $TOKEN" >/dev/null || true
done
echo "ok"

echo "==> CEL policy deny → 403 (no OpenAI required)"
POL_JSON=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/policies" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"verify-deny-echo","expression":"request.model != \"echo-demo\"","priority":1000}')
POL_ID=$(echo "$POL_JSON" | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')
for _ in $(seq 1 20); do
  sleep 0.5
  code=$(curl -s -o /tmp/afi-verify-pol.json -w '%{http_code}' "$GW/v1/chat/completions" \
    -H "Authorization: Bearer $VIRTUAL_KEY" \
    -H 'Content-Type: application/json' \
    -d '{"model":"echo-demo","messages":[{"role":"user","content":"x"}]}')
  if [ "$code" = "403" ]; then
    break
  fi
done
if [ "$code" != "403" ]; then
  echo "expected 403 after CEL deny policy, got $code body=$(cat /tmp/afi-verify-pol.json)" >&2
  curl -fsS -X DELETE "$CP/api/v1/platform/policies/$POL_ID" -H "Authorization: Bearer $TOKEN" >/dev/null || true
  exit 1
fi
curl -fsS -X DELETE "$CP/api/v1/platform/policies/$POL_ID" -H "Authorization: Bearer $TOKEN" >/dev/null
echo "ok"

echo "==> Redis minute rate limit → 429 (requires Redis)"
if ! redis-cli -u "${AFI_REDIS_URL:-redis://localhost:6379/0}" ping >/dev/null 2>&1; then
  echo "redis not reachable; skip minute rate-limit check" >&2
else
  curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"scope_type\":\"api_key\",\"scope_id\":\"$KEY_ID\",\"metric\":\"requests\",\"limit_value\":0,\"window\":\"minute\"}" >/dev/null
  for _ in $(seq 1 20); do
    sleep 0.5
    code=$(curl -s -o /dev/null -w '%{http_code}' "$GW/v1/chat/completions" \
      -H "Authorization: Bearer $VIRTUAL_KEY" \
      -H 'Content-Type: application/json' \
      -d '{"model":"echo-demo","messages":[{"role":"user","content":"x"}]}')
    if [ "$code" = "429" ]; then
      break
    fi
  done
  if [ "$code" != "429" ]; then
    echo "expected 429 after minute rate limit 0, got $code" >&2
    exit 1
  fi
  MIN_IDS=$(curl -fsS "$CP/api/v1/platform/organizations/org_local/quotas" \
    -H "Authorization: Bearer $TOKEN" | python3 -c '
import sys,json
for q in json.load(sys.stdin):
  if q.get("metric")=="requests" and q.get("window")=="minute" and int(q.get("limit_value",-1))==0 and q.get("scope_id")=="'"$KEY_ID"'":
    print(q["id"])
')
  for qid in $MIN_IDS; do
    curl -fsS -X DELETE "$CP/api/v1/platform/quotas/$qid" -H "Authorization: Bearer $TOKEN" >/dev/null || true
  done
  echo "ok"
fi

echo "verify-local: all checks passed"
