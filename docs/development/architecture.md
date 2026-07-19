# Architecture

AFI separates **control plane** and **data plane**.

## Principles

1. The control plane owns business rules.
2. The data plane only executes requests.
3. Configuration is immutable at runtime (snapshots).
4. Every request completes without configuration database access.
5. Performance and operational simplicity take precedence over architectural purity.

## Control plane

Uses pragmatic domain packages (full DDD bounded contexts grow over time).

Responsibilities today:

* Persist orgs, projects, users, virtual API keys, providers, routes
* Compile configuration into versioned snapshots
* Platform HTTP APIs (`/api/v1/platform/*`)
* Internal admin (`/internal/v1/*`, `/healthz`)

## Data plane

Implemented as a **request pipeline**, not DDD:

```text
Authenticate (virtual API key)
↓
Load Snapshot (in-memory current version)
↓
Routing (model → provider/model)
↓
Provider (OpenAI chat completions)
↓
Response (+ structured usage logs)
```

Pipeline stages stay stateless aside from the in-memory snapshot pointer.

## Snapshots

Snapshots contain:

* Virtual API keys → project binding
* Providers (type, base URL, API key env ref)
* Static model routes

Stored in Postgres (`gateway_snapshots`). The gateway watches for new versions (poll + `LISTEN/NOTIFY`) and hot-reloads.

## Eventual extensions

Plugin runtimes (gRPC / WASM), workers, quotas, billing, and multi-provider routing are planned but not required for the local vertical slice.
