# wasmhook

Example TinyGo WASM lifecycle hook for the AFI gateway.

## Behavior

- **before_call** — denies when `X-AFI-Tags` includes `plan:blocked`; otherwise allows and sets `metadata.wasm_hook=1`.
- **before_chat** — mutates typed chat IR (`{"request":{...}}`) and prefixes the last user message with `[wasm] `.
- **after_call** — no-op.

## Rebuild

Requires [TinyGo](https://tinygo.org/getting-started/install/):

```bash
make build
# or:
tinygo build -o hook.wasm -scheduler=none -target=wasip1 -buildmode=c-shared .
```

A prebuilt `hook.wasm` is committed for tests and local demos.

> **Important:** the checked-in `hook.wasm` may predate the typed `before_chat`
> ABI (`{"request":{...}}`). Loading a legacy module as `wasm_before_chat`
> empties chat messages (the host used to decode `{"body_b64":""}` as an empty
> request). Rebuild before enabling `AFI_WASM_BEFORE_CHAT` /
> `gateway.wasm_before_chat`:
>
> ```bash
> make -C extensions/wasmhook build
> ```

## Run with the gateway

```bash
export AFI_WASM_BEFORE_CALL=/path/to/afi/extensions/wasmhook/hook.wasm
# optional chat body mutation:
export AFI_WASM_BEFORE_CHAT=/path/to/afi/extensions/wasmhook/hook.wasm
```

See [docs/development/hooks/wasm.md](../../docs/development/hooks/wasm.md) for the host↔guest ABI and **WASM vs native Go benchmarks**.
