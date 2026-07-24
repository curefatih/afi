# API dialects

Point your **existing** OpenAI, Anthropic, or Gemini client at AFI. The path you call selects the **client wire format** (the dialect). Routing selects **which upstream model** serves the request. Those two choices are independent.

```mermaid
flowchart LR
  sdk[OpenAI, Anthropic, or Gemini SDK]
  dialect[Dialect path]
  route[Routed model]
  up[Any chat provider]

  sdk --> dialect
  dialect -->|wire shape| route
  route --> up
```

| You want to use… | Call AFI at… | Auth |
| ---------------- | ------------ | ---- |
| OpenAI SDK / `chat.completions` | `/openai/v1/...` (or `/v1/...`) | `Authorization: Bearer <virtual-key>` |
| Anthropic SDK / Messages | `/anthropic/v1/...` (or `/v1/messages`) | `x-api-key: <virtual-key>` or Bearer |
| Gemini SDK / `generate_content` | `/gemini/v1beta/models/{route}:...` | `x-goog-api-key`, `?key=`, or Bearer |

The virtual API key is always an **AFI** key from the platform — not the upstream vendor key. Upstream credentials stay on the provider config (BYOK / env / vault).

## Why dialects matter

Without dialects, gateways usually force one client shape (almost always OpenAI). Teams stuck on Anthropic SDKs or tooling cannot reuse the same gateway routes for Gemini or OpenAI backends.

With dialects:

* Keep your preferred SDK and prompts
* Route `"model"` to OpenAI, Anthropic, Gemini, or `openai_compatible` (Ollama, etc.)
* Still get AFI auth, quotas, policies, failover, usage, and hooks

## Supported today

| Capability | Status |
| ---------- | ------ |
| OpenAI chat dialect (`/openai/v1/chat/completions`, alias `/v1/...`) | Supported |
| Anthropic messages dialect (`/anthropic/v1/messages`, alias `/v1/messages`) | Supported |
| Gemini generateContent dialect (`/gemini/v1beta/models/{route}:...`) | Supported |
| Cross-provider routing (any chat-capable upstream behind either dialect) | Supported |
| Streaming (when provider advertises `stream`) | Supported |
| Auth: Bearer and `x-api-key` | Supported |
| Text chat: messages, system, max tokens, temperature, top_p, stop | Supported |
| Tools / function calling: `tools`, `tool_choice`, tool calls, tool results | Supported |
| Multimodal input (vision): image URL and inline base64 parts | Supported |
| Streaming tool calls (deltas translated between dialects) | Supported |
| Embeddings / TTS / STT / images under `/openai/v1/*` (+ `/v1/*` aliases) | Supported |

Tools and images round-trip across dialects: an OpenAI-SDK client can send `tools`/`image_url` to a route backed by Anthropic (and vice versa), and tool calls come back in the caller's dialect. `/v1/models` advertises `supports_tools` / `supports_vision` for chat-capable models accordingly. Features the gateway cannot represent yet return a clear **`400`** rather than silently dropping them.

## Paths

| Dialect | Canonical | Alias |
| ------- | --------- | ----- |
| OpenAI chat | `POST /openai/v1/chat/completions` | `POST /v1/chat/completions` |
| OpenAI models | `GET /openai/v1/models` | `GET /v1/models` |
| OpenAI embeddings | `POST /openai/v1/embeddings` | `POST /v1/embeddings` |
| OpenAI images | `POST /openai/v1/images/generations` | `POST /v1/images/generations` |
| OpenAI TTS | `POST /openai/v1/audio/speech` | `POST /v1/audio/speech` |
| OpenAI STT | `POST /openai/v1/audio/transcriptions` | `POST /v1/audio/transcriptions` |
| Anthropic messages | `POST /anthropic/v1/messages` | `POST /v1/messages` |
| Gemini generate | `POST /gemini/v1beta/models/{route}:generateContent` | — |
| Gemini stream | `POST /gemini/v1beta/models/{route}:streamGenerateContent` | — |

Full path map: [Gateway API](gateway.md).

## OpenAI SDK

```bash
export OPENAI_BASE_URL=http://localhost:8080/openai/v1
# alias also works:
# export OPENAI_BASE_URL=http://localhost:8080/v1
export OPENAI_API_KEY=sk-your-afi-virtual-key
```

