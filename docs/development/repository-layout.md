# Repository layout

Matches the intended project structure from the architecture document.

```text
cmd/
├── gateway/          # Data plane binary
├── controlplane/     # Control plane binary
├── worker/           # Async consumers (future)
└── cli/              # Local admin CLI (afi)

internal/
├── kernel/           # Shared primitives (logging, errors, IDs)
├── controlplane/     # Config persistence, platform APIs, seed, compile
├── dataplane/        # Request pipeline + provider adapters
├── snapshot/         # Snapshot types, store, watch
├── workers/          # Future workers
└── shared/           # Cross-cutting helpers

extensions/           # Runtime extensions (future)
sdk/                  # Extension SDK (future)
api/                  # Public API contracts (future)
web/                  # Platform UI (TanStack / Vite)
configs/              # Local/dev defaults
docs/                 # Public MkDocs site
```

## Ownership (current milestone)

| Path | Responsibility |
|------|----------------|
| `cmd/controlplane` | HTTP admin + platform API + migrate + seed + publish |
| `cmd/gateway` | Load/watch snapshot, serve `/v1/*` |
| `cmd/cli` | `seed`, `snapshot publish`, `version` |
| `internal/snapshot` | Types, compile helpers, Postgres store + watch |
| `internal/controlplane` | Schema, repositories, HTTP handlers |
| `internal/dataplane` | Auth → route → OpenAI pipeline |
| `internal/kernel` | Logging, request IDs, config loading |
| `configs/` | `local.yaml` defaults |
