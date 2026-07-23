# Observability (OpenTelemetry)

AFI exports **metrics** and **traces** through [OpenTelemetry](https://opentelemetry.io/). Application code uses the OTel API only — there are no Datadog, New Relic, or Prometheus client libraries in the request path.

## Signals

| Signal | Status |
| ------ | ------ |
| Traces | Supported (W3C Trace Context) |
| Metrics | Supported (OTLP push and/or Prometheus scrape) |
| Logs | Structured `slog` only (no OTel log bridge yet) |

## Configuration

All flags are under `telemetry` / `AFI_TELEMETRY_*` (see [config reference](../development/config-reference.md)).

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| `AFI_TELEMETRY_ENABLED` | `false` | Master switch |
| `AFI_TELEMETRY_SERVICE_NAME` | binary default (`afi-gateway`, …) | OTel `service.name` |
| `AFI_TELEMETRY_ENVIRONMENT` | empty | `deployment.environment` |
| `AFI_TELEMETRY_OTLP_ENDPOINT` | empty | Host:port or URL for OTLP |
| `AFI_TELEMETRY_OTLP_PROTOCOL` | `http` | `http` or `grpc` |
| `AFI_TELEMETRY_OTLP_HEADERS` | empty | `k=v,k=v` (vendor API keys) |
| `AFI_TELEMETRY_OTLP_INSECURE` | `false` | Disable TLS (local collector) |
| `AFI_TELEMETRY_METRICS_PROMETHEUS` | `false` | Expose `GET /metrics` on gateway and control plane |
| `AFI_TELEMETRY_TRACES_SAMPLER` | `parentbased_always_on` | Or `parentbased_traceidratio` |
| `AFI_TELEMETRY_TRACES_SAMPLER_ARG` | `1.0` | Ratio when using `traceidratio` |

**Worker:** OTLP push only (no listen port for `/metrics` in v1).

## Local Grafana

Use the opt-in Compose profile with Grafana’s all-in-one [`otel-lgtm`](https://hub.docker.com/r/grafana/otel-lgtm) image (OTLP → Prometheus + Tempo + Grafana UI).

```bash
make obs-up
# Grafana: http://localhost:3000
# OTLP HTTP: localhost:4318  OTLP gRPC: localhost:4317

set -a && source .env.observability && set +a
make run-controlplane   # separate terminals
make run-worker
make run-gateway
```

In Grafana Explore:

* **Prometheus** — query `afi_gateway_requests_total` (or `afi_gateway_requests`)
* **Tempo** — search by `service.name` = `afi-gateway` / `afi-controlplane` / `afi-worker`

Stop with `make obs-down`. Default `make dev-up` does **not** start this stack.

## Production backends

Point `AFI_TELEMETRY_OTLP_ENDPOINT` at:

* An [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) (then fan-out to Prometheus remote_write, Datadog, etc.), or
* A vendor’s native OTLP ingest (Grafana Cloud, Datadog, Honeycomb, New Relic, Elastic, …) with `AFI_TELEMETRY_OTLP_HEADERS` for auth.

For Kubernetes scrape-only setups, set `AFI_TELEMETRY_METRICS_PROMETHEUS=true` and scrape gateway/controlplane `/metrics`.

AFI does **not** embed vendor agents.

## Cardinality

Metric attributes stay low-cardinality: `afi.modality`, `afi.outcome`, `afi.provider_type`, HTTP route templates, status class. Org IDs, models, and end-user tags belong on **spans**, not metric labels.

## Key instruments

| Name | Process |
| ---- | ------- |
| `afi.gateway.requests` | gateway |
| `afi.gateway.request_duration` | gateway |
| `afi.gateway.upstream_duration` | gateway |
| `afi.gateway.tokens` | gateway |
| `afi.gateway.quota_rejections` | gateway |
| `afi.gateway.snapshot_version` | gateway |
| `afi.controlplane.http_requests` | control plane |
| `afi.controlplane.http_duration` | control plane |
| `afi.worker.usage_processed` | worker |
| `afi.worker.usage_errors` | worker |
| `afi.worker.platform_events_published` | worker |
| `afi.worker.process_duration` | worker |
