package wasm

import (
	"context"
	"fmt"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// BeforeCallAdapter runs the guest before_call export.
type BeforeCallAdapter struct {
	mod  *Module
	name string
}

// NewBeforeCall returns a BeforeCallHook backed by mod.
// The module must export before_call.
func NewBeforeCall(mod *Module) (*BeforeCallAdapter, error) {
	if mod == nil {
		return nil, fmt.Errorf("wasm: nil module")
	}
	if !mod.hasExport("before_call") {
		return nil, fmt.Errorf("wasm: module has no before_call export")
	}
	return &BeforeCallAdapter{mod: mod, name: mod.cfg.Name}, nil
}

func (a *BeforeCallAdapter) Name() string {
	if a == nil || a.name == "" {
		return "wasm"
	}
	return a.name
}

func (a *BeforeCallAdapter) BeforeCall(ctx context.Context, call *sdkhook.CallContext) (sdkhook.CallDecision, error) {
	if a == nil || a.mod == nil {
		return sdkhook.Allow(), nil
	}
	if call == nil {
		return sdkhook.Allow(), nil
	}
	in, err := encodeBeforeCallIn(call)
	if err != nil {
		return sdkhook.CallDecision{}, fmt.Errorf("wasm: encode: %w", err)
	}
	out, err := a.mod.invokeJSON(ctx, "before_call", in)
	if err != nil {
		return sdkhook.CallDecision{}, err
	}
	if len(out) == 0 {
		return sdkhook.Allow(), nil
	}
	return applyBeforeCallOut(call, out)
}

// BeforeChatAdapter runs the guest before_chat export.
type BeforeChatAdapter struct {
	mod  *Module
	name string
}

// NewBeforeChat returns a ChatHook backed by mod.
func NewBeforeChat(mod *Module) (*BeforeChatAdapter, error) {
	if mod == nil {
		return nil, fmt.Errorf("wasm: nil module")
	}
	if !mod.hasExport("before_chat") {
		return nil, fmt.Errorf("wasm: module has no before_chat export")
	}
	return &BeforeChatAdapter{mod: mod, name: mod.cfg.Name + ":before_chat"}, nil
}

func (a *BeforeChatAdapter) Name() string {
	if a == nil || a.name == "" {
		return "wasm:before_chat"
	}
	return a.name
}

func (a *BeforeChatAdapter) BeforeChat(ctx context.Context, body []byte) ([]byte, error) {
	if a == nil || a.mod == nil {
		return body, nil
	}
	in, err := encodeBeforeChatIn(body)
	if err != nil {
		return nil, fmt.Errorf("wasm: encode: %w", err)
	}
	out, err := a.mod.invokeJSON(ctx, "before_chat", in)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return body, nil
	}
	decoded, err := decodeBeforeChatOut(out)
	if err != nil {
		return nil, fmt.Errorf("wasm: decode: %w", err)
	}
	if decoded == nil {
		return body, nil
	}
	return decoded, nil
}

// LoadBeforeCall compiles path and returns a BeforeCallHook.
func LoadBeforeCall(ctx context.Context, path string, cfg Config) (sdkhook.BeforeCallHook, *Module, error) {
	mod, err := CompileFile(ctx, path, cfg)
	if err != nil {
		return nil, nil, err
	}
	hook, err := NewBeforeCall(mod)
	if err != nil {
		_ = mod.Close(ctx)
		return nil, nil, err
	}
	return hook, mod, nil
}

var (
	_ sdkhook.BeforeCallHook = (*BeforeCallAdapter)(nil)
	_ sdkhook.ChatHook       = (*BeforeChatAdapter)(nil)
)
