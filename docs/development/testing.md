# Testing

## Unit tests (test-first)

```bash
make test
# or
go test ./...
```

Covered areas:

* Snapshot compile + org-scoped `LookupRoute` + hashed `LookupKey` + quota resolve
* Data-plane auth, quota 429 / under-limit, Redis minute rate limit, CEL policy 403, provider registry, openai_compatible, capability stream reject, Anthropic SSE, `/v1/models`, route failover
* Usage outbox worker `ProcessOnce` + cost calculation

* Control-plane JWT/password helpers
* Internal token checks
* Org membership middleware (403 when not a member)
* Publish error surfaced on create project
* Migrate wipe policy (version bumps never wipe)

### WASM vs native hook benchmarks

```bash
go test ./internal/adapters/wasm/ \
  -bench='BenchmarkBefore(Call|Chat)_' \
  -benchmem -count=10 -benchtime=500ms
```

Compares in-process Go hooks to TinyGo WASM (`extensions/wasmhook`). Documented numbers, collapsible code samples, and guidance: [WASM hooks — Performance](../hooks/wasm.md#performance-wasm-vs-native-go).

## Automated local verify

With Postgres + control plane + gateway already running:

```bash
make verify
```

Script: [`scripts/verify-local.sh`](../../scripts/verify-local.sh).

Checks health, login, bad key → 401, internal publish auth, snapshot hot reload, optional live chat when `OPENAI_API_KEY` is set, optional usage drain when the worker is running, quota → 429, CEL deny → 403, and Redis minute rate limit → 429 when Redis is up.

## Manual smoke

See [Verify](../getting-started/verify.md) for curl recipes including UI-edited routes.

## Web

```bash
pnpm --dir web test
pnpm --dir web check
```
