# Verify

Prefer the automated path once the stack is up:

```bash
make verify
```

## Health

```bash
curl -s http://localhost:8081/healthz
curl -s http://localhost:8080/healthz
```

Expect `{"status":"ok"}` (gateway also reports `snapshot_version`).

## Chat completion (non-stream)

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

Expect a JSON response with `choices[0].message.content`. Requires `OPENAI_API_KEY`.

## List models

```bash
curl -s http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-project-local-dev-token-12345"
```

Expect `object: "list"` with route model ids (at least `gpt-4o-mini` after seed).

## Anthropic route (optional)

Seed includes `prov_anthropic`. Create a route (or use Routing UI), then:

```bash
export ANTHROPIC_API_KEY="sk-ant-..."

curl -s http://localhost:8081/api/v1/platform/organizations/org_local/routes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet","provider_id":"prov_anthropic","target_model":"claude-sonnet-4-20250514"}'

curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet","messages":[{"role":"user","content":"ping"}]}'

# streaming (OpenAI-shaped SSE)
curl -sN http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet","stream":true,"messages":[{"role":"user","content":"ping"}]}'

# native Anthropic Messages (same auth/quota/route)
curl -s http://localhost:8080/v1/messages \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet","max_tokens":64,"messages":[{"role":"user","content":"ping"}]}'
```

## OpenAI-compatible (Ollama, etc.)

Seed includes `prov_ollama` (`type=openai_compatible`). With Ollama running locally:

```bash
export OLLAMA_API_KEY=ollama

curl -s http://localhost:8081/api/v1/platform/organizations/org_local/routes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3","provider_id":"prov_ollama","target_model":"llama3.2"}'

curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"llama3","messages":[{"role":"user","content":"ping"}]}'
```

See [Providers](../development/providers.md).

## Gemini route (optional)

```bash
export GEMINI_API_KEY="..."

curl -s http://localhost:8081/api/v1/platform/organizations/org_local/routes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"gemini-flash","provider_id":"prov_gemini","target_model":"gemini-2.0-flash"}'

curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"gemini-flash","messages":[{"role":"user","content":"ping"}]}'

# streaming (OpenAI-shaped SSE via streamGenerateContent)
curl -sN http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"gemini-flash","stream":true,"messages":[{"role":"user","content":"ping"}]}'
```

Failover: set `"fallbacks":[{"provider_id":"prov_anthropic","target_model":"claude-sonnet-4-20250514"}]` on an OpenAI primary route to fail over on 5xx/timeout/429. Optional same-target `"retry":{"max_attempts":3,"backoff":{"strategy":"fixed","base_delay":"100ms"}}` retries the primary that many times (with backoff) before walking fallbacks.

## TTS / STT (OpenAI)

Seed includes routes `tts-1` and `whisper-1` → `prov_openai`.

```bash
export OPENAI_API_KEY="sk-..."

curl -s http://localhost:8080/v1/audio/speech \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"tts-1","input":"Hello from AFI","voice":"alloy"}' \
  --output /tmp/afi-tts.mp3

curl -s http://localhost:8080/v1/audio/transcriptions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -F model=whisper-1 \
  -F file=@/tmp/afi-tts.mp3
```

## Editable route (UI or API)

Create an alias route that maps `ping-model` → `gpt-4o-mini`, then call it without restarting the gateway.

```bash
TOKEN=$(curl -s http://localhost:8081/api/v1/platform/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@afi.local","password":"admin"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

# list providers to get provider_id (seed uses prov_openai)
curl -s http://localhost:8081/api/v1/platform/organizations/org_local/providers \
  -H "Authorization: Bearer $TOKEN"

curl -s http://localhost:8081/api/v1/platform/organizations/org_local/routes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"model":"ping-model","provider_id":"prov_openai","target_model":"gpt-4o-mini"}'

# wait briefly for hot reload, then:
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"ping-model","messages":[{"role":"user","content":"ping"}]}'
```

Or use **Routing** in the web UI (`pnpm --dir web dev`).

## Auth rejection

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-invalid" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"x"}]}'
```

Expect `401`.

## Platform login

```bash
curl -s http://localhost:8081/api/v1/platform/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@afi.local","password":"admin"}'
```

Expect `{"token":"..."}`.

## Snapshot hot reload

```bash
# CLI (in-process, no internal header)
make snapshot-publish

# or HTTP
curl -s -X POST http://localhost:8081/internal/v1/snapshots/publish \
  -H "X-AFI-Internal-Token: afi-local-internal-token"
```

Gateway `snapshot_version` increases without process restart.

## Quota enforcement (429)

Create a lifetime `requests` limit of `0` for the org (no OpenAI needed):

```bash
TOKEN=$(curl -s http://localhost:8081/api/v1/platform/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@afi.local","password":"admin"}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

curl -s http://localhost:8081/api/v1/platform/organizations/org_local/quotas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scope_type":"organization","scope_id":"org_local","metric":"requests","limit_value":0,"window":"total"}'

# after hot reload:
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"x"}]}'
```

Expect `429`.
