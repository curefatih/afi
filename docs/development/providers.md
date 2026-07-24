# Provider adapters

Gateway chat dispatch uses a **registry** of in-process adapters. The pipeline looks up `provider.type` and calls `ChatIR` — it does not hard-code vendor branches.

## Built-in types

| Type | Chat | Stream | TTS | STT | Embedding | Image | Notes |
|------|------|--------|-----|-----|-----------|-------|-------|
| `openai` | yes | yes | yes | yes | yes | yes | Chat + audio + `/embeddings` + `/images/generations` |
| `anthropic` | yes | yes | no | no | no | no | Messages API → OpenAI-shaped responses/SSE |
| `gemini` | yes | yes | no | no | no | no | `generateContent` / `streamGenerateContent` → OpenAI JSON/SSE |
| `bedrock` | yes | yes | no | no | no | no | Bedrock **Converse** / **ConverseStream** (SigV4); tools + vision |
| `openai_compatible` | yes | yes | yes | yes | yes | yes | Same wire protocol as OpenAI (incl. audio/embeddings/images if upstream supports it) |
| `echo` | yes | no | no | no | no | no | **SDK extension** (`extensions/echo`) — no network; echoes last user message |

Capabilities (`chat`, `stream`, `tts`, `stt`, `embedding`, `image`) are stored on the provider in the snapshot (defaults applied per type when empty). Streaming/TTS/STT/embeddings/images requests against unsupported providers return `400`.

## Modality ports (chat / audio / embeddings / images)

| Surface | Registry port | Resolved by |
|---------|---------------|-------------|
| `POST /openai/v1/chat/completions` (+ `/v1/...`) | `IRChatProvider.ChatIR` | `provider.type` |
| `POST /anthropic/v1/messages` (+ `/v1/messages`) | same chat IR path | `provider.type` (any chat-capable provider) |
| `POST /gemini/v1beta/models/{route}:...` | same chat IR path | `provider.type` (any chat-capable provider) |
| `POST /openai/v1/audio/speech` / `transcriptions` (+ `/v1/audio/...`) | `AudioBackend` (via `OpenAITransportProvider`) | routed `provider.type` |
| `POST /openai/v1/embeddings` (+ `/v1/embeddings`) | `EmbeddingsBackend` (via `OpenAITransportProvider`) | routed `provider.type` |
| `POST /openai/v1/images/generations` (+ `/v1/images/generations`) | `ImagesBackend` (via `OpenAITransportProvider`) | routed `provider.type` |

Client **dialect** (path prefix) selects wire format; routing selects the upstream provider. See [API dialects](../api/dialects.md).

Chat stays on `IRChatProvider.ChatIR`. TTS/STT, embeddings, and images use **optional** transport interfaces — they are **not** methods on the chat IR contract, so SDK chat extensions need no modality stubs.

When an organization enables **object store** in control-plane settings (`GET/PUT …/organizations/{orgID}/object-store`), successful image generations may be persisted to S3-compatible storage and response URLs rewritten to presigned GET URLs. Disabled (default) keeps upstream passthrough.

Adapters that do not implement the transport provider interface simply cannot serve that modality (handlers return `400` / `502` as today).

## Adding a provider (in-tree)

1. Implement `dataplane.IRChatProvider` (`Type`, `Capabilities`, `ChatIR`) on a `ChatProvider` registered in the registry.
2. Register it in `dataplane.DefaultRegistry()` / `RegistryFromClients` (or your gateway bootstrap).
   Outbound HTTP clients live in `internal/adapters/llm` and resolve keys via `adapters/secrets`.
3. Prefer reusing [`internal/dataplane/openaichat`](../../internal/dataplane/openaichat) helpers for OpenAI-shaped JSON/SSE.
4. Document the type, default `api_key_env`, and capabilities in this page.
5. Optionally seed an inactive provider (no route) like `prov_ollama`.

You should **not** need to edit `callProviderIR` or add a new `switch` case in the pipeline.

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

## gRPC extensions (process-isolated)

Remote plugins speak [`proto/afi/extension/v1`](../../proto/afi/extension/v1/) and are discovered via gateway YAML `gateway.grpc_extensions` (command spawn or dial address). The host adapts them to `sdk/provider` / `sdk/hook`.

* **Typed chat IR:** advertise `CAPABILITY_PROVIDER_CHAT` and implement `Provider.ChatIR` / `ChatIRStream`.
* **Typed request mutation:** advertise `CAPABILITY_HOOK_BEFORE_CHAT` and implement `Hook.BeforeChat`.

Example: [`extensions/grpcecho`](../../extensions/grpcecho).

## Hooks (in-process)

`BeforeCall` / `AfterCall` run on all modalities. `ChatHook.BeforeChat` receives typed `sdk/chatir.Request`; `AfterChatHook.AfterChat` handles chat completion side effects. Register via `dataplane.NewHookChain().RegisterHook(...)` / `RegisterBeforeCall(...)` (see `extensions/demohook`). Gateway `/healthz` reports each supported phase. `extensions/tagquota` is an example-only BeforeCall sample for per-tag limits (not registered by default). WASM hooks: set `AFI_WASM_BEFORE_CALL` / `AFI_WASM_BEFORE_CHAT` (see [WASM hooks](../hooks/wasm.md)).

## Example: local Ollama

1. Create provider type `openai_compatible`, base URL `http://127.0.0.1:11434/v1`, env `OLLAMA_API_KEY` (any non-empty value if Ollama ignores auth).
2. Add a route, e.g. requested model `llama3` → target `llama3.2`.
3. Call the gateway with `"model":"llama3"`.

## Example: AWS Bedrock

1. Create provider type `bedrock`.
2. Set `base_url` to the regional runtime endpoint, e.g. `https://bedrock-runtime.us-west-2.amazonaws.com` (region is parsed from the host).
3. Auth:
   - Leave `api_key_env` empty (and assign no BYOK credential) to use the **AWS default credential chain** (env vars, instance profile, IRSA, etc.).
   - Or set `api_key_env` / BYOK to a secret whose value is `accessKeyID:secretAccessKey` or `accessKeyID:secretAccessKey:sessionToken`.
4. Add a route whose `target_model` is a Bedrock model id or inference profile (e.g. `anthropic.claude-3-haiku-20240307-v1:0`).
5. Call the route through the OpenAI-compatible `/openai/v1/chat/completions` (or `/v1/chat/completions`) interface. Anthropic and Gemini client dialects can also target the same route.

Vision inputs to Bedrock require **inline base64** image bytes (URL-only images are rejected).
