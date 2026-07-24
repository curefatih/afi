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
| `AFI_REDIS_URL` | `redis://localhost:6379/0` | Gateway timed quotas; control plane SSO state when `auth.sso.state_store=redis` |
| `AFI_CONTROLPLANE_ADDR` | `:8081` | controlplane |
| `AFI_GATEWAY_ADDR` | `:8080` | gateway |
| `AFI_SNAPSHOT_POLL_INTERVAL` | `2s` | gateway watch |
| `AFI_WASM_BEFORE_CALL` | _(empty)_ | gateway — optional TinyGo `.wasm` path (`before_call`) |
| `AFI_WASM_BEFORE_CHAT` | _(empty)_ | gateway — optional TinyGo `.wasm` path (`before_chat`) |
| `AFI_WASM_S3_*` | _(empty)_ | gateway — S3-compatible fetch for `s3://` wasm module URIs |
| `AFI_PLUGIN_SOCK` | _(set by gateway)_ | gRPC plugin process — unix socket path when spawned via `gateway.grpc_extensions[].command` |
| `AFI_JWT_SECRET` | from yaml | controlplane auth |
| `AFI_INTERNAL_TOKEN` | from yaml (`afi-local-internal-token`) | HTTP `/internal/v1/*` |
| `AFI_TOKEN_TTL` | `24h` | JWT lifetime |
| `AFI_AUTH_PUBLIC_BASE_URL` | `http://localhost:8081` | SSO OAuth callback base URL |
| `AFI_SIGNUP_ENABLED` | `false` | Allow self-serve `POST /auth/register` |
| `AFI_SSO_ENABLED` | `false` | Enable platform SSO |
| `AFI_SSO_STATE_STORE` | `redis` | `redis` \| `memory` |
| `AFI_CREDENTIALS_MASTER_KEY` | from yaml (`credentials.master_key`) | controlplane + gateway (encrypted_db credentials) |
| `AFI_SECRETS_AWS_SM_ENABLED` | `false` | gateway: enable AWS Secrets Manager for `storage_kind=vault` |
| `AFI_SECRETS_AWS_SM_REGION` | _(AWS default)_ | default region for aws-sm refs |
| `AFI_SECRETS_VAULT_ADDR` / `VAULT_ADDR` | _(empty)_ | HashiCorp Vault address |
| `AFI_SECRETS_VAULT_TOKEN` / `VAULT_TOKEN` | _(empty)_ | HashiCorp Vault token |
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
| `AFI_TELEMETRY_ENABLED` | `false` | OpenTelemetry metrics + traces |
| `AFI_TELEMETRY_OTLP_ENDPOINT` | _(empty)_ | OTLP host:port (e.g. `127.0.0.1:4318`) |
| `AFI_TELEMETRY_OTLP_PROTOCOL` | `http` | `http` \| `grpc` |
| `AFI_TELEMETRY_OTLP_INSECURE` | `false` | Disable TLS for local OTLP |
| `AFI_TELEMETRY_METRICS_PROMETHEUS` | `false` | Expose `GET /metrics` on gateway/CP |
| `AFI_MAIL_PUBLIC_APP_URL` | `http://localhost:3000` | Invite accept links (web origin) |
| `AFI_MAIL_FROM` | `AFI <noreply@afi.local>` | From header |
| `AFI_MAIL_DEFAULT_PROVIDER` | `log` | `log` \| `smtp` \| `resend` |
| `AFI_MAIL_SMTP_ENABLED` | `false` | Enable SMTP transport |
| `AFI_MAIL_SMTP_HOST` / `PORT` | `localhost` / `1025` | SMTP (Mailpit/Mailhog) |
| `AFI_MAIL_RESEND_ENABLED` | `false` | Enable Resend API |
| `AFI_MAIL_RESEND_API_KEY` | _(empty)_ | Resend API key |

See [Platform domain events](platform-events.md) for the outbox flow. See [Observability](../deployment/observability.md) for OTel / Grafana.

### YAML-only: gRPC extensions

```yaml
gateway:
  grpc_extensions:
    - id: grpcecho
      command: ["./bin/grpcecho"]
    # - id: remote
    #   address: unix:///tmp/plugin.sock
    #   provider_type: optional-override
```

