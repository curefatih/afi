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
  OAI[OpenAI / Anthropic]

  UI --> CP
  CP -->|Builds gateway snapshot| SS
  SS -->|Watch / hot reload| DP
  DP --> PA
  PA --> OAI
```

## What works locally today

* Postgres + Adminer via `make dev-up`
* Control plane: migrate, seed, snapshot publish, platform auth APIs
* Gateway: virtual API key auth → route → OpenAI chat completions (stream + non-stream)
* Web UI against the control plane (`:8081`)
* Docs via `make doc-serve`
