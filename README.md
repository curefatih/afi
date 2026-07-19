<p align="center">
  <img src="assets/brand/logo.png" alt="AFI — AI Gateway" width="320" />
</p>

# AFI - AI Gateway

> [!WARNING]
> Under development. The control/data plane vertical slice is the supported local path.

AFI is a self-hostable LLM gateway with a **control plane** (configuration, identity, snapshots) and a **data plane** (high-performance inference). The data plane serves requests from immutable configuration snapshots — it never queries the config database on the hot path.

## Prerequisites

| Tool | Notes |
|------|--------|
| Go | See `go.mod` (`1.25.x`) |
| Docker | Postgres via Compose |
| pnpm | Optional, for `web/` |
| uv / uvx | Docs (`make doc-serve`) |
| OpenAI API key | For live inference |

## Quick start

```bash
# 1. Infrastructure
make dev-up

# 2. Provider key for the gateway
export OPENAI_API_KEY="sk-..."
# optional
export ANTHROPIC_API_KEY="sk-ant-..."
export GEMINI_API_KEY="..."

# 3. Control plane (migrate, seed, listen :8081)
make run-controlplane

# 4. Gateway (load snapshot, listen :8080) — second terminal
make run-gateway

# 5. Inference
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

Full checklist: [docs/getting-started/local-dev.md](docs/getting-started/local-dev.md).

## Components

| Process | Port | Role |
|---------|------|------|
| `controlplane` | `:8081` | Admin, platform API, snapshot publish |
| `gateway` | `:8080` | OpenAI-compatible inference + quota enforcement |
| `worker` | — | Drains usage outbox → `usage_events` |
| Postgres | `:5433` | Config + snapshots (`make dev-up`) |
| Adminer | `:5050` | DB UI |
| `web/` | `:3000` | Platform UI (`pnpm --dir web dev`) |

## Common commands

```bash
make build              # bin/controlplane, bin/gateway, bin/afi
make test               # unit tests (preferred quality bar)
make verify             # smoke against a running local stack
make seed               # CLI: seed local data + publish snapshot
make snapshot-publish
make doc-serve          # http://127.0.0.1:8000
```

## Architecture (short)

```mermaid
%%{init: {"theme":"base","themeVariables":{"primaryColor":"#ffffff","primaryTextColor":"#111111","primaryBorderColor":"#111111","lineColor":"#111111","secondaryColor":"#f5f5f5","tertiaryColor":"#ffffff","fontFamily":"ui-sans-serif,system-ui,sans-serif"}}}%%
flowchart LR
  UI[Platform UI / CLI] --> CP[Control Plane] --> SS[(Snapshot Store)]
  Clients --> GW[Gateway] --> PA[Provider adapters]
  SS -.->|watch / hot reload| GW

  classDef node fill:#ffffff,stroke:#111111,color:#111111,stroke-width:2px
  classDef store fill:#f5f5f5,stroke:#111111,color:#111111,stroke-width:2px
  class UI,CP,Clients,GW,PA node
  class SS store
```

See [docs/development/architecture.md](docs/development/architecture.md).
