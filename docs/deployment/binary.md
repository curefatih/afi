# Binary deployment

Deploy AFI as standalone binaries without containerizing the app processes. You still need Postgres (and Redis if you use timed quotas).

A release archive is enough to run: the four binaries plus a config file (copy `afi.example.yaml` → `afi.yaml` and edit).

## Download (recommended)

GitHub Actions builds portable archives for `linux`/`darwin` × `amd64`/`arm64` on every `v*` tag (and on demand via **Actions → Release → Run workflow**).

Each `afi-<version>-<os>-<arch>.tar.gz` contains:

| File | Purpose |
|------|---------|
| `controlplane`, `gateway`, `worker`, `afi` | Binaries (`CGO_ENABLED=0`) |
| `afi.example.yaml` | Starting config — copy and replace every `CHANGE_ME` |
| `README.txt` | Short run instructions |

```bash
tar -xzf afi-*-linux-amd64.tar.gz
cd afi-*-linux-amd64
cp afi.example.yaml afi.yaml
# edit afi.yaml
export AFI_CONFIG=$PWD/afi.yaml
./controlplane
```

Verify downloads with the matching `.sha256` file next to each archive.

## Build locally

### Local architecture

```bash
make build
# → bin/controlplane, bin/gateway, bin/worker, bin/afi
```

### Cross-compile release (default linux/amd64)

```bash
make build-release
# → bin/release/linux-amd64/{controlplane,gateway,worker,afi}

GOOS=linux GOARCH=arm64 make build-release

# Archive with example config (writes dist/*.tar.gz + .sha256):
PACKAGE=1 make build-release

# All CI targets:
TARGETS=linux/amd64,linux/arm64,darwin/amd64,darwin/arm64 PACKAGE=1 \
  bash scripts/build-release.sh
```

Script: [`scripts/build-release.sh`](../../scripts/build-release.sh). CI workflow: [`.github/workflows/release.yml`](../../.github/workflows/release.yml).

### Web UI (optional)

```bash
pnpm --dir web install
VITE_PLATFORM_API_URL=https://cp.example.com \
VITE_GATEWAY_API_URL=https://gw.example.com \
  pnpm --dir web build
# → web/dist  (serve with nginx/Caddy/CDN)
```

## Layout on the host

Suggested layout:

```text
/opt/afi/
  bin/controlplane
  bin/gateway
  bin/worker
  bin/afi
  etc/afi.yaml
```

```bash
export AFI_CONFIG=/opt/afi/etc/afi.yaml
# Prefer secrets via environment:
export AFI_DATABASE_URL='postgres://...'
export AFI_JWT_SECRET='...'
export AFI_INTERNAL_TOKEN='...'
export OPENAI_API_KEY='...'   # gateway only
```

Copy [`deploy/afi.example.yaml`](../../deploy/afi.example.yaml) as a starting point. Full knobs: [Customization reference](customization.md).

## Process order

1. Start **Postgres** (and **Redis** if needed).
2. Start **control plane** — migrates schema, seeds if empty, publishes a snapshot, listens.
3. Start **gateway** — loads snapshot, serves inference (needs provider env vars).
4. Start **worker** — drains `usage_outbox` (and platform events if enabled).

```bash
AFI_CONFIG=/opt/afi/etc/afi.yaml /opt/afi/bin/controlplane
AFI_CONFIG=/opt/afi/etc/afi.yaml OPENAI_API_KEY=... /opt/afi/bin/gateway
AFI_CONFIG=/opt/afi/etc/afi.yaml /opt/afi/bin/worker
```

## systemd examples

`/etc/systemd/system/afi-controlplane.service`:

```ini
[Unit]
Description=AFI control plane
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=afi
Group=afi
Environment=AFI_CONFIG=/opt/afi/etc/afi.yaml
EnvironmentFile=-/opt/afi/etc/controlplane.env
ExecStart=/opt/afi/bin/controlplane
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/afi-gateway.service`:

```ini
[Unit]
Description=AFI gateway
After=network-online.target afi-controlplane.service
Wants=afi-controlplane.service

[Service]
Type=simple
User=afi
Group=afi
Environment=AFI_CONFIG=/opt/afi/etc/afi.yaml
EnvironmentFile=-/opt/afi/etc/gateway.env
ExecStart=/opt/afi/bin/gateway
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/afi-worker.service`:

```ini
[Unit]
Description=AFI worker
After=network-online.target afi-controlplane.service
Wants=afi-controlplane.service

[Service]
Type=simple
User=afi
Group=afi
Environment=AFI_CONFIG=/opt/afi/etc/afi.yaml
EnvironmentFile=-/opt/afi/etc/worker.env
ExecStart=/opt/afi/bin/worker
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
```

Put JWT/DB secrets in `controlplane.env`, provider API keys in `gateway.env`, and event publisher settings in both control plane and worker env files as needed.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now afi-controlplane afi-gateway afi-worker
```

## Reverse proxy

Terminate TLS and route:

| Public path / host | Upstream |
|--------------------|----------|
| Platform API / UI API | `controlplane:8081` |
| Inference (`/v1/*`) | `gateway:8080` |
| Static UI | `web/dist` or nginx container |

Ensure `VITE_PLATFORM_API_URL` / `VITE_GATEWAY_API_URL` match the public URLs browsers use.

## CLI on the server

```bash
AFI_CONFIG=/opt/afi/etc/afi.yaml /opt/afi/bin/afi seed
AFI_CONFIG=/opt/afi/etc/afi.yaml /opt/afi/bin/afi snapshot publish
AFI_CONFIG=/opt/afi/etc/afi.yaml /opt/afi/bin/afi version
```

Destructive local wipe only: `afi db reset` (types `reset` to confirm).

## Health

```bash
curl -fsS http://127.0.0.1:8081/healthz
curl -fsS http://127.0.0.1:8080/healthz
AFI_CONTROLPLANE_URL=http://127.0.0.1:8081 \
AFI_GATEWAY_URL=http://127.0.0.1:8080 \
  make deploy-health
```

## Scaling notes

* Control plane and gateway are horizontally scalable behind a load balancer (shared Postgres).
* Gateway keeps the latest snapshot in memory and watches for new versions.
* Run **one or more** workers; they claim outbox batches cooperatively.
* Redis is shared for timed quota windows across gateway replicas.
