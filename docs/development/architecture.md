# Architecture

AFI separates **control plane** and **data plane**.

## Principles

1. The control plane owns business rules.
2. The data plane only executes requests.
3. Configuration is immutable at runtime (snapshots).
4. Every request completes without configuration database access (counters/outbox are operational state, not config).
5. Performance and operational simplicity take precedence over architectural purity.
6. New providers register through a stable adapter contract without editing the request pipeline core.

## Control plane

Uses pragmatic domain packages.

Responsibilities today:

* Persist orgs, projects, users, virtual API keys, providers, routes, quotas
* Create organizations and invite existing users by email (org membership roles: owner / admin / member)
* API keys: **personal** (user-scoped) and **service_account** (org- or project-scoped)
* Compile configuration into versioned snapshots (including provider capabilities)
* Platform HTTP APIs (`/api/v1/platform/*`)
* Internal admin (`/internal/v1/*`, `/healthz`)

## Data plane

Implemented as a **request pipeline**:

```mermaid
flowchart TD
  A[Authenticate — virtual API key]
  B[Load Snapshot — in-memory]
  C[QuotaCheck — request counters]
  D[Routing — model to provider + fallbacks]
  E[Provider registry — ChatProvider by type]
  F[EnqueueUsage — outbox]
  G[Response]

  A --> B --> C --> D --> E --> F --> G
```

Provider adapters (`openai`, `anthropic`, `gemini`, `openai_compatible`, …) implement `ChatProvider` and register in a registry. Optional modality ports (`AudioBackend`, `MessagesBackend`) are exposed by the same adapters and resolved by routed `provider.type`. See [Providers](providers.md).

Also exposes:

* `GET /v1/models` — virtual models from the key’s organization routes, enriched from the curated model catalog (`mode`, context limits, `supports_streaming` / `supports_tts` / `supports_stt`)
* `POST /v1/chat/completions` — OpenAI-shaped chat via `ChatProvider` (adapters translate native APIs)
* `POST /v1/messages` — Anthropic-shaped pass-through via `MessagesBackend`
* `POST /v1/audio/speech` / `POST /v1/audio/transcriptions` — TTS/STT via `AudioBackend`

The playground honors streaming/TTS/STT capabilities per model. Chat failover retries only before the response body is committed to the client (audio has no failover in this build).

Pipeline stages stay stateless aside from the in-memory snapshot pointer. Quota counters and the usage outbox use Postgres as operational stores.

## Snapshots

Snapshots contain:

* Virtual API keys (hashes) → org binding, optional project, kind, owner user id
* Providers (type, base URL, API key env ref, capabilities)
* Provider credentials (env ref or ciphertext) + assignments (provider type × org/project scope)
* Static model routes (optional fallbacks)
* Quotas (scope, metric, limit, window) — resolve order per window: api_key → user → project → organization
* CEL request policies (boolean allow-expressions)

Stored in Postgres (`gateway_snapshots`). The gateway watches for new versions (poll + `LISTEN/NOTIFY`) and hot-reloads.

## Async usage

```mermaid
flowchart LR
  Gateway --> usage_outbox --> worker --> usage_events
```

The request path never waits on `usage_events` consumers. Run `make run-worker` locally to populate the Usage UI (including `cost_usd` when prices match). Events carry a `modality` (`chat` / `messages` / `tts` / `stt`, …) and a `metrics` JSON object for non-token quantities; token columns remain for chat pricing. Cost uses DB `model_prices` overrides when present, otherwise the curated catalog in `internal/modelcatalog` (chat $/MTok, TTS $/character, STT $/second).

## Extensions (current)

In-process registration is live:

* **Providers** — `sdk/provider.ChatProvider` via `Registry.RegisterSDK` (example: `extensions/echo`)
* **Hooks** — `BeforeCall` / `AfterCall` (all modalities) plus `BeforeChat` / `AfterChat`; tags via `X-AFI-Tags` (`extensions/demohook`; example-only tag limits in `extensions/tagquota`)
* **WASM hooks** — sandboxed TinyGo guests via `internal/adapters/wasm` + org `wasm_hooks` in the snapshot (`AFI_WASM_*` env still works for demos). See [WASM hooks](../hooks/wasm.md).
* **Provider health** — control-plane rollup from `usage_events` for Providers UI

Control-plane WASM hook bindings are available; gRPC plugin runtimes, billing invoices, external HashiCorp Vault, and multi-region snapshot distribution remain future work.

**Shipped governance:**

* **Quotas** — `total` windows on Postgres; `minute` / `hour` / `day` rate limits on Redis (`AFI_REDIS_URL`)
* **CEL policies** — org allow-expressions in the snapshot; deny → HTTP 403 `policy_violation`
* **Provider credentials** — org-owned secrets (`env` or AES-GCM `encrypted_db`) assignable to organization/project scopes; gateway resolves project → org → provider `api_key_env` fallback
