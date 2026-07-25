<p align="center">
  <img src="assets/brand/logo.svg" alt="AFI — AI Gateway" width="280" />
</p>

# AFI - AI Gateway

AFI is a self-hostable LLM gateway with a **control plane** (configuration, identity, snapshots) and a **data plane** (high-performance inference). The data plane serves requests from immutable configuration snapshots — it never queries the config database on the hot path.

## Prerequisites

| Tool | Notes |
|------|--------|
| Docker | Compose v2 — enough for Quick start |
| Go | See `go.mod` (`1.25.x`) — local development only |
| pnpm | Optional, for `web/` and docs (`make doc-serve`) |
| OpenAI API key | For live inference |

## Quick start (Docker Compose)

```bash
export OPENAI_API_KEY="sk-..."   # optional but needed for chat
make quickstart
```

Then open http://localhost:3000 (`admin@afi.local` / `admin`) or call the gateway:

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
```

Guide: [docs/getting-started/quick-start.md](docs/getting-started/quick-start.md). Host-based Go workflow: [docs/getting-started/local-dev.md](docs/getting-started/local-dev.md).

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
make quickstart         # full stack via Docker Compose
make build              # bin/controlplane, bin/gateway, bin/afi
make test               # unit tests (preferred quality bar)
make verify             # smoke against a running local stack
make seed               # CLI: seed local data + publish snapshot
make snapshot-publish
make doc-serve          # http://localhost:4321
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
