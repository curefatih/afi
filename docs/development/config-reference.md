# Config reference

Development-oriented summary of ports, seed values, and platform APIs. For a complete operator customization list (every env var, YAML key, and deploy knob), see **[Customization reference](../deployment/customization.md)**. For self-hosting, see **[Deployment](../deployment.md)**.

## File

[`configs/local.yaml`](../../configs/local.yaml) — defaults for local development.

## Environment variables

Full operator table (defaults, required vs optional, which process): [Customization reference](../deployment/customization.md).

| Variable | Default | Used by |
|----------|---------|---------|
| `AFI_CONFIG` | `configs/local.yaml` | controlplane, gateway, cli |
| `AFI_DATABASE_URL` | from yaml / compose DSN | all DB clients |
| `AFI_REDIS_URL` | `redis://localhost:6379/0` | gateway timed quotas |
| `AFI_CONTROLPLANE_ADDR` | `:8081` | controlplane |
| `AFI_GATEWAY_ADDR` | `:8080` | gateway |
| `AFI_SNAPSHOT_POLL_INTERVAL` | `2s` | gateway watch |
| `AFI_JWT_SECRET` | from yaml | controlplane auth |
| `AFI_INTERNAL_TOKEN` | from yaml (`afi-local-internal-token`) | HTTP `/internal/v1/*` |
| `AFI_TOKEN_TTL` | `24h` | JWT lifetime |
| `OPENAI_API_KEY` | _(required for OpenAI live calls)_ | gateway → OpenAI |
| `ANTHROPIC_API_KEY` | _(required for Anthropic routes)_ | gateway → Anthropic |
| `GEMINI_API_KEY` | _(required for Gemini routes)_ | gateway → Gemini |
| `VITE_PLATFORM_API_URL` | `http://localhost:8081` | web UI (platform APIs) |
| `VITE_GATEWAY_API_URL` | `http://localhost:8080` | web playground |
| `VITE_GATEWAY_API_KEY` | seed virtual key | web playground |
| `AFI_EVENTS_OUTBOX_ENABLED` | `false` | enqueue platform events to Postgres |
| `AFI_EVENTS_PUBLISHER` | `log` | worker: `log` \| `nats` \| `kafka` \| `noop` |
| `AFI_EVENTS_NATS_URL` | `nats://127.0.0.1:4222` | NATS JetStream |
| `AFI_EVENTS_NATS_STREAM` | `AFI_PLATFORM` | JetStream stream name |
| `AFI_EVENTS_NATS_SUBJECT_PREFIX` | `afi.platform` | subject prefix |
| `AFI_EVENTS_KAFKA_BROKERS` | `127.0.0.1:9092` | Kafka brokers (CSV) |
| `AFI_EVENTS_KAFKA_TOPIC` | `afi.platform.events` | Kafka topic |
| `AFI_MAIL_PUBLIC_APP_URL` | `http://localhost:3000` | Invite accept links (web origin) |
| `AFI_MAIL_FROM` | `AFI <noreply@afi.local>` | From header |
| `AFI_MAIL_DEFAULT_PROVIDER` | `log` | `log` \| `smtp` \| `resend` |
| `AFI_MAIL_SMTP_ENABLED` | `false` | Enable SMTP transport |
| `AFI_MAIL_SMTP_HOST` / `PORT` | `localhost` / `1025` | SMTP (Mailpit/Mailhog) |
| `AFI_MAIL_RESEND_ENABLED` | `false` | Enable Resend API |
| `AFI_MAIL_RESEND_API_KEY` | _(empty)_ | Resend API key |

See [Platform domain events](platform-events.md) for the outbox flow.

## Internal admin HTTP

`POST /internal/v1/seed` and `POST /internal/v1/snapshots/publish` require header:

```http
X-AFI-Internal-Token: <auth.internal_token>
```

The CLI (`afi seed`, `afi snapshot publish`) and control-plane startup call seed/publish **in-process** and do not need this header.

## Destructive reset

```bash
afi db reset
```

Drops all AFI tables after typing `reset`. Use only for local recovery. Schema version bumps do **not** wipe data; only legacy UUID installs or incomplete schemas are auto-wiped once.

## Seed values

Written on first control-plane start (or `make seed`):

| Item | Value |
|------|--------|
| Virtual API key | `sk-project-local-dev-token-12345` (stored hashed in DB/snapshot) |
| Platform email | `admin@afi.local` |
| Platform password | `admin` |
| Default model route | `gpt-4o-mini` → OpenAI `gpt-4o-mini` |
| Seeded audio routes | `tts-1`, `whisper-1` → `prov_openai` |
| Seeded providers | `prov_openai`, `prov_anthropic`, `prov_gemini`, `prov_ollama` (`openai_compatible` → `http://127.0.0.1:11434/v1`, no default route) |
| `OLLAMA_API_KEY` | _(any value if Ollama ignores auth)_ | gateway → openai_compatible |
| Route `fallbacks` | optional ordered `[{provider_id,target_model}]` for 5xx/timeout/429 retry |
| Gateway models | `GET /v1/models` lists org route model ids (`supports_streaming` / `supports_tts` / `supports_stt`) |
| Gateway audio | `POST /v1/audio/speech`, `POST /v1/audio/transcriptions` (openai / openai_compatible) |

