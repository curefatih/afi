# Federation lab (local Docker Compose)

Simulate a **home control plane** and a **regional control plane** that pull-syncs one region, plus a gateway in each place.

```text
┌──────────────── hub ────────────────┐     pull sync      ┌────────── regional ──────────┐
│ hub-controlplane (:8081) API        │◄───────────────────│ regional-controlplane (:8181)│
│ hub-postgres / hub-redis            │                    │ regional-postgres / redis    │
│ hub-gateway (:8080)  us-east        │                    │ regional-gateway (:8180)     │
│ web (:3000) → hub API               │                    │ regional-web (:3100) → CP    │
└─────────────────────────────────────┘                    └──────────────────────────────┘
```

`:8081` / `:8181` are **JSON APIs** (`/healthz`, `/api/v1/…`) — browsing `/` returns 404. Use the web UIs on `:3000` (hub) and `:3100` (regional).

Local-only secrets are intentional (same idea as `make quickstart`). Do not expose this stack publicly.

## Quick start

```bash
cd examples/federation
chmod +x up.sh down.sh bootstrap.sh
./up.sh
```

`up.sh` will:

1. Copy `env.example` → `.env` if needed
2. Build images and start the **hub** profile
3. Run `bootstrap.sh` (regions, bind-all, federation peer, deployments, snapshot publish)
4. Start the **regional** profile with generated `regional.secrets.env`
5. Recreate the hub gateway so it picks up its deployment join token

Tear down:

```bash
./down.sh          # keep volumes
./down.sh -v       # wipe Postgres data
```

## Ports

| Service | Host URL |
|---------|----------|
| Web UI (hub) | http://localhost:3000 |
| Web UI (regional) | http://localhost:3100 |
| Hub control plane (API) | http://localhost:8081 |
| Hub gateway | http://localhost:8080 |
| Regional control plane (API) | http://localhost:8181 |
| Regional gateway | http://localhost:8180 |

Default logins:

| Where | Email | Password |
|-------|-------|----------|
| Hub | `admin@afi.local` | `admin` |
| Regional CP UI | `regional@afi.local` | `admin` |

Hub seed virtual API key: `sk-federation-lab-hub-key` (works on both gateways after bind + pull).

## What bootstrap creates

On the **hub**:

| Resource | Slug / name | Purpose |
|----------|-------------|---------|
| Region | `us-east` | Home-region gateway deployment |
| Region | `eu-west` | Scoped to the federation peer + regional gateway |
| Memberships | bind-all on both regions | Seed org appears in both regional snapshots |
| Federation peer | `EU regional CP` | Join token → `AFI_FEDERATION_JOIN_TOKEN` |
| Deployment | `hub-gateway-us` | Hub gateway heartbeats (optional) |
| Deployment | `regional-gateway-eu` | Regional gateway heartbeats + usage to hub |

Secrets land in `regional.secrets.env` (gitignored via local `.env` patterns / untracked file).

## Manual walkthrough (UI)

If you prefer the platform UI over the script:

1. Start only the hub:  
   `docker compose --env-file .env --profile hub up -d`
2. Open http://localhost:3000 and sign in as hub admin.
3. **Regions** → create `us-east` and `eu-west` → **Bind all orgs** on each.
4. On each region, **Register deployment** (`hub-gateway-us`, `regional-gateway-eu`) and save join tokens.
5. **Federation** → **Register peer** for `eu-west`; copy the join token once.
6. Put the peer token and deployment IDs into `regional.secrets.env` (see `env.example` keys), then:  
   `docker compose --env-file .env --env-file regional.secrets.env --profile hub --profile regional up -d`
7. Wait one pull interval (`AFI_FEDERATION_PULL_INTERVAL`, default 10s in this lab).
8. Confirm peer **last sync** on the Federation page and `GET http://localhost:8180/healthz` shows a snapshot version.

## Manual walkthrough (API)

```bash
HUB=http://localhost:8081
TOKEN=$(curl -sS -X POST "$HUB/api/v1/platform/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@afi.local","password":"admin"}' | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')

# Create region
curl -sS -X POST "$HUB/api/v1/platform/regions" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"slug":"eu-west","name":"EU West"}'

# Bind all orgs (use region id from create/list)
curl -sS -X POST "$HUB/api/v1/platform/regions/<regionID>/organizations/bind-all" \
  -H "Authorization: Bearer $TOKEN"

# Register federation peer (join_token shown once)
curl -sS -X POST "$HUB/api/v1/platform/federation/peers" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"EU regional CP","region_id":"<regionID>"}'

# Register gateway deployment
curl -sS -X POST "$HUB/api/v1/platform/regions/<regionID>/deployments" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"regional-gateway-eu","public_base_url":"http://localhost:8180"}'
```

Regional control plane env (already wired in Compose once secrets exist):

```bash
AFI_FEDERATION_MODE=regional
AFI_FEDERATION_HUB_URL=http://hub-controlplane:8081
AFI_FEDERATION_JOIN_TOKEN=...
AFI_FEDERATION_REGION_SLUG=eu-west
AFI_FEDERATION_PULL_INTERVAL=10s
```

Gateways use `AFI_FEDERATION_MODE=off` (they only read local Postgres snapshots; the regional CP runs the puller).

## Verify pull sync

1. On the hub, unbind an org from `eu-west` (or change an overlay) and publish if needed.
2. Within one pull interval, regional CP applies the export (membership tables + local snapshot).
3. Hub Federation peer row shows **last sync** / cursor.
4. Regional gateway `healthz` snapshot version advances when the export embeds a new snapshot.

```bash
# Both gateways should list models for the seed key after bind-all + pull
curl -s http://localhost:8080/v1/models -H 'Authorization: Bearer sk-federation-lab-hub-key'
curl -s http://localhost:8180/v1/models -H 'Authorization: Bearer sk-federation-lab-hub-key'
```

Set `OPENAI_API_KEY` (or another provider key) in `.env` and recreate gateways for live chat.

## Compose profiles

| Profile | Services |
|---------|----------|
| `hub` | `hub-postgres`, `hub-redis`, `hub-controlplane`, `hub-gateway`, `web` |
| `regional` | `regional-postgres`, `regional-redis`, `regional-controlplane`, `regional-gateway`, `regional-web` |

Always start **hub** before **regional**. Regional mode refuses to start without `AFI_FEDERATION_JOIN_TOKEN`.

## Relation to production docs

- Single-host deploy: [Docker Compose](../../docs/deployment/docker.md)
- Regions + object-store spokes (Phase 2): [Deployment index — hub-and-spoke](../../docs/deployment/index.md#hub-and-spoke-multi-region-gateways)
- Multi-CP pull sync (Phase 3): [Deployment index — federation](../../docs/deployment/index.md#hub-and-regional-control-planes-federation-pull-sync)

This lab uses **postgres snapshots on each CP** (no MinIO). Object-store fan-out remains the pattern for spoke gateways that share one control plane without a regional CP.
