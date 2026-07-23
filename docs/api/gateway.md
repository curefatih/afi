# Gateway API (overlay)

Point official OpenAI / Anthropic / MCP / A2A clients at the gateway base URL. AFI does **not** re-publish full vendor schemas.

- **Base URL:** `http://localhost:8080` (local)
- **Auth:** `Authorization: Bearer <virtual-api-key>`
- **Tags:** optional `X-AFI-Tags: key:value,key:value`
- **OpenAPI overlay:** [`api/openapi/gateway.openapi.yaml`](https://github.com/curefatih/afi/blob/main/api/openapi/gateway.openapi.yaml)

## Paths

| Method | Path | Notes |
| ------ | ---- | ----- |
| GET | `/healthz` | Snapshot version + registry info |
| GET | `/v1/models` | OpenAI-shaped, route/catalog enriched |
| POST | `/v1/chat/completions` | OpenAI-compatible |
| POST | `/v1/embeddings` | OpenAI-compatible |
| POST | `/v1/audio/speech` | TTS |
| POST | `/v1/audio/transcriptions` | STT |
| POST | `/v1/messages` | Native Anthropic Messages |
| POST/GET/DELETE | `/mcp/{alias}` | MCP Streamable HTTP proxy |
| POST | `/a2a/{alias}` | A2A JSON-RPC proxy |
| GET | `/a2a/{alias}/.well-known/agent-card.json` | Agent Card (URL rewrite) |

## Example (OpenAI SDK)

```bash
export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-afi-virtual-key
```

Use the OpenAI (or Anthropic) SDK as usual; AFI handles routing, quotas, policies, and usage.
