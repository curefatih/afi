# grpcecho

Sample **gRPC extension** plugin for AFI: process-isolated `ChatProvider` plus a demo `BeforeCall` that sets `X-Afi-Grpc-Echo: 1` on upstream request headers.

## Build

```bash
make build   # produces bin/grpcecho
# or
go build -o bin/grpcecho ./extensions/grpcecho
```

## Run with the gateway

In `configs/dev.yaml` (or your `AFI_CONFIG`):

```yaml
gateway:
  grpc_extensions:
    - id: grpcecho
      command: ["./bin/grpcecho"]
```

Restart the gateway. It spawns the plugin, sets `AFI_PLUGIN_SOCK`, performs the capability handshake, and registers provider type `grpcecho`.

Create a control-plane provider with `type: grpcecho` and a route (e.g. model `grpcecho-demo`), then:

```bash
curl -s http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-project-local-dev-token-12345" \
  -H 'Content-Type: application/json' \
  -d '{"model":"grpcecho-demo","messages":[{"role":"user","content":"ping"}]}'
```

Expect assistant content containing `grpcecho:`.

## Dial an already-running plugin

```bash
./bin/grpcecho -addr /tmp/afi-grpcecho.sock
```

```yaml
gateway:
  grpc_extensions:
    - id: grpcecho
      address: unix:///tmp/afi-grpcecho.sock
```
