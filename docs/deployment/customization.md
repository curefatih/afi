# Customization reference

This page lists **every operator-facing customization** for a self-hosted AFI deployment. For local defaults see [`configs/local.yaml`](../../configs/local.yaml). For deploy templates see [`deploy/afi.example.yaml`](../../deploy/afi.example.yaml) and [`deploy/env.example`](../../deploy/env.example).

## How configuration is loaded

1. Path from `AFI_CONFIG` (default `configs/local.yaml`).
2. If the file exists, YAML is loaded; **environment variables override YAML** for keys that have `env:` tags.
3. If the file is missing, config is read from the environment only, then code defaults apply.
4. **Seed** fields are YAML-only — they are **not** overridden by `AFI_*` env vars.

Loader: `internal/kernel/config.go` (cleanenv).

```bash
export AFI_CONFIG=/etc/afi/afi.yaml
export AFI_JWT_SECRET="..."   # overrides auth.jwt_secret in the file
```

---

## Core services (`AFI_*` and YAML)

| Env var | YAML key | Default | Used by | Notes |
|---------|----------|---------|---------|-------|
| `AFI_CONFIG` | — | `configs/local.yaml` | All | Config file path |
| `AFI_DATABASE_URL` | `database_url` | `postgres://afi:afi@localhost:5433/afi?sslmode=disable` | CP, GW, worker, CLI | **Required** in production |
| `AFI_REDIS_URL` | `redis_url` | `redis://localhost:6379/0` | Gateway, control plane | Gateway timed quotas; SSO CSRF state when `auth.sso.state_store=redis` |
| `AFI_CONTROLPLANE_ADDR` | `controlplane.addr` | `:8081` | Control plane | Listen address |
| `AFI_GATEWAY_ADDR` | `gateway.addr` | `:8080` | Gateway | Listen address |
| `AFI_SNAPSHOT_POLL_INTERVAL` | `gateway.snapshot_poll_interval` | `2s` | Gateway | Poll period (also uses Postgres `LISTEN`) |
| `AFI_JWT_SECRET` | `auth.jwt_secret` | `afi-local-dev-jwt-secret-change-me` | Control plane | **Change in prod** — HS256 signing |
| `AFI_TOKEN_TTL` | `auth.token_ttl` | `24h` | Control plane | Platform session JWT lifetime |
| `AFI_INTERNAL_TOKEN` | `auth.internal_token` | `afi-local-internal-token` | Control plane | Header `X-AFI-Internal-Token` for `/internal/v1/*` |
| `AFI_AUTH_PUBLIC_BASE_URL` | `auth.public_base_url` | `http://localhost:8081` | Control plane | Public control-plane URL (SSO callbacks) |
| `AFI_SSO_ENABLED` | `auth.sso.enabled` | `false` | Control plane | Enable platform OAuth2/OIDC SSO |
| `AFI_SSO_STATE_STORE` | `auth.sso.state_store` | `redis` | Control plane | `redis` (multi-node) or `memory` (single-node) |

### Auth behavior

* Platform login: `POST /api/v1/platform/auth/login` → JWT signed with `jwt_secret`.
* Platform SSO (OAuth2/OIDC): see **[Single sign-on (SSO)](../getting-started/sso.md)**. Providers are configured under `auth.sso.providers` (YAML). Prefer `state_store: redis` when running multiple control-plane replicas.
* Empty `internal_token` fails closed for HTTP internal admin routes.
* CLI `afi seed` / `afi snapshot publish` run in-process and do not need the header.

---

## Seed block (YAML only)

Applied when the database is empty (first control-plane start, or `afi seed` / `make seed`).

| YAML key | Default | Purpose |
|----------|---------|---------|
| `seed.virtual_api_key` | `sk-project-local-dev-token-12345` | Seeded project service-account key (stored hashed) |
| `seed.admin_email` | `admin@afi.local` | First platform user |
| `seed.admin_password` | `admin` | Password (bcrypt in DB) |
| `seed.admin_name` | `Admin` | Display name |
| `seed.openai_base_url` | `https://api.openai.com/v1` | Seeded OpenAI provider base URL |
| `seed.openai_api_key_env` | `OPENAI_API_KEY` | **Name** of the env var the gateway should read (not the secret itself) |
| `seed.default_model` | `gpt-4o-mini` | Default chat route model id |

