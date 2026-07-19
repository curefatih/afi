# Repository layout

Matches the intended project structure from the architecture document.

```text
cmd/
├── gateway/          # Data plane binary
├── controlplane/     # Control plane binary
├── worker/           # Usage outbox consumer
└── cli/              # Local admin CLI (afi)

internal/
├── kernel/           # Shared primitives (logging, errors, IDs)
├── adapters/
│   └── postgres/     # Counters, usage outbox/sink, prices, snapshots
├── controlplane/     # Config persistence, platform APIs, seed, compile
├── dataplane/        # Request pipeline + provider adapters
├── snapshot/         # Snapshot types, store, watch
├── workers/          # Outbox ports + ProcessOnce
└── shared/           # Placeholder

extensions/           # In-process examples (echo provider, demohook); gRPC/WASM later
sdk/provider/         # Documented ChatProvider contract
api/                  # Public API contracts (future)
web/                  # Platform UI (TanStack / Vite)
configs/              # Local/dev defaults
docs/                 # Public MkDocs site
```


## Ownership (current)

| Path | Responsibility |
|------|----------------|
| `cmd/controlplane` | HTTP admin + platform API + migrate + seed + publish |
| `cmd/gateway` | Load/watch snapshot, quotas, `/v1/*`, enqueue usage (via adapters) |
| `cmd/worker` | Drain `usage_outbox` → `usage_events` (+ cost) via adapters |
| `cmd/cli` | `seed`, `snapshot publish`, `db reset`, `version` |
| `internal/adapters/postgres` | Lifetime counters, usage outbox/sink, prices, snapshot store |
| `internal/snapshot` | Types, compile, Store port (no Postgres) |
| `internal/controlplane` | Schema, repositories, HTTP handlers |
| `internal/dataplane` | Auth → quota → route → provider registry (+ failover, `/v1/models`) |
| `sdk/provider` | Documented adapter contract for multi-model extensibility |
| `extensions/*` | Example SDK providers + hooks registered from `cmd/gateway` |
| `internal/workers` | Outbox ports + process loop helpers |
| `internal/kernel` | Logging, request IDs, config loading |
| `configs/` | `local.yaml` defaults |
