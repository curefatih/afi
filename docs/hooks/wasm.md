# WASM hooks (data plane)

AFI can run **sandboxed TinyGo WASM modules** as lifecycle hooks on the gateway request path.

## Enable on the gateway

```bash
export AFI_WASM_BEFORE_CALL=/path/to/hook.wasm   # optional BeforeCall
export AFI_WASM_BEFORE_CHAT=/path/to/hook.wasm   # optional BeforeChat
```

Unset = no WASM hooks (default). Built-in CEL and quota gates are unchanged.

Example module: [`extensions/wasmhook`](../../extensions/wasmhook).

??? example "Gateway wiring (`cmd/gateway`)"

    ```go
    import afiwasm "github.com/curefatih/afi/internal/adapters/wasm"

    if path := cfg.Gateway.WasmBeforeCall; path != "" {
        mod, err := afiwasm.CompileFile(ctx, path, afiwasm.Config{Name: "wasm:before_call"})
        // handle err...
        hook, err := afiwasm.NewBeforeCall(mod)
        hooks = hooks.RegisterBeforeCall(hook)
    }
    if path := cfg.Gateway.WasmBeforeChat; path != "" {
        mod, err := afiwasm.CompileFile(ctx, path, afiwasm.Config{Name: "wasm:before_chat"})
        // handle err...
        hook, err := afiwasm.NewBeforeChat(mod)
        hooks = hooks.Register(hook)
    }
    ```

## Guest ABI (v1)

Guest must export TinyGo allocator symbols and at least one hook:

| Export | Signature | Role |
|--------|-----------|------|
| `malloc` | `(size i32) -> i32` | Host allocates input buffer |
| `free` | `(ptr i32)` | Host frees input/output buffers |
| `before_call` | `(ptr i32, len i32) -> i64` | Optional; packed `ptr<<32 \| len` of JSON out |
| `before_chat` | `(ptr i32, len i32) -> i64` | Optional; same packing |

Build with:

```bash
tinygo build -o hook.wasm -scheduler=none -target=wasip1 -buildmode=c-shared .
```

The host instantiates with `_initialize` (not `_start`). WASI preview1 is provided with **no filesystem or network**.

### `before_call` JSON

**Input**

```json
{
  "principal": {
    "organization_id": "",
    "project_id": "",
    "api_key_id": "",
    "kind": "",
    "owner_user_id": "",
    "name": ""
  },
  "route": {
    "model": "",
    "path": "",
    "stream": false,
    "modality": ""
  },
  "tags": {},
  "metadata": {},
  "body_b64": ""
}
```

**Output**

```json
{
  "allow": true,
  "status": 0,
  "reason": "",
  "message": "",
  "headers": {},
  "tags": {},
  "metadata": {},
  "body_b64": null
}
```

- `allow: false` denies the request (default status 403 if unset).
- If `tags` / `metadata` are present, they **replace** the call’s maps.
- If `body_b64` is a JSON string (including `""`), it replaces `CallContext.Body`. Omit or `null` to leave the body unchanged.

??? example "TinyGo guest `before_call` (excerpt from `extensions/wasmhook`)"

    ```go
    //go:wasmexport before_call
    func _before_call(ptr, size uint32) uint64 {
        in := ptrToBytes(ptr, size)
        var req beforeCallIn
        if err := json.Unmarshal(in, &req); err != nil {
            return leakJSON(beforeCallOut{
                Allow: false, Status: 500, Reason: "wasm_error",
            })
        }
        if req.Tags["plan"] == "blocked" {
            return leakJSON(beforeCallOut{
                Allow: false, Status: 403, Reason: "plan_blocked",
                Message: "plan=blocked denied by wasm hook",
            })
        }
        if req.Metadata == nil {
            req.Metadata = map[string]any{}
        }
        req.Metadata["wasm_hook"] = "1"
        return leakJSON(beforeCallOut{
            Allow: true, Tags: req.Tags, Metadata: req.Metadata,
        })
    }
    ```

### `before_chat` JSON

**Input:** `{ "body_b64": "..." }`  
**Output:** `{ "body_b64": "..." }` — decoded bytes become the chat request body.

??? example "TinyGo guest `before_chat` (excerpt)"

    ```go
    //go:wasmexport before_chat
    func _before_chat(ptr, size uint32) uint64 {
        // decode body_b64 → OpenAI chat JSON
        // prefix last user message with "[wasm] "
        // return {"body_b64": "..."}
    }
    ```

## Limits

- Default invoke timeout: **100ms** (wall clock via context).
- Default memory limit: **32 MiB** (512 WASM pages).
- Guest panic / timeout / invalid JSON → hook **error** → HTTP 500 (fail closed), same as a Go hook error.

## Field names