Also seeded in code (not YAML-configurable): Anthropic, Gemini, Ollama (`openai_compatible` → `http://127.0.0.1:11434/v1`), TTS/STT routes, echo provider/route. Customize further via the platform UI/API after first boot.

---

## Platform domain events

| Env var | YAML key | Default | Notes |
|---------|----------|---------|-------|
| `AFI_EVENTS_OUTBOX_ENABLED` | `events.outbox_enabled` | `false` | Enable on **control plane** (enqueue) and **worker** (drain) |
| `AFI_EVENTS_PUBLISHER` | `events.publisher` | `log` | `log` \| `nats` \| `kafka` \| `noop` |
| `AFI_EVENTS_NATS_URL` | `events.nats.url` | `nats://127.0.0.1:4222` | When `publisher=nats` |
| `AFI_EVENTS_NATS_STREAM` | `events.nats.stream` | `AFI_PLATFORM` | JetStream stream |
| `AFI_EVENTS_NATS_SUBJECT_PREFIX` | `events.nats.subject_prefix` | `afi.platform` | Subjects: `{prefix}.{event.name}` |
| `AFI_EVENTS_KAFKA_BROKERS` | `events.kafka.brokers` | `127.0.0.1:9092` | CSV broker list |
| `AFI_EVENTS_KAFKA_TOPIC` | `events.kafka.topic` | `afi.platform.events` | Topic name |

Details: [Platform domain events](../development/platform-events.md).

---

## Upstream provider secrets (gateway process)

Provider rows store an **environment variable name** (`api_key_env`), not the secret. The gateway resolves secrets at runtime via `os.Getenv`.

| Typical env var | Default for provider type | When needed |
|-----------------|---------------------------|-------------|
| `OPENAI_API_KEY` | `openai` | OpenAI / compatible routes |
| `ANTHROPIC_API_KEY` | `anthropic` | Anthropic routes |
| `GEMINI_API_KEY` | `gemini` | Gemini routes |
| `OLLAMA_API_KEY` | `openai_compatible` | Any non-empty value if the backend ignores auth |
| `ECHO_UNUSED` | `echo` | Echo extension (unused) |
| *custom* | whatever you set on the provider | Must exist in the **gateway** environment |

You can rename `api_key_env` per provider in the UI/API; inject that exact name into the gateway container/process.

---

## Web UI (Vite build-time)

These are compiled into the static bundle. Changing them requires a **rebuild** of the web image/assets.

| Variable | Default | Purpose |
|----------|---------|---------|
| `VITE_PLATFORM_API_URL` | `http://localhost:8081` | Control plane base URL (browser-reachable) |
| `VITE_GATEWAY_API_URL` | `http://localhost:8080` | Gateway base URL for playground |
| `VITE_GATEWAY_API_KEY` | seed virtual key | Default playground key |

Example: [`web/.env.example`](../../web/.env.example). In Compose, set these in `deploy/.env` before `make deploy-up` / `build-images`.

Use public hostnames or published host ports — **not** Docker service names like `http://controlplane:8081` (browsers cannot resolve those).

---

## Compose / infra customization

From [`deploy/env.example`](../../deploy/env.example):

| Variable | Default | Purpose |
|----------|---------|---------|
| `COMPOSE_PROJECT_NAME` | `afi` | Docker Compose project name |
| `POSTGRES_USER` / `PASSWORD` / `DB` | `afi` / *(required)* / `afi` | Database bootstrap |
| `POSTGRES_HOST_PORT` | `5432` | Host → container `5432` |
| `REDIS_HOST_PORT` | `6379` | Host Redis port |
| `CONTROLPLANE_HOST_PORT` | `8081` | Published control plane port |
| `GATEWAY_HOST_PORT` | `8080` | Published gateway port |
| `WEB_HOST_PORT` | `3000` | Published web UI port |

