# Provider adapters

Gateway chat dispatch uses a **registry** of in-process adapters. The pipeline looks up `provider.type` and calls `ChatIR` — it does not hard-code vendor branches.

Built-in types, default credentials, UI presets, and local-dev seed rows all come from a single catalog in [`internal/providercatalog`](../../internal/providercatalog). Each in-tree adapter registers a factory via `registerBuiltin` in [`internal/dataplane`](../../internal/dataplane).

## Built-in types

| Type | Chat | Stream | TTS | STT | Embedding | Image | Notes |
|------|------|--------|-----|-----|-----------|-------|-------|
| `openai` | yes | yes | yes | yes | yes | yes | Chat + audio + `/embeddings` + `/images/generations` |
| `openai_compatible` | yes | yes | yes | yes | yes | yes | Same wire protocol as OpenAI (incl. audio/embeddings/images if upstream supports it) |
| `azure_openai` | yes | yes | yes | yes | yes | yes | Azure OpenAI; `api-key` auth; `config.api_style` = `deployments` (default) or `openai_v1` |
| `anthropic` | yes | yes | no | no | no | no | Messages API → OpenAI-shaped responses/SSE |
| `gemini` | yes | yes | no | no | no | no | `generateContent` / `streamGenerateContent` → OpenAI JSON/SSE |
| `bedrock` | yes | yes | no | no | no | no | Bedrock **Converse** / **ConverseStream** (SigV4); tools + vision |
| `elevenlabs` | no | no | yes | yes | no | no | TTS/STT via ElevenLabs API (`xi-api-key`); OpenAI audio dialect in, vendor wire out |
| `echo` | yes | no | no | no | no | no | **SDK extension** (`extensions/echo`) — no network; echoes last user message |

Capabilities (`chat`, `stream`, `tts`, `stt`, `embedding`, `image`) are stored on the provider in the snapshot (defaults applied per type when empty). Streaming/TTS/STT/embeddings/images requests against unsupported providers return `400`.

## Modality ports (chat / audio / embeddings / images)

| Surface | Registry port | Resolved by |
|---------|---------------|-------------|
| `POST /openai/v1/chat/completions` (+ `/v1/...`) | `IRChatProvider.ChatIR` | `provider.type` |
| `POST /anthropic/v1/messages` (+ `/v1/messages`) | same chat IR path | `provider.type` (any chat-capable provider) |
| `POST /gemini/v1beta/models/{route}:...` | same chat IR path | `provider.type` (any chat-capable provider) |
| `POST /openai/v1/audio/speech` / `transcriptions` (+ `/v1/audio/...`) | `AudioBackend` (via `AudioTransportProvider` or `OpenAITransportProvider`) | routed `provider.type` |
| `POST /openai/v1/embeddings` (+ `/v1/embeddings`) | `EmbeddingsBackend` (via `OpenAITransportProvider`) | routed `provider.type` |
| `POST /openai/v1/images/generations` (+ `/v1/images/generations`) | `ImagesBackend` (via `OpenAITransportProvider`) | routed `provider.type` |

Client **dialect** (path prefix) selects wire format; routing selects the upstream provider. See [API dialects](../api/dialects.md).

Chat stays on `IRChatProvider.ChatIR`. TTS/STT, embeddings, and images use **optional** transport interfaces — they are **not** methods on the chat IR contract, so SDK chat extensions need no modality stubs.

When an organization enables **object store** in control-plane settings (`GET/PUT …/organizations/{orgID}/object-store`), successful image generations may be persisted to S3-compatible storage and response URLs rewritten to presigned GET URLs. Disabled (default) keeps upstream passthrough.

Adapters that do not implement the transport provider interface simply cannot serve that modality (handlers return `400` / `502` as today).

## Adding a provider (in-tree)

