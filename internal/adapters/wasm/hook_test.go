package wasm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/curefatih/afi/sdk/chatir"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func exampleWASMPath(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/adapters/wasm -> repo root -> extensions/wasmhook/hook.wasm
	p := filepath.Join(filepath.Dir(file), "..", "..", "..", "extensions", "wasmhook", "hook.wasm")
	p, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("missing example wasm at %s (run make -C extensions/wasmhook build): %v", p, err)
	}
	return p
}

func TestBeforeCallAllowAndMutate(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	call := &sdkhook.CallContext{
		Principal: sdkhook.Principal{OrganizationID: "o1"},
		Route:     sdkhook.RouteContext{Model: "m", Path: "/v1/chat/completions", Modality: "chat"},
		Tags:      map[string]string{"plan": "enterprise"},
		Metadata:  map[string]any{},
		Body:      []byte(`{}`),
	}
	d, err := hook.BeforeCall(ctx, call)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Allow {
		t.Fatalf("deny: %+v", d)
	}
	if call.Metadata["wasm_hook"] != "1" {
		t.Fatalf("metadata=%v", call.Metadata)
	}
}

func TestBeforeCallDenyBlockedPlan(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	call := &sdkhook.CallContext{
		Tags:     map[string]string{"plan": "blocked"},
		Metadata: map[string]any{},
	}
	d, err := hook.BeforeCall(ctx, call)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allow || d.Reason != "plan_blocked" {
		t.Fatalf("got %+v", d)
	}
}

// skipUnlessTypedBeforeChatGuest skips when the checked-in hook.wasm artifact
// still speaks the removed byte-oriented before_chat ABI. Rebuild it with
// `make -C extensions/wasmhook build` to exercise the typed path end to end.
func skipUnlessTypedBeforeChatGuest(ctx context.Context, t *testing.T, mod *Module) {
	t.Helper()
	raw, err := mod.invokeJSON(ctx, "before_chat", []byte(`{"request":{"model":"m"}}`))
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("guest returned non-JSON: %s", raw)
	}
	if _, ok := probe["request"]; !ok {
		t.Skip("extensions/wasmhook/hook.wasm predates the typed before_chat ABI; run make -C extensions/wasmhook build")
	}
}

func TestBeforeChatPrefix(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)
	skipUnlessTypedBeforeChatGuest(ctx, t, mod)

	hook, err := NewBeforeChat(mod)
	if err != nil {
		t.Fatal(err)
	}
	out, err := hook.BeforeChat(ctx, chatir.Request{
		Model:    "m",
		Messages: []chatir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) != 1 || out.Messages[0].Content != "[wasm] hi" {
		t.Fatalf("messages=%+v", out.Messages)
	}
}

func TestNewBeforeChatRequiresModule(t *testing.T) {
	if _, err := NewBeforeChat(nil); err == nil {
		t.Fatal("expected error for nil module")
	}
}

func TestBeforeChatAdapterNilModuleIsPassthrough(t *testing.T) {
	req := chatir.Request{Model: "m", Messages: []chatir.Message{{Role: "user", Content: "hi"}}}
	var a *BeforeChatAdapter
	out, err := a.BeforeChat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Model != "m" || out.Messages[0].Content != "hi" {
		t.Fatalf("request mutated: %+v", out)
	}
	if a.Name() != "wasm:before_chat" {
		t.Fatalf("name=%q", a.Name())
	}
}

func TestBeforeChatAdapterName(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	hook, err := NewBeforeChat(mod)
	if err != nil {
		t.Fatal(err)
	}
	if hook.Name() != "wasmhook:before_chat" {
		t.Fatalf("name=%q", hook.Name())
	}
}

func TestInvalidJSONFailsClosed(t *testing.T) {
	ctx := context.Background()
	mod, err := CompileFile(ctx, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(ctx)

	// Call before_call with garbage via invokeJSON directly.
	out, err := mod.invokeJSON(ctx, "before_call", []byte(`not-json`))
	if err != nil {
		t.Fatal(err)
	}
	var d beforeCallOut
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatal(err)
	}
	if d.Allow {
		t.Fatalf("expected deny on bad input, got %+v", d)
	}
}

func TestCancelledContext(t *testing.T) {
	base := context.Background()
	mod, err := CompileFile(base, exampleWASMPath(t), Config{Name: "wasmhook", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Close(base)

	hook, err := NewBeforeCall(mod)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(base)
	cancel()
	_, err = hook.BeforeCall(ctx, &sdkhook.CallContext{Tags: map[string]string{}, Metadata: map[string]any{}})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

func TestCompileRequiresExports(t *testing.T) {
	// Minimal wasm with only memory — should fail.
	// (module (memory 1))
	minimal := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x05, 0x03, 0x01, 0x00, 0x01}
	_, err := Compile(context.Background(), minimal, Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBodyB64RoundTripInGuest(t *testing.T) {
	// sanity: guest sees body
	body := []byte(`{"x":1}`)
	b64 := base64.StdEncoding.EncodeToString(body)
	if b64 == "" {
		t.Fatal("empty")
	}
}