## Ports

| Port | Service |
|------|---------|
| 8080 | Gateway (inference) |
| 8081 | Control plane (platform + admin) |
| 5433 | Postgres |
| 6379 | Redis (timed quota windows) |
| 5050 | Adminer |
| 3000 | Web UI |
| 8000 | MkDocs (`make doc-serve`) |

## Platform config APIs

| Method | Path |
|--------|------|
| GET/POST | `/api/v1/platform/organizations` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/members` (POST = invite; org admin) |
| PATCH | `/api/v1/platform/organizations/{orgID}/members/{userID}` |
| GET/DELETE | `/api/v1/platform/organizations/{orgID}/invites[/{inviteID}]` |
| POST | `/api/v1/platform/organizations/{orgID}/invites/{inviteID}/resend` |
| GET/PATCH | `/api/v1/platform/organizations/{orgID}/mail` |
| POST | `/api/v1/platform/organizations/{orgID}/mail/test` |
| GET/POST | `/api/v1/platform/auth/invites/{token}` / `…/accept` (public) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/keys` |
| DELETE | `/api/v1/platform/keys/{keyID}` |
| GET/POST | `/api/v1/platform/projects/{projectID}/keys` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/providers` |
| GET | `/api/v1/platform/organizations/{orgID}/providers/health` |
| PATCH/DELETE | `/api/v1/platform/providers/{providerID}` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/routes` |
| PATCH/DELETE | `/api/v1/platform/routes/{routeID}` |
| GET | `/api/v1/platform/organizations/{orgID}/usage` |
| GET | `/api/v1/platform/organizations/{orgID}/usage/summary` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/quotas` |
| PATCH/DELETE | `/api/v1/platform/quotas/{quotaID}` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/policies` |
| PATCH/DELETE | `/api/v1/platform/policies/{policyID}` |

Member invite (org admin): existing users are added and emailed; unknown emails get a pending invite + accept link (`/auth/invite/{token}`). Mail transports: `log` (default local), `smtp`, `resend` — org admins pick among enabled providers in Settings. Org roles: `owner` / `admin` / `member`. Only the **owner** can `PATCH` a member role (`{ "role": "admin" }`); setting `owner` transfers ownership. Native Anthropic inference: gateway `POST /v1/messages` (Anthropic providers only).

### Usage

| Query | Notes |
|-------|--------|
| `limit`, `project_id`, `api_key_id`, `model`, `modality`, `from`, `to` | List + summary filters (`from`/`to` as RFC3339 or `YYYY-MM-DD`) |
| `group_by` | Summary only: `day` (default), `model`, `key`, `modality` |

Each event has `modality` (`chat`, `messages`, `tts`, `stt`, …), extensible `metrics` JSON (e.g. TTS `characters`), optional token fields for chat, and key owner fields (`key_name`, `key_kind`, `owner_email`, …). Personal keys attribute to the owner user; service-account keys have no human owner.

### API keys

| `kind` | Scope | Who can create |
|--------|--------|----------------|
| `personal` | org + `owner_user_id` (no project) | any org member (self only) |
| `service_account` | org-wide or project (`project_id` optional) | org owner/admin |

Seed key `sk-project-local-dev-token-12345` is a project **service_account** key.

### Quotas

| Field | Values |
|-------|--------|
| `scope_type` | `organization`, `project`, `user`, `api_key` |
| `metric` | `requests`, `tokens` |
| `window` | `total` (Postgres lifetime), `minute` / `hour` / `day` (Redis fixed windows) |
| `limit_value` | integer ≥ 0 (`0` blocks immediately) |

Most specific scope wins **per window**: api_key → user → project → organization. Timed windows require Redis (`redis_url` / `AFI_REDIS_URL`). Create/update/delete quotas require org owner/admin.

### CEL policies

| Field | Notes |
|-------|--------|
| `name` | Display name |
| `expression` | Boolean CEL; all enabled org policies must be true |
| `priority` | Higher first (default 100) |
| `enabled` | Default true |

CEL variables: `request.model`, `request.path`, `request.stream`, `key.id`, `key.organization_id`, `key.project_id`, `key.kind`, `key.owner_user_id`, `key.name`. Denial → HTTP 403 `policy_violation`. Owner/admin to create/update/delete.
