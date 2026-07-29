# AFI Platform HTTP clients

Thin clients generated / maintained against [`../api/openapi/platform.openapi.yaml`](../api/openapi/platform.openapi.yaml).

| Package | Language | Install |
| ------- | -------- | ------- |
| [`typescript/`](typescript/) | TypeScript | `npm i @afi-ai/platform-client` |
| [`python/`](python/) | Python | `pip install afi-platform` |
| [`java/`](java/) | Java | `ai.afi:platform-client` (Maven) |

Local development: path install for TS/Python, or `mvn -f clients/java test`.

These are **not** the in-process extension SDKs under [`../sdk/`](../sdk/).

Gateway callers should keep using official OpenAI / Anthropic / MCP / A2A SDKs with the gateway base URL — see [`../api/openapi/gateway.openapi.yaml`](../api/openapi/gateway.openapi.yaml).

For **signed-request auth** (RFC 9421) instead of a virtual API key, use the shared helpers:

| Language | Helper |
| -------- | ------ |
| Go | [`../sdk/httpsign`](../sdk/httpsign) — `SignRequest`, `Client` |
| Python | `afi_platform.sign_headers` |
| TypeScript | `signHeaders` from `@afi-ai/platform-client` |
| Java | `ai.afi.platform.HttpSign.signHeaders` |

## Releasing

Pushes to `main` that touch `clients/typescript`, `clients/python`, or `clients/java` run [`.github/workflows/release-clients.yml`](../.github/workflows/release-clients.yml). Each changed client is tested, versioned (patch auto-bump when the local version is already published), and published to npm / PyPI / Maven Central.

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

After a successful publish, CI pushes the release tag, opens
`chore/clients-version-bump`, and **merges it** (`gh pr merge --squash`).
Title includes `[skip release]` so merge does not republish.

**Required for PR create/merge with `GITHUB_TOKEN`:**

1. **Settings → Actions → General → Workflow permissions**
   - Read and write permissions
   - ✅ **Allow GitHub Actions to create and approve pull requests**

Or set secret `CLIENT_RELEASE_GH_TOKEN` to a fine-grained PAT (contents +
pull requests) / classic `repo` PAT — the workflow prefers that token.

If merge still fails under branch rules, enable **Allow auto-merge** and/or
allow the bot/PAT to bypass required reviews/checks.

If CI prints `ENEEDAUTH` or the “bypass 2FA” notice with no token configured, npm never completed the OIDC exchange — usually an empty `_authToken` in `$NPM_CONFIG_USERCONFIG`, a Trusted Publisher mismatch, or npm &lt; 11.5.1.

### PyPI auth (Python)

Repo secret `PYPI_API_TOKEN` (or configure a PyPI trusted publisher for this workflow).

### Maven Central (Java)

1. Claim namespace `ai.afi` on [Sonatype Central Portal](https://central.sonatype.com/)
2. Configure GPG signing and a `~/.m2/settings.xml` server id `central` (or CI secrets `MAVEN_USERNAME` + `MAVEN_PASSWORD` / `MAVEN_CENTRAL_TOKEN`)
3. First publish: `DRY_RUN=1 CLIENTS=java FORCE=1 make release-clients`, then publish with `-Prelease`

Until Central credentials exist, use `DRY_RUN=1` / `SKIP_PUBLISH=1`.

### Manual / dry-run

```bash
DRY_RUN=1 make release-clients          # detect changes since HEAD~1
DRY_RUN=1 CLIENTS=java FORCE=1 make release-clients
CLIENTS=typescript FORCE=1 make release-clients
```
