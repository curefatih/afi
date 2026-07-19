# Provider adapters

Gateway chat dispatch uses a **registry** of in-process adapters. The pipeline looks up `provider.type` and calls `Chat` — it does not hard-code vendor branches.

## Built-in types

| Type | Chat | Stream | Notes |
|------|------|--------|-------|
| `openai` | yes | yes | OpenAI `/v1/chat/completions` |
| `anthropic` | yes | yes | Messages API → OpenAI-shaped responses/SSE |
| `gemini` | yes | yes | `generateContent` / `streamGenerateContent` → OpenAI JSON/SSE |
| `openai_compatible` | yes | yes | Same wire protocol as OpenAI; any compatible base (Ollama, Groq, Azure OpenAI-compatible, etc.) |

Capabilities (`chat`, `stream`) are stored on the provider in the snapshot (defaults applied per type when empty). Streaming requests against a non-streaming provider return `400`.

## Adding a provider (in-tree)

1. Implement `dataplane.ChatProvider` (`Type`, `Capabilities`, `Chat`).
2. Register it in `dataplane.DefaultRegistry()` (or your gateway bootstrap).
3. Prefer reusing [`internal/dataplane/openaichat`](../../internal/dataplane/openaichat) helpers for OpenAI-shaped JSON/SSE.
4. Document the type, default `api_key_env`, and capabilities in this page.
5. Optionally seed an inactive provider (no route) like `prov_ollama`.

You should **not** need to edit `callProvider` or add a new `switch` case in the pipeline.

## Out-of-tree / future

See [`sdk/provider`](../../sdk/provider) for the documented contract. Full gRPC/WASM plugin runtimes are not shipped yet; the stable path today is in-process registration.

## Example: local Ollama

1. Create provider type `openai_compatible`, base URL `http://127.0.0.1:11434/v1`, env `OLLAMA_API_KEY` (any non-empty value if Ollama ignores auth).
2. Add a route, e.g. requested model `llama3` → target `llama3.2`.
3. Call the gateway with `"model":"llama3"`.
