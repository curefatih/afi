# A2A echo server

Minimal [Agent2Agent](https://a2a-protocol.org/) upstream for local AFI testing. It serves an Agent Card and echoes `message/send` text — no LLM calls.

## Run

```bash
go run ./examples/a2a-echo
# or: make run-a2a-echo
```

Listens on `:8091` by default.

| Env / flag | Default | Purpose |
|------------|---------|---------|
| `A2A_ECHO_ADDR` / `-addr` | `:8091` | Listen address |
| `A2A_ECHO_URL` / `-url` | `http://127.0.0.1:<port>/` | `url` in the Agent Card |
| `A2A_ECHO_API_KEY` / `-api-key` | (empty) | Optional Bearer required on card + JSON-RPC |

## Endpoints

| Method | Path | Notes |
|--------|------|--------|
| `GET` | `/.well-known/agent-card.json` | Agent Card (`skills`: `echo`) |
| `POST` | `/` | JSON-RPC (`message/send` → echo Message) |
| `GET` | `/healthz` | Liveness |

### Direct curl

```bash
curl -s http://127.0.0.1:8091/.well-known/agent-card.json | jq .

curl -s http://127.0.0.1:8091/ \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"hi"}]}}}'
```

## Register with AFI

1. Start this server and the AFI control plane + gateway.
2. In **Governance → A2A**, create an agent:
   - **Alias:** `echo`
   - **Name:** `Echo`
   - **Upstream URL:** `http://127.0.0.1:8091/`
   - **Card URL:** leave blank (defaults to `{upstream}/.well-known/agent-card.json`)
   - If you set `A2A_ECHO_API_KEY`, put the same name in **API key env** on the agent and export it on the **gateway** process (and control plane for **Test connection**).
3. Use **Playground → A2A**, pick `echo`, **Fetch agent card**, then **Send message**.

Via gateway:

```bash
export AFI_API_KEY=sk-project-local-dev-token-12345

curl -s http://localhost:8080/a2a/echo/.well-known/agent-card.json \
  -H "Authorization: Bearer $AFI_API_KEY" | jq .

curl -s http://localhost:8080/a2a/echo \
  -H "Authorization: Bearer $AFI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"text":"hi"}]}}}'
```
