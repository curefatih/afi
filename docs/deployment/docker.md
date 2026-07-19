# Docker Compose deployment

Run the full AFI stack (Postgres, Redis, control plane, gateway, worker, web) on a single host.

## Prerequisites

* Docker with Compose v2
* Outbound network for `docker pull` / builds (and upstream LLM APIs from the gateway)

## Quick start

```bash
# 1. Create local secret files from examples
make deploy-init

# 2. Edit secrets — replace every CHANGE_ME
#    deploy/.env          — Compose + runtime env
#    deploy/afi.yaml      — AFI YAML (seed + defaults)

# 3. Build images and start
make deploy-up

# 4. Health check
make deploy-health
```

`scripts/deploy-up.sh` refuses to start while `CHANGE_ME` placeholders remain.

## What gets started

| Service | Image / build | Default host port |
|---------|---------------|-------------------|
| `postgres` | `postgres:16-alpine` | `5432` |
| `redis` | `redis:7-alpine` | `6379` |
| `controlplane` | `Dockerfile` (`AFI_SERVICE=controlplane`) | `8081` |
| `gateway` | `Dockerfile` (`AFI_SERVICE=gateway`) | `8080` |
| `worker` | `Dockerfile` (`AFI_SERVICE=worker`) | — |
| `web` | `Dockerfile.web` (nginx + Vite build) | `3000` |

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
make build-images
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d web
```

## Day-2 operations

```bash
make deploy-logs          # follow all service logs
make deploy-down          # stop (keep volumes)
bash scripts/deploy-down.sh --volumes   # stop + wipe Postgres/Redis data
```

Publish a new config snapshot after platform changes (usually automatic from the UI). Manual:

```bash
curl -X POST http://localhost:8081/internal/v1/snapshots/publish \
  -H "X-AFI-Internal-Token: $AFI_INTERNAL_TOKEN"
```

## Optional: platform events with NATS

1. Uncomment the `nats` service in `deploy/docker-compose.yml`.
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
| Services | Postgres, Redis, Adminer | Full AFI stack |
| Postgres port | `5433` | `5432` (configurable) |
| App processes | Run via `make run-*` on the host | Containers |

Do not run both stacks against the same host ports at once.

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `deploy-up` exits on `CHANGE_ME` | Finish editing `deploy/.env` and `deploy/afi.yaml` |
| Control plane crash-loops | `AFI_DATABASE_URL`, Postgres healthy, logs |
| Gateway 401 | Virtual API key / snapshot published |
| Timed quotas fail | Redis up; `AFI_REDIS_URL` |
| Usage UI empty | Worker running; wait for outbox drain |
| Web calls wrong API host | Rebuild web with correct `VITE_*` |