1. Implement the outbound client under [`internal/adapters/llm`](../../internal/adapters/llm) (and tests).
2. Add a `providercatalog.Spec` in [`internal/providercatalog/builtins.go`](../../internal/providercatalog/builtins.go) (type, display name, default base URL / `api_key_env`, capabilities, `AuthMode`, `UIVisible`, `Seed`, optional `CatalogAlias` / `SeedRoute`).
3. Register a factory with `registerBuiltin(type, factory)` in [`internal/dataplane/register_builtins.go`](../../internal/dataplane/register_builtins.go) (or a new `register_*.go` in the same package) that wraps the client as a `ChatProvider` / modality ports.
4. Prefer reusing [`internal/dataplane/openaichat`](../../internal/dataplane/openaichat) helpers for OpenAI-shaped JSON/SSE.
5. Optionally add curated model rows to [`internal/modelcatalog/catalog.json`](../../internal/modelcatalog/catalog.json) (pricing/context windows stay manual).

That single Spec + factory covers: gateway registry, capability/env defaults, auth rules (`AuthOptional` for empty `api_key_env`), UI presets (`GET /api/v1/platform/provider-types`), and local-dev seed rows.

You should **not** need to edit `callProviderIR`, modality type allowlists, `Clients` structs, or hard-coded UI preset maps.

## Adding an extension (SDK)

1. Implement [`sdk/provider.ChatProvider`](../../sdk/provider) in a package under [`extensions/`](../../extensions/) (or an external module).
2. Add a metadata-only Spec in `providercatalog/builtins.go` (no `registerBuiltin` factory).
3. In [`cmd/gateway`](../../cmd/gateway), call `reg.RegisterSDK(your.New())` after `DefaultRegistry()`.
4. Create a control-plane provider with matching `type` and a route (seed does this for `echo` → model `echo-demo`).
5. Restart the gateway so the adapter is registered.

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

`BeforeCall` / `AfterCall` run on all modalities. `ChatHook.BeforeChat` receives typed `sdk/chatir.Request`; `AfterChatHook.AfterChat` handles chat completion side effects. Register via `dataplane.NewHookChain().RegisterHook(...)` / `RegisterBeforeCall(...)` (see `extensions/demohook`). Gateway `/healthz` reports each supported phase. `extensions/tagquota` is an example-only BeforeCall sample for per-tag limits (not registered by default). WASM hooks: set `AFI_WASM_BEFORE_CALL` / `AFI_WASM_BEFORE_CHAT` (see [WASM hooks](hooks/wasm.md)).

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

## Example: Azure OpenAI

1. Create provider type `azure_openai`.
2. Set `base_url`:
   - **deployments** (default): resource root, e.g. `https://myresource.openai.azure.com`
   - **openai_v1**: include the v1 prefix, e.g. `https://myresource.openai.azure.com/openai/v1`
3. Set provider `config`:
   - `api_style`: `deployments` or `openai_v1`
   - `api_version` (optional): defaults to `2024-10-21` for deployments; for `openai_v1` only sent when set
4. Set `api_key_env` to `AZURE_OPENAI_API_KEY` (or your BYOK credential). The gateway sends the `api-key` header.
5. Add a route whose `target_model` is the **deployment name**.
6. Call through `/openai/v1/chat/completions` (or `/v1/...`). Anthropic and Gemini client dialects can target the same route.

## Example: ElevenLabs (TTS / STT)

1. Create provider type `elevenlabs`, base URL `https://api.elevenlabs.io`, env `ELEVENLABS_API_KEY`.
2. Add a TTS route, e.g. requested model `eleven-tts` → target `eleven_multilingual_v2` (or `eleven_turbo_v2_5` / `eleven_flash_v2_5`).
3. Call OpenAI-shaped `POST /v1/audio/speech` with `"model":"eleven-tts"`. Optional `"voice"` may be an ElevenLabs voice id; OpenAI voice names map to a default voice.
4. For STT, route target `scribe_v2` and call `POST /v1/audio/transcriptions` (multipart `model` + `file`).

The gateway keeps the OpenAI client dialect and translates to ElevenLabs `/v1/text-to-speech/{voice_id}` and `/v1/speech-to-text`.
