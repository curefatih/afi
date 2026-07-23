# Platform API

Control plane admin API used by the web UI and automation.

- **Base URL:** `http://localhost:8081` (local)
- **Prefix:** `/api/v1/platform`
- **Auth:** `Authorization: Bearer <jwt>` from `POST /auth/login`, invite accept, or SSO
- **Errors:** `{ "error": "..." }`
- **OpenAPI:** [`api/openapi/platform.openapi.yaml`](https://github.com/curefatih/afi/blob/main/api/openapi/platform.openapi.yaml)

Internal ops routes (`POST /internal/v1/seed`, `POST /internal/v1/snapshots/publish`) require `X-AFI-Internal-Token` and are **not** part of the public SDK.

## TypeScript

```bash
# from repo root
cd clients/typescript && npm install && npm test
```

```ts
import { PlatformClient } from "@afi-ai/platform-client";

let token = "";
const client = new PlatformClient({
  baseUrl: "http://localhost:8081",
  getToken: () => token,
});

token = (await client.login({ email: "admin@example.com", password: "…" })).token;
const orgs = await client.listOrganizations();
```

## Python

```bash
pip install -e clients/python
```

```python
from afi_platform import PlatformClient

with PlatformClient("http://localhost:8081") as c:
    token = c.login("admin@example.com", "…")["token"]

client = PlatformClient("http://localhost:8081", get_token=lambda: token)
print(client.list_organizations())
```

## Route index

The OpenAPI file is the canonical path/method inventory. For operational notes (RBAC, quotas, keys), see [Config reference](../development/config-reference.md).
