# Docker Compose deployment

Run AFI on a single host with Docker Compose. Start the **full** stack, or only the pieces you need (control plane, data plane / gateway, worker, web, or infra).

## Prerequisites

* Docker with Compose v2
* Outbound network for `docker pull` / builds (and upstream LLM APIs from the gateway)

## Try it first

For a local trial with fixed defaults (no secret editing):

```bash
make quickstart
```

See [Quick start](../getting-started/quick-start.md). Use the steps below when you want your own secrets on a real host.

## Setup

```bash
# 1. Create local secret files from examples
make deploy-init

# 2. Edit secrets — replace every CHANGE_ME
#    deploy/.env          — Compose + runtime env
#    deploy/afi.yaml      — AFI YAML (seed + defaults)

# 3. Build images and start (full stack)
make deploy-up

# 4. Health check
make deploy-health
```

`scripts/deploy-up.sh` refuses to start while `CHANGE_ME` placeholders remain.

## Deployment profiles

Every service is gated by a Compose **profile**. The default is `full`.

| Profile | Starts | When to use |
|---------|--------|-------------|
| `full` | Postgres, Redis, control plane, gateway, worker, web | Single-host / quick self-host (default) |
| `infra` | Postgres, Redis | Shared database for other hosts or local binaries |
| `controlplane` | Postgres, Redis, control plane | Platform API + seed/migrate only |
| `dataplane` | Postgres, Redis, gateway | Inference only (needs an existing snapshot in Postgres) |
| `worker` | Postgres, worker | Usage / events outbox drain only |
| `web` | Web UI | UI only — set `VITE_*` to reachable control plane / gateway URLs |

`gateway` is accepted as an alias for `dataplane`.

### Convenience targets

```bash
make deploy-up                 # full stack
make deploy-infra
make deploy-controlplane
make deploy-dataplane          # gateway + Postgres + Redis
make deploy-worker
make deploy-web
```

### Combine profiles

```bash
# Control plane + data plane + worker (no web UI)
make deploy-up PROFILE="controlplane dataplane worker"

# Or pass profiles to the script
bash scripts/deploy-up.sh controlplane dataplane
```

### Split across hosts

Typical pattern:

1. Host A: `make deploy-controlplane` (migrate, seed, publish snapshots).
2. Point Host B’s `AFI_DATABASE_URL` / `AFI_REDIS_URL` at that Postgres/Redis (or run `make deploy-infra` once and share it).
3. Host B: `make deploy-dataplane` and/or `make deploy-worker` with the same config secrets.
4. Optionally `make deploy-web` anywhere browsers can reach the APIs — rebuild after setting `VITE_PLATFORM_API_URL` / `VITE_GATEWAY_API_URL`.

App services no longer wait on each other at start: the gateway only needs a healthy Postgres (with a published snapshot) and Redis; the worker only needs Postgres.

## What gets started (`full`)

| Service | Image / build | Default host port | Profiles |
|---------|---------------|-------------------|----------|
| `postgres` | `postgres:16-alpine` | `5432` | `full`, `infra`, `controlplane`, `dataplane`, `worker` |
| `redis` | `redis:7-alpine` | `6379` | `full`, `infra`, `controlplane`, `dataplane` |
| `controlplane` | `Dockerfile` (`AFI_SERVICE=controlplane`) | `8081` | `full`, `controlplane` |
| `gateway` | `Dockerfile` (`AFI_SERVICE=gateway`) | `8080` | `full`, `dataplane` |
| `worker` | `Dockerfile` (`AFI_SERVICE=worker`) | — | `full`, `worker` |
| `web` | `Dockerfile.web` (nginx + Vite build) | `3000` | `full`, `web` |

Compose file: [`deploy/docker-compose.yml`](../../deploy/docker-compose.yml).

## Configuration

### Files

| File | Role |
|------|------|
| `deploy/env.example` → `deploy/.env` | Compose substitution + container env (secrets) |
| `deploy/afi.example.yaml` → `deploy/afi.yaml` | Mounted at `/config/afi.yaml` (`AFI_CONFIG`) |

Both `deploy/.env` and `deploy/afi.yaml` are gitignored.

### Must customize

1. `POSTGRES_PASSWORD` and matching `AFI_DATABASE_URL`
2. `AFI_JWT_SECRET`, `AFI_INTERNAL_TOKEN`
3. Seed block in `afi.yaml` (admin credentials + virtual API key)
4. Provider keys: `OPENAI_API_KEY` / `ANTHROPIC_API_KEY` / `GEMINI_API_KEY` (gateway)
5. Web URLs: `VITE_PLATFORM_API_URL`, `VITE_GATEWAY_API_URL` (browser-reachable host/ports)

Full list: [Customization reference](customization.md).

### Rebuild web after URL changes

Vite vars are baked at image build time:

```bash
# edit VITE_* in deploy/.env
make build-images PROFILE=web
docker compose -f deploy/docker-compose.yml --env-file deploy/.env --profile web up -d web
```

## Day-2 operations

```bash
make deploy-logs                    # follow logs (respects PROFILE)
make deploy-logs PROFILE=dataplane
make deploy-down                    # stop (keep volumes)
bash scripts/deploy-down.sh --volumes   # stop + wipe Postgres/Redis data
```

Publish a new config snapshot after platform changes (usually automatic from the UI). Manual:

```bash
curl -X POST http://localhost:8081/internal/v1/snapshots/publish \
  -H "X-AFI-Internal-Token: $AFI_INTERNAL_TOKEN"
```

## Optional: platform events with NATS

1. Uncomment the `nats` service in `deploy/docker-compose.yml` (add a `profiles` entry such as `["full"]`).
2. Set in `deploy/.env` / YAML:

```bash
AFI_EVENTS_OUTBOX_ENABLED=true
AFI_EVENTS_PUBLISHER=nats
AFI_EVENTS_NATS_URL=nats://nats:4222
```

3. Restart control plane and worker.

See [Platform domain events](../development/platform-events.md).

## Local vs deploy Compose

| | Dev (`make dev-up`) | Deploy (`make deploy-up`) |
|--|---------------------|---------------------------|
| File | `docker-compose.yml` | `deploy/docker-compose.yml` |
| Services | Postgres, Redis, Adminer | Profile-selected AFI services |
| Postgres port | `5433` | `5432` (configurable) |
| App processes | Run via `make run-*` on the host | Containers |

Do not run both stacks against the same host ports at once.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `deploy-up` exits on `CHANGE_ME` | Finish editing `deploy/.env` and `deploy/afi.yaml` |
| No containers start | Pass a profile (`full`, `controlplane`, …); bare Compose with no profile starts nothing |
| Control plane crash-loops | `AFI_DATABASE_URL`, Postgres healthy, logs |
| Gateway 401 / empty snapshot | Run control plane once to seed + publish; confirm shared Postgres |
| Timed quotas fail | Redis up; `AFI_REDIS_URL` |
| Usage UI empty | Worker running; wait for outbox drain |
| Web calls wrong API host | Rebuild web with correct `VITE_*` |
