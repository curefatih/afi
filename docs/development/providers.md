# Provider adapters

Gateway chat dispatch uses a **registry** of in-process adapters. The pipeline looks up `provider.type` and calls `Chat` — it does not hard-code vendor branches.

## Built-in types

| Type | Chat | Stream | TTS | STT | Notes |
|------|------|--------|-----|-----|-------|
| `openai` | yes | yes | yes | yes | Chat + `/audio/speech` + `/audio/transcriptions` |
| `anthropic` | yes | yes | no | no | Messages API → OpenAI-shaped responses/SSE |
| `gemini` | yes | yes | no | no | `generateContent` / `streamGenerateContent` → OpenAI JSON/SSE |
| `openai_compatible` | yes | yes | yes | yes | Same wire protocol as OpenAI (incl. audio if upstream supports it) |
| `echo` | yes | no | no | no | **SDK extension** (`extensions/echo`) — no network; echoes last user message |

Capabilities (`chat`, `stream`, `tts`, `stt`) are stored on the provider in the snapshot (defaults applied per type when empty). Streaming/TTS/STT requests against unsupported providers return `400`.

## Modality ports (chat / messages / audio)

| Surface | Registry port | Resolved by |
|---------|---------------|-------------|
| `POST /v1/chat/completions` | `ChatProvider.Chat` | `provider.type` |
| `POST /v1/messages` | `MessagesBackend` (via `AnthropicTransportProvider`) | routed `provider.type` |
| `POST /v1/audio/speech` / `transcriptions` | `AudioBackend` (via `OpenAITransportProvider`) | routed `provider.type` |

Chat stays on `ChatProvider`. TTS/STT and native Anthropic messages use **optional** transport interfaces implemented by the same adapters — they are **not** methods on `ChatProvider`, so SDK chat extensions need no audio stubs.

Adapters that do not implement the transport provider interface simply cannot serve that modality (handlers return `400` / `502` as today).

## Adding a provider (in-tree)

1. Implement `dataplane.ChatProvider` (`Type`, `Capabilities`, `Chat`).
2. Register it in `dataplane.DefaultRegistry()` / `RegistryFromClients` (or your gateway bootstrap).
   Outbound HTTP clients live in `internal/adapters/llm` and resolve keys via `adapters/secrets`.
3. Prefer reusing [`internal/dataplane/openaichat`](../../internal/dataplane/openaichat) helpers for OpenAI-shaped JSON/SSE.
4. Document the type, default `api_key_env`, and capabilities in this page.
5. Optionally seed an inactive provider (no route) like `prov_ollama`.

You should **not** need to edit `callProvider` or add a new `switch` case in the pipeline.

## Adding an extension (SDK)

1. Implement [`sdk/provider.ChatProvider`](../../sdk/provider) in a package under [`extensions/`](../../extensions/) (or an external module).
2. In [`cmd/gateway`](../../cmd/gateway), call `reg.RegisterSDK(your.New())` after `DefaultRegistry()`.
3. Create a control-plane provider with matching `type` and a route (seed does this for `echo` → model `echo-demo`).
4. Restart the gateway so the adapter is registered.

Working example: [`extensions/echo`](../../extensions/echo) — verify with:

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H 'Content-Type: application/json' \
  -d '{"model":"echo-demo","messages":[{"role":"user","content":"ping"}]}'
```

Expect assistant content containing `echo:` (and `[hook:demo]` if the demo BeforeChat hook is registered).

## Hooks (in-process)

`BeforeCall` / `AfterCall` run on all modalities; `ChatHook.BeforeChat` / `AfterChatHook.AfterChat` remain for chat body mutation. Register via `dataplane.NewHookChain().RegisterHook(...)` / `RegisterBeforeCall` (see `extensions/demohook`). Gateway `/healthz` lists hook objects with `before_call` / `after_call` / `before_chat` / `after_chat`. `extensions/tagquota` is an example-only BeforeCall sample for per-tag limits (not registered by default). Full gRPC/WASM plugin runtimes remain future work.

## Example: local Ollama

1. Create provider type `openai_compatible`, base URL `http://127.0.0.1:11434/v1`, env `OLLAMA_API_KEY` (any non-empty value if Ollama ignores auth).
2. Add a route, e.g. requested model `llama3` → target `llama3.2`.
3. Call the gateway with `"model":"llama3"`.
