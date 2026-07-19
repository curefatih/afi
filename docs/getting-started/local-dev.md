# Local development

This is the golden path for running AFI on a developer machine.

## Prerequisites

* Go (version in `go.mod`)
* Docker (Compose)
* An OpenAI API key for live chat completions
* Optional: pnpm (web UI), `uv`/`uvx` (MkDocs)

## 1. Start infrastructure

```bash
make dev-up
```

| Service | URL |
|---------|-----|
| Postgres | `localhost:5433` (user/password/db: `afi`) |
| Redis | `localhost:6379` (rate-limit windows) |
| Adminer | http://localhost:5050 |

## 2. Export provider credentials

```bash
export OPENAI_API_KEY="sk-..."
# optional — Anthropic / Gemini routes (seeded as prov_anthropic / prov_gemini)
export ANTHROPIC_API_KEY="sk-ant-..."
export GEMINI_API_KEY="..."
```

The gateway reads these env vars when calling upstream providers (see snapshot provider `api_key_env`). Playgrounds load models from `GET /v1/models` (stream / TTS / STT flags). Seed includes chat route plus `tts-1` and `whisper-1` on OpenAI.

## 3. Run the control plane

```bash
make run-controlplane
```

On startup the control plane:

1. Applies migrations (resets incompatible legacy schemas automatically)
2. Seeds a local org, project, user, virtual API key, OpenAI provider, and route (if empty)
3. Publishes snapshot version 1+
4. Listens on **`:8081`**

Default seed credentials:

| Item | Value |
|------|--------|
| Virtual API key | `sk-project-local-dev-token-12345` |
| Platform user | `admin@afi.local` / `admin` |

## 4. Run the gateway

In a second terminal:

```bash
export OPENAI_API_KEY="sk-..."
make run-gateway
```

The gateway loads the latest snapshot from Postgres, watches for new versions, and listens on **`:8080`**.

## 5. Run the usage worker (optional but recommended)

In a third terminal:

```bash
make run-worker
```

The worker drains `usage_outbox` into `usage_events` (Usage page: filters, owner, modality charts). Chat/TTS/STT still work if the worker is stopped; usage just lags.

Alternatively: `make run-all` (control plane + worker in background, gateway in foreground). Stop with Ctrl+C then `make stop-all` if needed.

## 6. Verify inference

See [Verify](verify.md) (`make verify` includes quota → 429).

## 7. Optional: platform UI

```bash
pnpm --dir web install
pnpm --dir web dev
```

Open http://localhost:3000 and sign in with `admin@afi.local` / `admin`.

Use **Organizations** to create another org and add an existing user by email (user must already exist — no SMTP invite).

**API Keys:** create a **personal** key for yourself, or (as org owner/admin) a **service account** key scoped to the org or a project. The seeded `sk-project-local-dev-token-12345` is a project service-account key. Admins set per-user or per-key quotas under **Quotas** (`total` or Redis `minute`/`hour`/`day`) and CEL allow-rules under **Policies**.

The web app calls the control plane at `http://localhost:8081` (override with `VITE_PLATFORM_API_URL`).

## 8. Optional: docs site

```bash
make doc-serve
```

## CLI helpers

```bash
make seed                 # re-seed + publish
make snapshot-publish     # compile config → new snapshot
make verify               # automated checks (stack must be running)
go run ./cmd/cli version
afi db reset              # destructive local wipe (type `reset` to confirm)
```

Internal HTTP admin (`/internal/v1/*`) requires `X-AFI-Internal-Token` (see config reference). CLI seed/publish do not.

## Config

Defaults live in [`configs/local.yaml`](../../configs/local.yaml). Override with env vars (see [Config reference](../development/config-reference.md)).