```python
from openai import OpenAI

client = OpenAI()  # reads OPENAI_BASE_URL + OPENAI_API_KEY
resp = client.chat.completions.create(
    model="gpt-4o-mini",  # AFI route name
    messages=[{"role": "user", "content": "ping"}],
)
print(resp.choices[0].message.content)
```

## Anthropic SDK

Set the base URL to the gateway **host + `/anthropic`**. The SDK appends `/v1/messages` and sends `x-api-key`.

```bash
export ANTHROPIC_BASE_URL=http://localhost:8080/anthropic
export ANTHROPIC_API_KEY=sk-your-afi-virtual-key
```

```python
import anthropic

client = anthropic.Anthropic()
msg = client.messages.create(
    model="gpt-4o-mini",  # still an AFI route — can target OpenAI upstream
    max_tokens=1024,
    messages=[{"role": "user", "content": "Hello"}],
)
print(msg.content[0].text)
```

Curl equivalent:

```bash
curl -s http://localhost:8080/anthropic/v1/messages \
  -H "content-type: application/json" \
  -H "x-api-key: sk-your-afi-virtual-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "gpt-4o-mini",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## Gemini SDK

Set the SDK base URL to the gateway host plus `/gemini`. The model argument is
an **AFI route alias**, even when that route targets OpenAI or Anthropic.

```python
from google import genai
from google.genai import types

client = genai.Client(
    api_key="sk-your-afi-virtual-key",
    http_options=types.HttpOptions(base_url="http://localhost:8080/gemini"),
)
response = client.models.generate_content(
    model="gpt-4o-mini",
    contents="Hello",
)
print(response.text)
```

Curl equivalent:

```bash
curl -s "http://localhost:8080/gemini/v1beta/models/gpt-4o-mini:generateContent" \
  -H "content-type: application/json" \
  -H "x-goog-api-key: sk-your-afi-virtual-key" \
  -d '{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}'
```

## Model names are routes

The model is the **AFI route alias** configured in the platform (Routing), not necessarily the upstream vendor id. OpenAI and Anthropic carry it in `"model"`; Gemini carries it in the `{route}` URL segment. Map `gpt-4o-mini` → OpenAI, `claude-sonnet` → Anthropic, or use either through Gemini-shaped `generateContent` — the client dialect stays unchanged while the provider follows the route.

## Tools and vision

Tools (function calling) and image input work in either dialect and are translated to the routed provider. Send them exactly as your SDK does:

```bash
# OpenAI-shaped tools request, routed to whatever backs "gpt-4o-mini"
curl -s http://localhost:8080/openai/v1/chat/completions \
  -H "content-type: application/json" \
  -H "authorization: Bearer sk-your-afi-virtual-key" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "weather in SF?"}],
    "tools": [{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}],
    "tool_choice": "auto"
  }'
```

The model's tool call comes back in the caller's dialect (`tool_calls` for OpenAI, `tool_use` blocks for Anthropic, `functionCall` parts for Gemini), including when streaming. Feed tool results back as `role: "tool"` messages (OpenAI), `tool_result` blocks (Anthropic), or `functionResponse` parts (Gemini). Images are sent as `image_url`, `image`, or `inlineData` / `fileData` parts; inline base64 and hosted/file URIs round-trip.

## Streaming

All three dialects support streaming when the routed provider advertises `stream` capability. AFI translates stream events — including tool calls — so the client sees OpenAI chunks, Anthropic message events, or Gemini SSE candidates.

## Known limitations

Chat dialects cover text, tools/function calling, and image input (vision) — streaming included. Errors from the gateway and upstream providers are rewritten into the **client dialect** envelope, so SDKs do not see a foreign vendor body. Features the internal representation does not model yet receive a clear **`400`** (not silently dropped): Anthropic `thinking`/`document` blocks, OpenAI `logprobs`, `n>1`, OpenAI `functions`/`function_call`, and Gemini `candidateCount > 1`, `cachedContent`, `safetySettings`, structured-output / thinking generation config, or non-function tools. Gemini function-call IDs are synthesized when the wire response omits one, allowing tool results to round-trip through OpenAI/Anthropic providers.

## Related

* [Gateway overlay](gateway.md) — full path and auth reference
* [Providers](../development/providers.md) — adapter capabilities
* [Verify](../getting-started/verify.md) — local smoke checks
