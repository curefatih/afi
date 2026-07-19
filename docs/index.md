# AFI

AFI is a self-hostable, cloud-native **LLM gateway**.

It has two major parts:

* **Control plane** — configuration, identities, policies, quotas, routing, and platform APIs. Owns business rules and compiles **immutable snapshots**.
* **Data plane (gateway)** — processes inference with a request pipeline. Loads snapshots and never queries the configuration database during a request.

Start here: [Local development](getting-started/local-dev.md).

## High-level flow

```mermaid
flowchart TB
  UI[Platform UI]
  CP[Control Plane]
  SS[Snapshot Store]
  DP[Data Plane]
  PA[Provider Adapters]
  Prov[OpenAI / Anthropic / Gemini]

  UI --> CP
  CP -->|Builds gateway snapshot| SS
  SS -->|Watch / hot reload| DP
  DP --> PA
  PA --> Prov
```

## What works locally today

* Postgres + Adminer via `make dev-up`
* Control plane: migrate, seed, snapshot publish, platform auth APIs
* Gateway: virtual API key auth → quotas → routes (with failover) → provider registry (`openai`, `anthropic`, `gemini`, `openai_compatible`, …)
* OpenAI-compatible `POST /v1/chat/completions` (streaming gated by provider capabilities)
* OpenAI-compatible `GET /v1/models` (lists org routes from the snapshot)
* Usage outbox + worker with optional `cost_usd`
* Web UI against the control plane (`:8081`), playground against the gateway (`:8080`)
* Docs via `make doc-serve`
