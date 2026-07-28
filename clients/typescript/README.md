# @afi-ai/platform-client

Thin TypeScript fetch client for the AFI control plane.

```bash
cd clients/typescript && pnpm install && pnpm test
```

```ts
import { PlatformClient } from "@afi-ai/platform-client";

const client = new PlatformClient({
  baseUrl: "http://localhost:8081",
  getToken: () => process.env.AFI_TOKEN!,
});

const orgs = await client.listOrganizations();
```

OpenAPI types are generated into `src/schema.gen.ts` via `make openapi-gen`.

## Gateway signed requests

```ts
import { signHeaders } from "@afi-ai/platform-client";

const headers = {
  "Content-Type": "application/json",
  ...signHeaders({
    method: "POST",
    url: "http://localhost:8080/v1/chat/completions",
    body,
    privateKeyPem,
    keyId: "local-signer",
  }),
};
```

Release time: Jul 29 01:14 AM