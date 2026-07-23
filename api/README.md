# Public API contracts

Hand-authored OpenAPI 3.1 specs for AFI HTTP surfaces.

| Spec | Audience |
| ---- | -------- |
| [`openapi/platform.openapi.yaml`](openapi/platform.openapi.yaml) | Control plane `/api/v1/platform/*` (admin / JWT) |
| [`openapi/gateway.openapi.yaml`](openapi/gateway.openapi.yaml) | Data plane overlay (virtual keys, AFI headers, path map) |

Proto under `proto/` is reserved for the gRPC extension runtime (P2.2); it is not part of the public HTTP contract.

## Commands

```bash
make openapi-lint    # Spectral lint
make openapi-gen     # Regenerate TypeScript types + Python models
make openapi-check   # Lint + path drift vs Go handlers + gen freshness
```

## Versioning

Specs use `info.version: 1.0.0` aligned with the `/api/v1` and `/v1` URL prefixes. Breaking changes require a new major API path (`v2`), not silent schema breaks.

## Clients

Thin HTTP clients live in [`clients/`](../clients/) (not under `sdk/`, which is for in-process extensions).
