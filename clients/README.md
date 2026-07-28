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

CI publishes via [OIDC trusted publishing](https://docs.npmjs.com/trusted-publishers/) — **no `NPM_TOKEN`**.

Configure once (package already exists at `1.0.0`):

1. Open **https://www.npmjs.com/package/@afi-ai/platform-client/access** (package Access page, not account settings)
2. **Trusted Publisher** → GitHub Actions:
   - Owner: `curefatih`
   - Repository: `afi`
   - Workflow filename: `release-clients.yml` (filename only)
   - **Allowed actions:** must include `npm publish` (required for publishers created after May 2026)
   - **Environment:** leave empty
3. Delete GitHub Actions secret `NPM_TOKEN` if present
4. Re-run **Release clients** on `main`

If CI still prints the “tokens that bypass 2FA” notice, a token is still being used — Trusted Publisher is missing/mismatched, or `NPM_TOKEN` is still injected.

### PyPI auth (Python)

Repo secret `PYPI_API_TOKEN` (or configure a PyPI trusted publisher for this workflow).

### Manual / dry-run

```bash
DRY_RUN=1 make release-clients          # detect changes since HEAD~1
DRY_RUN=1 CLIENTS=typescript FORCE=1 make release-clients
NODE_AUTH_TOKEN=… make release-client-typescript
PYPI_API_TOKEN=… make release-client-python
```
