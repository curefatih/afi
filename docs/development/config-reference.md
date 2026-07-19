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
| `AFI_INTERNAL_TOKEN` | from yaml (`afi-local-internal-token`) | HTTP `/internal/v1/*` |
| `AFI_TOKEN_TTL` | `24h` | JWT lifetime |
| `OPENAI_API_KEY` | _(required for OpenAI live calls)_ | gateway → OpenAI |
| `ANTHROPIC_API_KEY` | _(required for Anthropic routes)_ | gateway → Anthropic |
| `GEMINI_API_KEY` | _(required for Gemini routes)_ | gateway → Gemini |
| `VITE_PLATFORM_API_URL` | `http://localhost:8081` | web UI (platform APIs) |
| `VITE_GATEWAY_API_URL` | `http://localhost:8080` | web playground |
| `VITE_GATEWAY_API_KEY` | seed virtual key | web playground |

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
| Seeded providers | `prov_openai`, `prov_anthropic`, `prov_gemini`, `prov_ollama` (`openai_compatible` → `http://127.0.0.1:11434/v1`, no default route) |
| `OLLAMA_API_KEY` | _(any value if Ollama ignores auth)_ | gateway → openai_compatible |
| Route `fallbacks` | optional ordered `[{provider_id,target_model}]` for 5xx/timeout/429 retry |
| Gateway models | `GET /v1/models` lists org route model ids for the virtual key |

## Ports

| Port | Service |
|------|---------|
| 8080 | Gateway (inference) |
| 8081 | Control plane (platform + admin) |
| 5433 | Postgres |
| 5050 | Adminer |
| 3000 | Web UI |
| 8000 | MkDocs (`make doc-serve`) |

## Platform config APIs

| Method | Path |
|--------|------|
| GET/POST | `/api/v1/platform/organizations` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/members` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/keys` |
| DELETE | `/api/v1/platform/keys/{keyID}` |
| GET/POST | `/api/v1/platform/projects/{projectID}/keys` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/providers` |
| PATCH/DELETE | `/api/v1/platform/providers/{providerID}` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/routes` |
| PATCH/DELETE | `/api/v1/platform/routes/{routeID}` |
| GET | `/api/v1/platform/organizations/{orgID}/usage` |
| GET/POST | `/api/v1/platform/organizations/{orgID}/quotas` |
| PATCH/DELETE | `/api/v1/platform/quotas/{quotaID}` |

Member invite looks up an existing user by email (no SMTP). Org roles: `owner` / `admin` / `member`. Native Anthropic inference: gateway `POST /v1/messages` (Anthropic providers only).

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
| `window` | `total` (lifetime counter) |
| `limit_value` | integer ≥ 0 (`0` blocks immediately) |

Most specific scope wins: api_key → user → project → organization. Create/update/delete quotas require org owner/admin.
