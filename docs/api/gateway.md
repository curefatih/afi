# Gateway API (overlay)

Point official OpenAI / Anthropic / MCP / A2A clients at the gateway base URL. AFI does **not** re-publish full vendor schemas.

- **Base URL:** `http://localhost:8080` (local)
- **Auth:** `Authorization: Bearer <virtual-api-key>` (OpenAI-style), or `x-api-key: <virtual-api-key>` (Anthropic SDK)
- **Tags:** optional `X-AFI-Tags: key:value,key:value`
- **OpenAPI overlay:** [`api/openapi/gateway.openapi.yaml`](https://github.com/curefatih/afi/blob/main/api/openapi/gateway.openapi.yaml)

**Chat dialects** (OpenAI vs Anthropic client shape, any routed model): see **[API dialects](dialects.md)**.

## Paths

| Method | Path | Notes |
| ------ | ---- | ----- |
| GET | `/healthz` | Snapshot version + registry info |
| GET | `/openai/v1/models` | OpenAI-shaped models (alias: `/v1/models`) |
| POST | `/openai/v1/chat/completions` | OpenAI dialect chat (alias: `/v1/chat/completions`) |
| POST | `/anthropic/v1/messages` | Anthropic dialect chat (alias: `/v1/messages`) |
| POST | `/openai/v1/embeddings` | Embeddings (alias: `/v1/embeddings`) |
| POST | `/openai/v1/images/generations` | Images (alias: `/v1/images/generations`) |
| POST | `/openai/v1/audio/speech` | TTS (alias: `/v1/audio/speech`) |
| POST | `/openai/v1/audio/transcriptions` | STT (alias: `/v1/audio/transcriptions`) |
| POST/GET/DELETE | `/mcp/{alias}` | MCP Streamable HTTP proxy |
| POST | `/a2a/{alias}` | A2A JSON-RPC proxy |
| GET | `/a2a/{alias}/.well-known/agent-card.json` | Agent Card (URL rewrite) |

SDK setup and examples live on [API dialects](dialects.md).
