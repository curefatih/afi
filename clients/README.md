# AFI Platform HTTP clients

Thin clients generated / maintained against [`../api/openapi/platform.openapi.yaml`](../api/openapi/platform.openapi.yaml).

| Package | Language | Install |
| ------- | -------- | ------- |
| [`typescript/`](typescript/) | TypeScript | `npm i @afi-ai/platform-client` |
| [`python/`](python/) | Python | `pip install afi-platform` |

Local development: `pnpm add ../clients/typescript` or `pip install -e clients/python`.

These are **not** the in-process extension SDKs under [`../sdk/`](../sdk/).

Gateway callers should keep using official OpenAI / Anthropic / MCP / A2A SDKs with the gateway base URL — see [`../api/openapi/gateway.openapi.yaml`](../api/openapi/gateway.openapi.yaml).

For **signed-request auth** (RFC 9421) instead of a virtual API key, use the shared helpers:

| Language | Helper |
| -------- | ------ |
| Go | [`../sdk/httpsign`](../sdk/httpsign) — `SignRequest`, `Client` |
| Python | `afi_platform.sign_headers` |
| TypeScript | `signHeaders` from `@afi-ai/platform-client` |

## Releasing

Pushes to `main` that touch `clients/typescript` or `clients/python` run [`.github/workflows/release-clients.yml`](../.github/workflows/release-clients.yml). Each changed client is tested, versioned (patch auto-bump when the local version is already published), and published to npm / PyPI.

### npm auth (TypeScript)

Prefer [trusted publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC) — no long-lived write token:

1. Publish the package once (local `npm publish` with 2FA, or a one-time [granular access token](https://docs.npmjs.com/about-access-tokens) with **Bypass two-factor authentication**).
2. On npmjs.com → `@afi-ai/platform-client` → **Settings → Trusted Publisher** → GitHub Actions:
   - Repository: `curefatih/afi`
   - Workflow filename: `release-clients.yml`
3. Remove any classic `NPM_TOKEN` secret (classic tokens get `E403` under 2FA).

Optional CI fallback: repo secret `NPM_TOKEN` must be a **granular** token with publish + Bypass 2FA — not a classic token.

### PyPI auth (Python)

Repo secret `PYPI_API_TOKEN` (or configure a PyPI trusted publisher for this workflow).

### Manual / dry-run

```bash
DRY_RUN=1 make release-clients          # detect changes since HEAD~1
DRY_RUN=1 CLIENTS=typescript FORCE=1 make release-clients
NODE_AUTH_TOKEN=… make release-client-typescript
PYPI_API_TOKEN=… make release-client-python
```
