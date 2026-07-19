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
| Adminer | http://localhost:5050 |

## 2. Export provider credentials

```bash
export OPENAI_API_KEY="sk-..."
```

The gateway reads this env var when calling OpenAI (see snapshot provider `api_key_env`).

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

Alternatively: `make run-all` (control plane in background, gateway in foreground). Stop with Ctrl+C then `make stop-all` if needed.

## 5. Verify inference

See [Verify](verify.md).

## 6. Optional: platform UI

```bash
pnpm --dir web install
pnpm --dir web dev
```

Open http://localhost:3000 and sign in with `admin@afi.local` / `admin`.

The web app calls the control plane at `http://localhost:8081` (override with `VITE_PLATFORM_API_URL`).

## 7. Optional: docs site

```bash
make doc-serve
```

## CLI helpers

```bash
make seed                 # re-seed + publish
make snapshot-publish     # compile config → new snapshot
go run ./cmd/cli version
```

## Config

Defaults live in [`configs/local.yaml`](../../configs/local.yaml). Override with env vars (see [Config reference](../development/config-reference.md)).
