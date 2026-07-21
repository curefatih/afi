# wasmhook

Example TinyGo WASM lifecycle hook for the AFI gateway.

## Behavior

- **before_call** — denies when `X-AFI-Tags` includes `plan:blocked`; otherwise allows and sets `metadata.wasm_hook=1`.
- **before_chat** — prefixes the last user message with `[wasm] `.

## Rebuild

Requires [TinyGo](https://tinygo.org/getting-started/install/):

```bash
make build
# or:
tinygo build -o hook.wasm -scheduler=none -target=wasip1 -buildmode=c-shared .
```

A prebuilt `hook.wasm` is committed for tests and local demos.

## Run with the gateway

```bash
export AFI_WASM_BEFORE_CALL=/path/to/afi/extensions/wasmhook/hook.wasm
# optional chat body mutation:
export AFI_WASM_BEFORE_CHAT=/path/to/afi/extensions/wasmhook/hook.wasm
```

See [docs/hooks/wasm.md](../../docs/hooks/wasm.md) for the host↔guest ABI and **WASM vs native Go benchmarks**.
