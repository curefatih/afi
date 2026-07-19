# Repository layout

Matches the intended project structure from the architecture document.

```text
cmd/
├── gateway/          # Data plane binary
├── controlplane/     # Control plane binary
├── worker/           # Usage + platform-event outbox consumer
└── cli/              # Local admin CLI (afi)

internal/
├── kernel/           # Shared primitives (logging, errors, IDs)
├── identity/         # User domain/ports
├── tenancy/          # Org/Team/Project + membership
├── access/           # APIKey domain/ports
├── gatewayconfig/    # Quota, Policy, Provider, Route domain/ports
├── usage/            # usage.Event (emit/outbox) + reporting read models
├── adapters/
│   ├── auth/         # JWT + bcrypt
│   ├── postgres/     # Persistence + usage/platform event outboxes
│   ├── redis/        # Timed quota CounterStore
│   ├── llm/          # OpenAI / Anthropic / Gemini HTTP clients
│   ├── secrets/      # SecretResolver (env today)
│   ├── natsjs/       # JetStream platform-event publisher
│   ├── kafka/        # Kafka platform-event publisher
│   ├── logpub/       # Log stand-in publisher
│   └── eventpub/     # Publisher factory (log|nats|kafka|noop)
├── app/platform/     # Queries, commands, event bus + outbox handler
├── controlplane/     # HTTP transport, Store facade, seed, migrate
├── dataplane/        # Request pipeline + ChatProvider registry
├── snapshot/         # Snapshot types, compile, Store port
├── workers/          # Usage + platform-event ProcessOnce
└── shared/           # Placeholder

extensions/           # In-process examples (echo provider, demohook); gRPC/WASM later
sdk/provider/         # Documented ChatProvider contract
api/                  # Public API contracts (future)
web/                  # Platform UI (TanStack / Vite)
configs/              # Local/dev defaults
docs/                 # Public MkDocs site
```

Platform events (bus + durable outbox): [Platform domain events](platform-events.md).

## Ownership (current)

| Path | Responsibility |
|------|----------------|
| `cmd/controlplane` | HTTP admin + platform API + migrate + seed + publish |
| `cmd/gateway` | Load/watch snapshot, quotas, `/v1/*`, enqueue usage (via adapters) |
| `cmd/worker` | Drain `usage_outbox` → `usage_events`; drain `platform_event_outbox` → broker |
| `cmd/cli` | `seed`, `snapshot publish`, `db reset`, `version` |
| `internal/identity` | User domain + repository port |
| `internal/tenancy` | Organization, Team, Project, membership role rules |
| `internal/access` | APIKey domain, ports, create use case |
| `internal/gatewayconfig` | Quota, RequestPolicy, Provider, Route domain/ports |
| `internal/usage` | Canonical usage.Event (emit/outbox) + Record/Filter/Summary reporting types |
| `internal/adapters/postgres` | Persistence + usage/platform event outboxes |
| `internal/adapters/auth` | JWT + bcrypt |
| `internal/adapters/redis` | Timed quota windows |
| `internal/adapters/llm` | Vendor HTTP clients (+ secrets) |
| `internal/adapters/secrets` | SecretResolver |
| `internal/adapters/natsjs` | NATS JetStream event publisher |
| `internal/adapters/kafka` | Kafka event publisher |
| `internal/adapters/logpub` | Log stand-in event publisher |
| `internal/adapters/eventpub` | Publisher factory (log\|nats\|kafka\|noop) |
| `internal/app/platform` | Platform queries + commands + event bus / outbox handler |
| `internal/snapshot` | Types, compile, Store port (no Postgres) |
| `internal/controlplane` | HTTP handlers, Store facade, seed, migrate |
| `internal/dataplane` | Auth → quota → route → provider registry (+ failover, `/v1/models`) |
| `sdk/provider` | Documented adapter contract for multi-model extensibility |
| `extensions/*` | Example SDK providers + hooks registered from `cmd/gateway` |
| `internal/workers` | Usage + platform-event outbox ProcessOnce |
| `internal/kernel` | Logging, request IDs, config loading |
| `configs/` | `local.yaml` defaults |