Go types in [`sdk/hook`](../../sdk/hook) are the source of truth for semantics; wire names are snake_case as above.

## Performance: WASM vs native Go

Comparable hooks (same allow/deny + metadata stamp / chat prefix logic) are benchmarked in [`internal/adapters/wasm/bench_test.go`](../../internal/adapters/wasm/bench_test.go):

* **Native** — in-process Go `sdk/hook` implementation (no sandbox).
* **WASM** — TinyGo `extensions/wasmhook/hook.wasm` via the wazero host (current v1: **fresh module instance per invoke**).

Each benchmark warms up with **64 invokes** before `ResetTimer()` so reported numbers exclude compile/load and cold-cache effects as much as Go’s harness allows.

### How to run

```bash
go test ./internal/adapters/wasm/ \
  -bench='BenchmarkBefore(Call|Chat)_' \
  -benchmem -count=10 -benchtime=500ms
```

### Results (warmed, `-count=10`)

Machine: `darwin/arm64` (Apple Silicon). Values are **median** ns/op across 10 runs (min–max shown for stability).

| Benchmark | Native (med) | WASM (med) | Approx. ratio | Native range | WASM range |
|-----------|--------------|------------|---------------|--------------|------------|
| `BeforeCall` allow | 46.5 ns, 0 allocs | 358 µs, ~523 KiB, 322 allocs | ~**7700×** | 46–50 ns | 353–374 µs |
| `BeforeCall` deny | 97.8 ns, 0 allocs | 263 µs, ~523 KiB, 319 allocs | ~**2700×** | 97–99 ns | 262–266 µs |
| `BeforeChat` prefix | 1.68 µs, 38 allocs | 71.3 µs, ~193 KiB, 297 allocs | ~**42×** | 1.64–1.88 µs | 70–74 µs |

After warmup, run-to-run spread is small (a few percent). Cold-start / first-load cost is **not** in these numbers — module compile happens once in setup; per-op cost is still dominated by **instantiate + JSON ABI**.

??? example "Native Go hook used in the benchmark"

    ```go
    type nativeBeforeCall struct{}

    func (nativeBeforeCall) BeforeCall(_ context.Context, call *sdkhook.CallContext) (sdkhook.CallDecision, error) {
        if call.Tags["plan"] == "blocked" {
            return sdkhook.Deny(403, "plan_blocked", "plan=blocked denied by native hook"), nil
        }
        if call.Metadata == nil {
            call.Metadata = map[string]any{}
        }
        call.Metadata["wasm_hook"] = "1"
        return sdkhook.Allow(), nil
    }
    ```

??? example "Benchmark harness (warmup + timed loop)"

    ```go
    func BenchmarkBeforeCall_WASM(b *testing.B) {
        ctx := context.Background()
        mod, err := CompileFile(ctx, exampleWASMPath(b), Config{Name: "bench", Timeout: time.Second})
        // ...
        h, err := NewBeforeCall(mod)
        for i := 0; i < 64; i++ { // warmup
            _, _ = h.BeforeCall(ctx, benchCallAllow())
        }
        b.ReportAllocs()
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            d, err := h.BeforeCall(ctx, benchCallAllow())
            if err != nil || !d.Allow {
                b.Fatal(err)
            }
        }
    }
    ```

??? tip "Full raw `go test` output (`-count=10`)"

    ```text
    BenchmarkBeforeCall_Native-12         	…	        46.51 ns/op	       0 B/op	       0 allocs/op
    BenchmarkBeforeCall_WASM-12           	…	    358450 ns/op	  522785 B/op	     322 allocs/op
    BenchmarkBeforeCall_Native_Deny-12    	…	        97.80 ns/op	       0 B/op	       0 allocs/op
    BenchmarkBeforeCall_WASM_Deny-12      	…	    263276 ns/op	  522579 B/op	     319 allocs/op
    BenchmarkBeforeChat_Native-12         	…	      1677 ns/op	    1520 B/op	      38 allocs/op
    BenchmarkBeforeChat_WASM-12           	…	     71271 ns/op	  193115 B/op	     297 allocs/op
    ```

    Medians from the same run set; individual lines vary slightly within the ranges in the table above.

### How to read this

* Prefer **native Go hooks** for hot-path gates when you control the binary and do not need sandbox isolation.
* Prefer **WASM** when tenants or operators supply untrusted transform/deny logic and sandboxing matters more than microseconds.
* Most of the WASM cost in v1 is **per-call instantiate + JSON ABI**, not the guest’s business logic. Instance pooling / reuse would narrow the gap substantially (not shipped yet).
* Absolute WASM latency (~0.07–0.36 ms median) is still small versus typical LLM round-trips (tens–hundreds of ms), so it is often acceptable on the gateway path when isolation is required.
