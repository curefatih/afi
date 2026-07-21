# WASM hooks (data plane)

AFI can run **sandboxed TinyGo WASM modules** as lifecycle hooks on the gateway request path.

## Enable on the gateway

### Env (process-global demos)

```bash
export AFI_WASM_BEFORE_CALL=/path/to/hook.wasm   # optional BeforeCall
export AFI_WASM_BEFORE_CHAT=/path/to/hook.wasm   # optional BeforeChat
```

### Control plane (org-scoped, snapshot-backed)

Admins manage bindings via platform API; they compile into the gateway snapshot and hot-reload:

```http
POST /api/v1/platform/organizations/{orgID}/wasm-hooks
Content-Type: application/json

{
  "name": "block-plan",
  "phase": "before_call",
  "module_uri": "file:///var/afi/hooks/block.wasm",
  "digest": "<sha256 hex of the .wasm file>",
  "enabled": true,
  "priority": 100,
  "config": {}
}
```

| Field | Notes |
|-------|--------|
| `phase` | `before_call` \| `before_chat` \| `after_call` |
| `module_uri` | Local path or `file://` URI (content-addressed by optional `digest`) |
| `digest` | SHA-256 hex of module bytes; empty skips verification |
| `priority` | Higher runs first within a phase |
| `config` | JSON object passed to the guest as `"config"` |

List / update / delete: `GET …/wasm-hooks`, `PATCH /api/v1/platform/wasm-hooks/{hookID}`, `DELETE …`.

Request path order: **CEL → env/user BeforeCall hooks → snapshot WASM BeforeCall → quota**.

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
| `after_call` | `(ptr i32, len i32) -> i64` | Optional; side effects (empty return OK) |

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
- Default **instance pool size: 8** (`Config.PoolSize`; set negative to disable pooling).
- Guest panic / timeout / invalid JSON → hook **error** → HTTP 500 (fail closed), same as a Go hook error.

## Field names

Go types in [`sdk/hook`](../../sdk/hook) are the source of truth for semantics; wire names are snake_case as above.

## Performance: WASM vs native Go

Comparable hooks (same allow/deny + metadata stamp / chat prefix logic) are benchmarked in [`internal/adapters/wasm/bench_test.go`](../../internal/adapters/wasm/bench_test.go):

* **Native** — in-process Go `sdk/hook` implementation (no sandbox).
* **WASM (pooled)** — default host: reused guest instances (`PoolSize=8`).
* **WASM (no pool)** — `PoolSize: -1` instantiate+close every invoke (pre-pool behavior).

Each benchmark warms up with **64 invokes** before `ResetTimer()`.

### How to run

```bash
go test ./internal/adapters/wasm/ \
  -bench='BenchmarkBefore(Call|Chat)_(Native|WASM)' \
  -benchmem -count=5 -benchtime=300ms
```

### Results (warmed, pooled default, `-count=5`)

Machine: `darwin/arm64` (Apple Silicon). Values are **median** ns/op.

| Benchmark | Native | WASM pooled | WASM no pool | Pooled vs native | Pool speedup |
|-----------|--------|-------------|--------------|------------------|--------------|
| `BeforeCall` allow | 46 ns, 0 allocs | **33 µs**, ~3.3 KiB, 49 allocs | 352 µs, ~523 KiB, 331 allocs | ~**720×** | ~**10.6×** |
| `BeforeCall` deny | 98 ns, 0 allocs | **33 µs**, ~3.0 KiB, 45 allocs | — | ~**340×** | — |
| `BeforeChat` prefix | 1.7 µs, 38 allocs | **24 µs**, ~1.7 KiB, 26 allocs | ~71 µs pre-pool | ~**14×** | ~**3×** |

Pooling removes most of the per-call instantiate cost. Remaining gap vs native is mainly **JSON ABI + guest execution**.

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

??? example "Pooled vs no-pool compile"

    ```go
    // Default: pool of 8 reused instances
    mod, _ := CompileFile(ctx, path, Config{Name: "hook"})

    // Disable pooling (instantiate every call)
    mod, _ := CompileFile(ctx, path, Config{Name: "hook", PoolSize: -1})
    ```

??? tip "Raw medians from the pooled run set"

    ```text
    BenchmarkBeforeCall_Native-12         	…	        46.13 ns/op	       0 B/op	       0 allocs/op
    BenchmarkBeforeCall_WASM-12           	…	     33149 ns/op	    3331 B/op	      49 allocs/op
    BenchmarkBeforeCall_WASM_NoPool-12    	…	    352217 ns/op	  535518 B/op	     331 allocs/op
    BenchmarkBeforeCall_Native_Deny-12    	…	        97.50 ns/op	       0 B/op	       0 allocs/op
    BenchmarkBeforeCall_WASM_Deny-12      	…	     32902 ns/op	    3027 B/op	      45 allocs/op
    BenchmarkBeforeChat_Native-12         	…	      1719 ns/op	    1520 B/op	      38 allocs/op
    BenchmarkBeforeChat_WASM-12           	…	     24195 ns/op	    1657 B/op	      26 allocs/op
    ```

### How to read this

* Prefer **native Go hooks** for hot-path gates when you control the binary and do not need sandbox isolation.
* Prefer **WASM** when tenants or operators supply untrusted transform/deny logic and sandboxing matters more than tens of microseconds.
* Keep pooling enabled (default). Absolute pooled WASM latency (~24–33 µs median) is negligible vs typical LLM round-trips.
* Guest modules should free host-visible buffers; long-lived pooled instances may still grow TinyGo heap over time — monitor in production and recycle via gateway restart / future snapshot reload.

## Roadmap (next)

1. ~~Control-plane Plugin bindings~~ — shipped (`wasm_hooks` table + snapshot + gateway runner).
2. ~~Content-addressed artifacts~~ — `digest` (sha256) + `file://` / path URIs (HTTP object store later).
3. ~~`AfterCall` guest export~~ — shipped in ABI + example module.
4. Header mutation helpers / richer response transforms.
5. gRPC providers / remote hooks.
