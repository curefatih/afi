# Plugins / hooks

## Today (in-process)

The gateway runs an in-process hook chain on the chat path:

1. **BeforeChat** — after auth/quota, before provider dispatch (`ChatHook`)
2. **AfterChat** — after the attempt finishes, success or error (`AfterChatHook`)

Register in `cmd/gateway`:

```go
hooks := dataplane.NewHookChain().RegisterHook(demohook.NewWithLog(log))
pipeline.Hooks = hooks
```

Inspect active hooks on `GET /healthz` (`hooks` array of `{name, before_chat, after_chat}`) or the platform **Hooks** page.

Seeded demo: [`extensions/demohook`](../../extensions/demohook) prefixes the last user message with `[hook:demo]` and logs AfterChat to the gateway logger.

Provider adapters use the same in-process path via [`sdk/provider`](../../sdk/provider) + `Registry.RegisterSDK` (see [`extensions/echo`](../../extensions/echo)).

## Future

* gRPC extensions — providers, auth, secrets, notifications
* WASM extensions — prompt/header mutation, PII masking, enrichment

Redis rate limits and CEL request policies are shipped (see Quotas / Policies in the platform UI).
