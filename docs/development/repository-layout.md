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
├── controlplane/     # Config persistence, platform APIs, seed, compile
├── dataplane/        # Request pipeline + provider adapters
├── snapshot/         # Snapshot types, store, watch
├── workers/          # Outbox processing helpers
└── shared/           # Cross-cutting helpers

extensions/           # Runtime extensions (future)
sdk/                  # Extension SDK (future)
api/                  # Public API contracts (future)
web/                  # Platform UI (TanStack / Vite)
configs/              # Local/dev defaults
docs/                 # Public MkDocs site
```

## Ownership (current)

| Path | Responsibility |
|------|----------------|
| `cmd/controlplane` | HTTP admin + platform API + migrate + seed + publish |
| `cmd/gateway` | Load/watch snapshot, quotas, `/v1/*`, enqueue usage |
| `cmd/worker` | Drain `usage_outbox` → `usage_events` (+ cost) |
| `cmd/cli` | `seed`, `snapshot publish`, `db reset`, `version` |
| `internal/snapshot` | Types, compile, Postgres store + watch |
| `internal/controlplane` | Schema, repositories, HTTP handlers |
| `internal/dataplane` | Auth → quota → route → OpenAI/Anthropic/Gemini (+ failover, `/v1/models`) |
| `internal/workers` | Outbox process loop helpers |
| `internal/kernel` | Logging, request IDs, config loading |
| `configs/` | `local.yaml` defaults |
