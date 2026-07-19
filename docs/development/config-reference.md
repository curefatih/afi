# Config reference

## File

[`configs/local.yaml`](../../configs/local.yaml) — defaults for local development.

## Environment variables

| Variable | Default | Used by |
|----------|---------|---------|
| `AFI_CONFIG` | `configs/local.yaml` | controlplane, gateway, cli |
| `AFI_DATABASE_URL` | from yaml / compose DSN | all DB clients |
| `AFI_CONTROLPLANE_ADDR` | `:8081` | controlplane |
| `AFI_GATEWAY_ADDR` | `:8080` | gateway |
| `AFI_SNAPSHOT_POLL_INTERVAL` | `2s` | gateway watch |
| `AFI_JWT_SECRET` | from yaml | controlplane auth |
| `OPENAI_API_KEY` | _(required for live calls)_ | gateway → OpenAI |
| `VITE_PLATFORM_API_URL` | `http://localhost:8081` | web UI (platform APIs) |
| `VITE_GATEWAY_API_URL` | `http://localhost:8080` | web playground |
| `VITE_GATEWAY_API_KEY` | seed virtual key | web playground |

## Database

Compose defaults ([`docker-compose.yml`](../../docker-compose.yml)):

```text
postgres://afi:afi@localhost:5433/afi?sslmode=disable
```

## Seed values

Written on first control-plane start (or `make seed`):

| Item | Value |
|------|--------|
| Virtual API key | `sk-project-local-dev-token-12345` |
| Platform email | `admin@afi.local` |
| Platform password | `admin` |
| Default model route | `gpt-4o-mini` → OpenAI `gpt-4o-mini` |
| Provider `api_key_env` | `OPENAI_API_KEY` |

## Ports

| Port | Service |
|------|---------|
| 8080 | Gateway (inference) |
| 8081 | Control plane (platform + admin) |
| 5433 | Postgres |
| 5050 | Adminer |
| 3000 | Web UI |
| 8000 | MkDocs (`make doc-serve`) |
