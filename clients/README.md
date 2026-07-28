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

`@afi-ai/platform-client@1.0.0` is already on npm. CI publishes via [trusted publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC) — **required** for automated releases (GAT bypass tokens are being restricted for direct publish).

Configure once on npmjs.com:

1. Open https://www.npmjs.com/package/@afi-ai/platform-client → **Settings** / **Access** → **Trusted Publisher**
2. Add **GitHub Actions**:
   - Organization or user: `curefatih`
   - Repository: `afi`
   - Workflow filename: `release-clients.yml` (filename only, including `.yml`)
3. Remove repo secret `NPM_TOKEN` if present (it interferes with OIDC)
4. Re-run **Release clients** on `main` (or merge this branch and dispatch)

Without that Trusted Publisher row, CI gets `E404` on `npm publish` even though the package exists.

### PyPI auth (Python)

Repo secret `PYPI_API_TOKEN` (or configure a PyPI trusted publisher for this workflow).

### Manual / dry-run

```bash
DRY_RUN=1 make release-clients          # detect changes since HEAD~1
DRY_RUN=1 CLIENTS=typescript FORCE=1 make release-clients
NODE_AUTH_TOKEN=… make release-client-typescript
PYPI_API_TOKEN=… make release-client-python
```
