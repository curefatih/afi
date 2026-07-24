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
| POST | `/v1/embeddings` | OpenAI-compatible embeddings |
| POST | `/v1/images/generations` | OpenAI-compatible images (optional org object-store persist) |
| POST | `/v1/audio/speech` | TTS |
| POST | `/v1/audio/transcriptions` | STT |
| POST/GET/DELETE | `/mcp/{alias}` | MCP Streamable HTTP proxy |
| POST | `/a2a/{alias}` | A2A JSON-RPC proxy |
| GET | `/a2a/{alias}/.well-known/agent-card.json` | Agent Card (URL rewrite) |

SDK setup and examples live on [API dialects](dialects.md).