See [providers.md](providers.md#grpc-extensions-process-isolated) and [`extensions/grpcecho`](../../extensions/grpcecho).

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
| Route `fallbacks` | optional `[{provider_id,target_model,weight?}]` for 5xx/timeout/429 failover (list order after first pick) |
| Route `routing_strategy` | `ordered` (default), `weighted`, `latency`, or `cost` — first-target selection before retry/failover |
| Route `weight` | primary target weight for `weighted` (default `1`); each fallback may set `weight` |
| Route `retry` | optional per-route override; else org `default_retry` from Settings → General |
| Org `default_retry` | optional org-wide `{max_attempts,backoff…}` applied when a route has no `retry` |
| Gateway models | `GET /v1/models` lists org route model ids (`supports_streaming` / `supports_tts` / `supports_stt`) |
| Gateway audio | `POST /v1/audio/speech`, `POST /v1/audio/transcriptions` (openai / openai_compatible) |
| Gateway embeddings | `POST /v1/embeddings` (openai / openai_compatible) |
| Gateway images | `POST /v1/images/generations` (openai / openai_compatible; optional org object-store persist) |

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

Canonical contract: OpenAPI [`api/openapi/platform.openapi.yaml`](../../api/openapi/platform.openapi.yaml) and docs [Platform API](../api/platform.md). Summary of common routes:

| Method | Path |
|--------|------|
| GET/POST | `/api/v1/platform/organizations` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/members` (POST = invite; org admin) |
| PATCH | `/api/v1/platform/organizations/{orgID}/members/{userID}` (owner only) |
| GET/DELETE | `/api/v1/platform/organizations/{orgID}/invites[/{inviteID}]` (org admin) |
| POST | `/api/v1/platform/organizations/{orgID}/invites/{inviteID}/resend` (org admin) |
| GET/PATCH | `/api/v1/platform/organizations/{orgID}/mail` (org admin) |
| POST | `/api/v1/platform/organizations/{orgID}/mail/test` (org admin) |
| GET/POST | `/api/v1/platform/auth/invites/{token}` / `…/accept` (public) |
| GET | `/api/v1/platform/auth/sso/providers` (public) |
| GET | `/api/v1/platform/auth/sso/{provider}/start` (public; redirects to IdP) |
| GET | `/api/v1/platform/auth/sso/{provider}/callback` (public; OAuth/OIDC → web UI) |
| POST | `/api/v1/platform/auth/sso/{provider}/callback` (public; SAML ACS → web UI) |
| GET | `/api/v1/platform/auth/sso/{provider}/metadata` (public; SAML SP metadata) |
| GET/PUT | `/api/v1/platform/organizations/{orgID}/default-retry` (PUT = org admin; publishes snapshot) |
| GET/PUT | `/api/v1/platform/organizations/{orgID}/object-store` (PUT = org admin; publishes snapshot; optional image asset persistence) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/keys` (personal = member; service_account = org admin) |
| DELETE | `/api/v1/platform/keys/{keyID}` (admin or personal key owner) |
| GET/POST | `/api/v1/platform/projects/{projectID}/keys` (POST = org admin) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/providers` (POST = org admin) |
| GET | `/api/v1/platform/organizations/{orgID}/providers/health` |
| PATCH/DELETE | `/api/v1/platform/providers/{providerID}` (org admin) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/routes` (POST = org admin) |
| PATCH/DELETE | `/api/v1/platform/routes/{routeID}` (org admin) |
| GET | `/api/v1/platform/organizations/{orgID}/usage` |
| GET | `/api/v1/platform/organizations/{orgID}/usage/summary` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/quotas` (POST = org admin) |
| PATCH/DELETE | `/api/v1/platform/quotas/{quotaID}` (org admin) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/policies` (POST = org admin) |
| POST | `/api/v1/platform/organizations/{orgID}/policies/reorder` (org admin; batch priority update) |
| PATCH/DELETE | `/api/v1/platform/policies/{policyID}` (org admin) |
| GET/POST | `/api/v1/platform/organizations/{orgID}/credentials` (POST = org admin) |
| PATCH/DELETE | `/api/v1/platform/credentials/{credentialID}` (org admin) |
| POST | `/api/v1/platform/credentials/{credentialID}/rotate` (org admin) |
| GET/PUT | `/api/v1/platform/organizations/{orgID}/credential-assignments` (PUT = org admin) |
| DELETE | `/api/v1/platform/credential-assignments/{assignmentID}` (org admin) |

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
| `scope_type` | `organization`, `team`, `project`, `user`, `api_key` |
| `metric` | `requests`, `tokens` |
| `window` | `total` (Postgres lifetime), `minute` / `hour` / `day` (Redis fixed windows) |
| `limit_value` | integer ≥ 0 (`0` blocks immediately) |

Most specific scope wins **per window**: api_key → user → project → team → organization. Timed windows require Redis (`redis_url` / `AFI_REDIS_URL`). Create/update/delete quotas, providers, and routes require org owner/admin.

Team quotas apply when the authenticated key’s project has a `team_id` (compiled onto the snapshot key). Personal keys without a project do not match team scopes.

### Environments

Projects own named environments (`dev` / `stage` / `prod` / custom). Service-account keys may optionally bind an `environment_id` for usage attribution. Routes and providers remain organization-scoped.

| Method | Path |
| ------ | ---- |
| GET/POST | `/api/v1/platform/organizations/{orgID}/projects/{projectID}/environments` (POST = org admin) |
| GET/DELETE | `/api/v1/platform/environments/{environmentID}` (DELETE = org admin) |

### Route retry

Optional same-target retry before ordered `fallbacks` failover. Create/update via `POST/PATCH` route bodies (`retry` field). Omitted/`null` means a single attempt per target (current gateway behavior).

| Field | Notes |
|-------|--------|
| `max_attempts` | Total tries including the first (≥ 1) |
| `backoff.strategy` | `fixed` or `exponential` |
| `backoff.base_delay` | Go duration string (e.g. `100ms`, `1s`) |
| `backoff.max_delay` | Exponential only; caps delay |
| `backoff.multiplier` | Exponential only; defaults to `2` when omitted |

`fixed` rejects `max_delay` / `multiplier`. Delay before retry index `n` (0 = first retry): fixed → `base_delay`; exponential → `min(base_delay * multiplier^n, max_delay)`.

Compiled into the gateway snapshot. Resolution order: **route `retry` → org `default_retry` → none**. On chat (`/v1/chat/completions`) and Anthropic messages (`/v1/messages`), the gateway selects the attempt list (`ordered` / `weighted` / `latency` / `cost`), retries the same target up to `max_attempts` with the configured backoff on transport errors, HTTP 5xx, and 429, then walks the remainder.

### Route routing strategy

| `routing_strategy` | First attempt | Failover order |
| ------------------ | ------------- | -------------- |
| `ordered` (default) | Primary | Config list order of `fallbacks` |
| `weighted` | Weighted random among primary + fallbacks (`weight`, default `1`) | Remaining candidates in original config order |
| `latency` | Lowest gateway-local EWMA latency among primary + fallbacks | Remaining by ascending EWMA (unknown → median of known) |
| `cost` | Lowest embedded catalog unit price (`input+output` $/MTok) | Remaining by ascending price (unknown last) |

Unknown strategies are rejected at write time. Adaptive signals are process-local (multi-instance skew accepted); see `internal-docs/weighted-adaptive-routing.md`.

### CEL policies

| Field | Notes |
|-------|--------|
| `name` | Display name |
| `expression` | Boolean CEL **when** clause |
| `actions` | Ordered Then steps: `[{ "type", "config"? }, …]` (at least one) |
| `priority` | Higher first (default 100). Batch-reorder via `POST .../policies/reorder`. |
| `enabled` | Default true |

Legacy create/update bodies may still send a single `action` + `action_config`; the API stores them as a one-element `actions` array.

**Then actions** (when expression is true: steps run **in order**; policies by priority desc):
| Action | Config | Behavior |
|--------|--------|----------|
| `deny` | `{}` | Stop; HTTP 403 `policy_violation` |
| `allow` | `{}` | Stop; allow (skips remaining Then steps and lower-priority rules) |
| `set_header` | `{ "header", "value"? , "value_expr"? }` | Set outbound upstream header; continue. Later writes to the same header overwrite. `value_expr` is CEL → string and wins over `value`. |
| `use_credential` | `{ "credential_name"? , "credential_name_expr"? }` | Select secret by name; continue. Later Then steps (and later matching policies) overwrite. `credential_name_expr` is CEL → string (e.g. header value) and wins over `credential_name`. |

CEL variables: `request.model`, `request.path`, `request.stream`, `request.tags`, `request.headers` (lowercased; `authorization` / `cookie` omitted), `key.*`, `credential.*`. Owner/admin to create/update/delete.

Examples:
- Deny a model: when `request.model == "blocked-model"` then `deny`
- Partner key by header value: when `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] != ""` then `use_credential` with `credential_name_expr: "request.headers[\\"x-tenant-id\\"]"` (secret name = header value)
- Fixed partner key: when header equals `acme` then `use_credential` with `credential_name: "partner-acme"`
- Multi-then: same when → `use_credential` (name from header) **then** `set_header` with `header: "X-Partner"`, `value_expr: "request.headers[\\"x-tenant-id\\"]"`

### Provider credentials (BYOK)

| `storage_kind` | Notes |
|----------------|--------|
| `env` | `secret_ref` = process env name (or `env://NAME`) |
| `encrypted_db` | plaintext sealed with `AFI_CREDENTIALS_MASTER_KEY` |
| `vault` | `secret_ref` = `aws-sm://{region}/{secret-id}[#jsonKey]` or `hashicorp://{path}[#jsonKey]` |

Create secrets normally, then select them from a `use_credential` policy and/or assign scopes. Runtime resolve order: **policy use_credential → api_key → project → organization → provider.api_key_env**. Unknown credential names fail closed.

Usage list/summary query params: `exclude_byok`, `byok_only`, `credential_id`; summary `group_by=byok`.
