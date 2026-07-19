# Plugins / hooks

## Today (in-process)

The gateway runs a small **BeforeChat** hook chain after auth/quota and before the provider adapter:

1. Implement `dataplane.ChatHook` (`Name`, `BeforeChat`).
2. Register it on the pipeline: `pipeline.Hooks = dataplane.NewHookChain().Register(myHook)`.
3. Inspect active hooks on `GET /healthz` (`hooks` array) or the platform **Hooks** page.

Seeded demo: [`extensions/demohook`](../../extensions/demohook) prefixes the last user message with `[hook:demo]`.

Provider adapters use the same in-process path via [`sdk/provider`](../../sdk/provider) + `Registry.RegisterSDK` (see [`extensions/echo`](../../extensions/echo)).

## Future

Target architecture still includes:

* gRPC extensions — providers, auth, secrets, notifications
* WASM extensions — prompt/header mutation, PII masking, enrichment

Those runtimes are **not** shipped yet.
