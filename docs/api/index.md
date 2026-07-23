# Public HTTP APIs

AFI exposes two HTTP surfaces:

| Surface | Default | Auth | Contract |
| ------- | ------- | ---- | -------- |
| **Platform** (control plane) | `:8081` `/api/v1/platform` | Session JWT | [platform.openapi.yaml](https://github.com/curefatih/afi/blob/main/api/openapi/platform.openapi.yaml) |
| **Gateway** (data plane) | `:8080` `/v1/*`, `/mcp/*`, `/a2a/*` | Virtual API key | [gateway.openapi.yaml](https://github.com/curefatih/afi/blob/main/api/openapi/gateway.openapi.yaml) (overlay) |

Specs live in the repo under [`api/openapi/`](https://github.com/curefatih/afi/tree/main/api/openapi). Regenerate / check with `make openapi-check`.

## Client SDKs

Thin platform clients (not for chat completions — use OpenAI/Anthropic SDKs against the gateway):

- TypeScript: `clients/typescript` (`@afi-ai/platform-client`)
- Python: `clients/python` (`afi-platform`)

See [Platform API](platform.md) and [Gateway overlay](gateway.md).

## Versioning

URL prefixes stay on `v1`. OpenAPI `info.version` tracks documentation/schema revisions within that major API.