Root [`docker-compose.yml`](../../docker-compose.yml) (dev infra only) uses Postgres host port **5433** and includes Adminer on **5050**.

---

## Verify / ops script overrides

| Variable | Default | Purpose |
|----------|---------|---------|
| `AFI_CONTROLPLANE_URL` | `http://localhost:8081` | `scripts/verify-local.sh`, `scripts/deploy-health.sh` |
| `AFI_GATEWAY_URL` | `http://localhost:8080` | same |
| `AFI_WEB_URL` | _(empty)_ | Optional web check in `deploy-health.sh` |
| `AFI_INTERNAL_TOKEN` | `afi-local-internal-token` | verify script internal calls |
| `AFI_VIRTUAL_API_KEY` | seed key | verify script gateway auth |

Release builds:

| Variable | Default | Purpose |
|----------|---------|---------|
| `GOOS` / `GOARCH` | `linux` / `amd64` | `scripts/build-release.sh` target |
| `VERSION` | git describe | Label printed by the script |

---

## Platform runtime customization (API / UI)

These are **not** env vars — configure them after deploy via the control plane:

| Area | Customizable fields |
|------|---------------------|
| **Organizations / members** | Orgs; roles `owner` / `admin` / `member` (invite by existing email — no SMTP) |
| **API keys** | `personal` or `service_account`; project scope optional |
| **Providers** | type, base URL, `api_key_env`, health |
| **Routes** | model id, provider, target model, modality, ordered `fallbacks` |
| **Quotas** | scope (`organization`/`project`/`user`/`api_key`), metric (`requests`/`tokens`), window (`total`/`minute`/`hour`/`day`), `limit_value` |
| **CEL policies** | `expression`, `priority`, `enabled` |
| **Model prices** | DB overrides for usage `cost_usd` (else embedded catalog) |

API surface summary: [Config reference](../development/config-reference.md).

---

## Model catalog

* Embedded at build time: `internal/modelcatalog/catalog.json`.
* No runtime file/env override — change catalog → rebuild binaries.
* Usage cost: `model_prices` in Postgres overrides catalog entries.

---

## Hardcoded runtime knobs (not configurable today)

| Knob | Value |
|------|-------|
| Worker poll interval | `2s` |
| Outbox claim batch size | `50` |
| HTTP `ReadHeaderTimeout` | `10s` |
| Graceful shutdown timeout | `10s` |
| Snapshot NOTIFY channel | `afi_snapshot` |
| CORS | `Access-Control-Allow-Origin: *` |

If you need different values, open an issue or fork and change the corresponding `cmd/` / `internal/` constants.

---

## Example production YAML sketch

```yaml
database_url: "postgres://afi:STRONG@db.internal:5432/afi?sslmode=require"
redis_url: "redis://:STRONG@redis.internal:6379/0"

controlplane:
  addr: ":8081"

gateway:
  addr: ":8080"
  snapshot_poll_interval: "2s"

auth:
  jwt_secret: "long-random-string"
  token_ttl: "12h"
  internal_token: "another-long-random-string"
  public_base_url: "https://afi-control.example.com"
  sso:
    enabled: false
    state_store: redis
    # providers: see getting-started/sso.md

seed:
  virtual_api_key: "sk-live-initial-only"
  admin_email: "ops@example.com"
  admin_password: "strong-password"
  admin_name: "Ops"
  openai_base_url: "https://api.openai.com/v1"
  openai_api_key_env: "OPENAI_API_KEY"
  default_model: "gpt-4o-mini"

events:
  outbox_enabled: true
  publisher: nats
  nats:
    url: "nats://nats.internal:4222"
    stream: AFI_PLATFORM
    subject_prefix: afi.platform
```

Prefer putting secrets in the process environment (`AFI_JWT_SECRET`, `AFI_DATABASE_URL`, `OPENAI_API_KEY`, …) rather than committing them to the YAML file.
