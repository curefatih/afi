# Verify

With control plane and gateway running (`make run-controlplane`, `make run-gateway`) and `OPENAI_API_KEY` set:

## Health

```bash
curl -s http://localhost:8081/healthz
curl -s http://localhost:8080/healthz
```

Expect `{"status":"ok"}` (gateway may also report snapshot version).

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

Expect a JSON response with `choices[0].message.content`.

## Chat completion (stream)

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}],
    "stream": true
  }'
```

Expect SSE `data:` frames ending with `[DONE]`.

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
make snapshot-publish
```

Within a few seconds the gateway log should show a new snapshot version without restarting the process.
