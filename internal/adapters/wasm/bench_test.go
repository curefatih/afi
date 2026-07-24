package wasm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/curefatih/afi/sdk/chatir"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// nativeBeforeCall mirrors extensions/wasmhook before_call logic in-process
// (deny plan=blocked; else stamp metadata) without JSON/WASM overhead.
type nativeBeforeCall struct{}

func (nativeBeforeCall) Name() string { return "native_bench" }

func (nativeBeforeCall) BeforeCall(_ context.Context, call *sdkhook.CallContext) (sdkhook.CallDecision, error) {
	if call.Tags == nil {
		call.Tags = map[string]string{}
	}
	if call.Metadata == nil {
		call.Metadata = map[string]any{}
	}
	if call.Tags["plan"] == "blocked" {
		return sdkhook.Deny(403, "plan_blocked", "plan=blocked denied by native hook"), nil
	}
	call.Metadata["wasm_hook"] = "1"
	return sdkhook.Allow(), nil
}

// nativeBeforeChat mirrors extensions/wasmhook before_chat (prefix last user message).
type nativeBeforeChat struct{}

func (nativeBeforeChat) Name() string { return "native_bench_chat" }

func (nativeBeforeChat) BeforeChat(_ context.Context, req chatir.Request) (chatir.Request, error) {
	const prefix = "[wasm] "
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role != "user" {
			continue
		}
		content := req.Messages[i].Content
		if strings.HasPrefix(content, prefix) {
			break
		}
		req.Messages[i].Content = prefix + content
		break
	}
	return req, nil
}

func benchCallAllow() *sdkhook.CallContext {
	return &sdkhook.CallContext{
		Principal: sdkhook.Principal{
			OrganizationID: "org_bench",
			ProjectID:      "proj_bench",
			APIKeyID:       "key_bench",
			Kind:           "virtual",
			Name:           "bench",
		},
		Route: sdkhook.RouteContext{
			Model:    "gpt-4o-mini",
			Path:     "/v1/chat/completions",
			Stream:   false,
			Modality: "chat",
		},
		Tags:     map[string]string{"plan": "enterprise", "end-user-id": "u1"},
		Metadata: map[string]any{},
		Body:     []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`),
	}
}

func benchChatRequest() chatir.Request {
	return chatir.Request{
		Model:    "gpt-4o-mini",
		Messages: []chatir.Message{{Role: "user", Content: "hello benchmark"}},
	}
}

func BenchmarkBeforeCall_Native(b *testing.B) {
	h := nativeBeforeCall{}
	ctx := context.Background()
	// Warmup: avoid first-call / cache cold-start skew in reported ns/op.
	for i := 0; i < 64; i++ {
		if _, err := h.BeforeCall(ctx, benchCallAllow()); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := benchCallAllow()
		d, err := h.BeforeCall(ctx, call)
		if err != nil || !d.Allow {
			b.Fatalf("native: allow=%v err=%v", d.Allow, err)
		}
	}
}

func BenchmarkBeforeCall_WASM(b *testing.B) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(b), Config{Name: "bench", Timeout: time.Second})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = mod.Close(context.Background()) })
	h, err := NewBeforeCall(mod)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if _, err := h.BeforeCall(ctx, benchCallAllow()); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := benchCallAllow()
		d, err := h.BeforeCall(ctx, call)
		if err != nil || !d.Allow {
			b.Fatalf("wasm: allow=%v err=%v", d.Allow, err)
		}
	}
}

func BenchmarkBeforeCall_WASM_NoPool(b *testing.B) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(b), Config{Name: "bench-nopool", Timeout: time.Second, PoolSize: -1})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = mod.Close(context.Background()) })
	h, err := NewBeforeCall(mod)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if _, err := h.BeforeCall(ctx, benchCallAllow()); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := benchCallAllow()
		d, err := h.BeforeCall(ctx, call)
		if err != nil || !d.Allow {
			b.Fatalf("wasm nopool: allow=%v err=%v", d.Allow, err)
		}
	}
}

func BenchmarkBeforeCall_Native_Deny(b *testing.B) {
	h := nativeBeforeCall{}
	ctx := context.Background()
	for i := 0; i < 64; i++ {
		call := benchCallAllow()
		call.Tags["plan"] = "blocked"
		if _, err := h.BeforeCall(ctx, call); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := benchCallAllow()
		call.Tags["plan"] = "blocked"
		d, err := h.BeforeCall(ctx, call)
		if err != nil || d.Allow {
			b.Fatalf("native deny: allow=%v err=%v", d.Allow, err)
		}
	}
}

func BenchmarkBeforeCall_WASM_Deny(b *testing.B) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(b), Config{Name: "bench", Timeout: time.Second})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = mod.Close(context.Background()) })
	h, err := NewBeforeCall(mod)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		call := benchCallAllow()
		call.Tags["plan"] = "blocked"
		if _, err := h.BeforeCall(ctx, call); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		call := benchCallAllow()
		call.Tags["plan"] = "blocked"
		d, err := h.BeforeCall(ctx, call)
		if err != nil || d.Allow {
			b.Fatalf("wasm deny: allow=%v err=%v", d.Allow, err)
		}
	}
}

func BenchmarkBeforeChat_Native(b *testing.B) {
	h := nativeBeforeChat{}
	ctx := context.Background()
	for i := 0; i < 64; i++ {
		if _, err := h.BeforeChat(ctx, benchChatRequest()); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := h.BeforeChat(ctx, benchChatRequest())
		if err != nil || len(out.Messages) == 0 {
			b.Fatalf("native chat: err=%v messages=%d", err, len(out.Messages))
		}
	}
}

func BenchmarkBeforeChat_WASM(b *testing.B) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(b), Config{Name: "bench", Timeout: time.Second})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = mod.Close(context.Background()) })
	h, err := NewBeforeChat(mod)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if _, err := h.BeforeChat(ctx, benchChatRequest()); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := h.BeforeChat(ctx, benchChatRequest())
		if err != nil || len(out.Messages) == 0 {
			b.Fatalf("wasm chat: err=%v messages=%d", err, len(out.Messages))
		}
	}
}
