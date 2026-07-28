# Quick start

Run the full AFI stack (Postgres, Redis, control plane, gateway, worker, and web UI) with Docker Compose. No Go or Node toolchain required.

For day-to-day development on the host, see [Local development](local-dev.md). For production hardening or running only the control plane / data plane / worker / web, see [Docker Compose deployment](../deployment/docker.md).

## Prerequisites

* [Docker](https://docs.docker.com/get-docker/) with Compose v2
* Optional: an OpenAI API key (or Anthropic / Gemini) for live inference

## 1. Clone and start

```bash
git clone https://github.com/curefatih/afi.git
cd afi

# Optional — bake a provider key into deploy/.env
export OPENAI_API_KEY="sk-..."

make quickstart
```

What this does:

1. Writes local-only defaults to `deploy/.env` and `deploy/afi.yaml` (from [`deploy/quickstart.env`](../../deploy/quickstart.env) and [`deploy/quickstart.afi.yaml`](../../deploy/quickstart.afi.yaml))
2. Builds container images
3. Starts the stack in the background

First build can take several minutes. Existing `deploy/.env` / `deploy/afi.yaml` files are kept unless they still contain `CHANGE_ME` placeholders.

## 2. Check health

```bash
make deploy-health
```

Or manually:

```bash
curl -s http://localhost:8081/healthz
curl -s http://localhost:8080/healthz
```

Expect `"status":"ok"` (the gateway also reports `snapshot_version`).

| Service | URL |
|---------|-----|
| Web UI | http://localhost:3000 |
| Control plane | http://localhost:8081 |
| Gateway | http://localhost:8080 |
| Postgres | `localhost:5432` |
| Redis | `localhost:6379` |

## 3. Sign in to the UI

Open http://localhost:3000 and sign in with:

| Item | Value |
|------|--------|
| Email | `admin@afi.local` |
| Password | `admin` |

Use **Providers**, **Routing**, and **Keys** to explore the console. Chat / TTS / STT playgrounds call the gateway at `:8080`.

## 4. Call the gateway

Use the seeded virtual API key (same as local development):

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "ping"}]
  }'
```

Requires `OPENAI_API_KEY` in `deploy/.env` (or set via `export` before `make quickstart`). To add a key after the stack is already up:

```bash
# edit OPENAI_API_KEY=... in deploy/.env, then:
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d gateway
```

More checks: [Verify](verify.md).

## Day-2 commands

```bash
make deploy-logs     # follow logs
make deploy-health   # health probes
make deploy-down     # stop (keep volumes)
bash scripts/deploy-down.sh --volumes   # stop and wipe Postgres/Redis data
```

## What gets started

| Service | Role | Host port |
|---------|------|-----------|
| `postgres` | Config, snapshots, outboxes | `5432` |
| `redis` | Timed quota windows | `6379` |
| `controlplane` | Platform API, seed, snapshots | `8081` |
| `gateway` | Inference pipeline | `8080` |
| `worker` | Usage outbox drain | — |
| `web` | Platform console | `3000` |

Compose file: [`deploy/docker-compose.yml`](../../deploy/docker-compose.yml).

## Local-only credentials

Quick start uses fixed secrets so the docs stay copy-pasteable. **Do not expose this stack on a public network** without replacing them.

| Secret | Value |
|--------|--------|
| Platform admin | `admin@afi.local` / `admin` |
| Virtual API key | `sk-project-local-dev-token-12345` |
| Internal token | `afi-quickstart-internal-token` |
| Postgres password | `afi-quickstart-db` |

To harden or change ports, URLs, and seed values, see [Customization](../deployment/customization.md) and [Docker Compose deployment](../deployment/docker.md).

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Port already in use | Change `*_HOST_PORT` in `deploy/.env`, then `make deploy-up` |
| Gateway returns 401 | Use the seeded key above, or create a key in the UI |
| Chat fails / upstream errors | Set `OPENAI_API_KEY` (or another provider key) and restart `gateway` |
| Web UI calls wrong host | Rebuild web after editing `VITE_*` — see [Docker Compose](../deployment/docker.md) |
| Want a clean DB | `bash scripts/deploy-down.sh --volumes` then `make quickstart` |

## Next steps

* [Web UI](web-ui.md) — console overview
* [API dialects](../api/dialects.md) — OpenAI / Anthropic / Gemini paths
* [Local development](local-dev.md) — run services on the host with Go
* [Deployment](../deployment.md) — production checklist
