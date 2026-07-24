# Plugins / hooks

## Today (in-process)

The gateway runs an in-process hook chain on every modality (chat, messages, TTS, STT):

1. **BeforeCall** — after auth, before routing/provider. Receives a mutable `CallContext` (principal, route, tags from `X-AFI-Tags`, metadata, body). May **allow**, **enrich**, or **deny** (`Allow=false` + status/reason).
2. **BeforeChat** — OpenAI, Anthropic, and Gemini chat dialects; mutates the **OpenAI-shaped bridge body** after BeforeCall allows (existing `ChatHook`).
3. **AfterCall** — after the upstream attempt finishes (all modalities).
4. **AfterChat** — all three chat dialects; logging/side effects after AfterCall (`AfterChatInfo` includes `Dialect` and `Modality`).

### Chat dialect hook bodies

| Hook | Body / info shape |
| ---- | ----------------- |
| **BeforeCall** `call.Body` | OpenAI chat.completions JSON derived from IR (stable bridge for all chat dialects) |
| **BeforeCall** `call.Metadata["dialect"]` | `"openai"` or `"anthropic"` |
| **BeforeCall** `call.Metadata["client_body"]` | Original client wire bytes (Anthropic Messages JSON or OpenAI JSON) |
| **BeforeChat** / WASM `before_chat` | Same OpenAI-shaped bridge JSON as `call.Body` — mutate messages here; keep `model` intact (routing uses the original route name) |
| **AfterChat** | `AfterChatInfo` with `Model`, `Status`, `LatencyMs`, `ProviderType`, `TargetModel`, plus `Dialect` / `Modality` |

Dialect-native **mutation** of Anthropic JSON (or typed IR) in BeforeChat is a follow-on for WASM/gRPC extensions; until then, rewrite via the OpenAI bridge or inspect `client_body` in BeforeCall.

Built-in gates always run in the request path (not registered on `pipeline.Hooks`, so replacing the chain cannot bypass them). Order:

1. **cel_policy** — org CEL allow-policies → 403 `policy_violation`
2. user **BeforeCall** hooks
3. **quota** — request quotas → 429 `insufficient_quota` (last so denied/errored hooks do not consume quota)

Token quotas still increment after the response (`incrTokens`).

Register in `cmd/gateway`:

```go
hooks := dataplane.NewHookChain().
    RegisterHook(demohook.NewWithLog(log)).
    RegisterBeforeCall(myBeforeCall)
pipeline.Hooks = hooks
```

Inspect active hooks on `GET /healthz` (`hooks` array of `{name, before_call, after_call, before_chat, after_chat}`) or the platform **Hooks** page.

### Request tags

Clients may send:

```http
X-AFI-Tags: end-user-id:fatihcure,tenant:acme
```

Parsed into `CallContext.Tags` and persisted on usage events (`tags` JSONB).

### Example: per-tag request limits

[`extensions/tagquota`](../../extensions/tagquota) is a **reference implementation** of `BeforeCall` with independent counters per tag value (e.g. each `end-user-id`). It is **not** registered by the default gateway — copy or register it yourself if you need that behavior. See the package [README](../../extensions/tagquota/README.md).

Seeded demo: [`extensions/demohook`](../../extensions/demohook) prefixes the last user message with `[hook:demo]` and logs AfterChat.

Provider adapters use [`sdk/provider`](../../sdk/provider) + `Registry.RegisterSDK` (see [`extensions/echo`](../../extensions/echo)). Lifecycle types live in [`sdk/hook`](../../sdk/hook).

## WASM (data plane)

Sandboxed TinyGo modules via [`internal/adapters/wasm`](../../internal/adapters/wasm):

* **Env:** `AFI_WASM_BEFORE_CALL` / `AFI_WASM_BEFORE_CHAT` (process-global)
* **Control plane:** org `wasm-hooks` API → snapshot → gateway module cache (URI + optional SHA-256 digest)

Guest ABI, limits, pooling benchmarks: [wasm.md](wasm.md). Example: [`extensions/wasmhook`](../../extensions/wasmhook).

## Future

* Dialect-native or typed-IR **BeforeChat** mutation bodies (WASM/gRPC) — today mutation stays on the OpenAI bridge; see [dialects](../api/dialects.md)
* gRPC auth / secrets / notifications host adapters (capabilities reserved; Chat + hooks shipped — see [`extensions/grpcecho`](../../extensions/grpcecho))
* Remote / HTTP-backed WASM artifact stores (beyond local `file://` paths)

Redis rate limits and CEL request policies remain available via the built-in BeforeCall gates (see Quotas / Policies in the platform UI).

## gRPC (process-isolated)

Configure `gateway.grpc_extensions` in YAML. Plugins implement Handshake + optional Provider/Hook services. See [providers.md](../development/providers.md#grpc-extensions-process-isolated).
