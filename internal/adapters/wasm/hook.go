package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// BeforeCallAdapter runs the guest before_call export.
type BeforeCallAdapter struct {
	mod    *Module
	name   string
	config json.RawMessage
}

// NewBeforeCall returns a BeforeCallHook backed by mod.
func NewBeforeCall(mod *Module) (*BeforeCallAdapter, error) {
	return NewBeforeCallWithConfig(mod, nil)
}

// NewBeforeCallWithConfig is like NewBeforeCall but passes binding config to the guest.
func NewBeforeCallWithConfig(mod *Module, config json.RawMessage) (*BeforeCallAdapter, error) {
	if mod == nil {
		return nil, fmt.Errorf("wasm: nil module")
	}
	if !mod.hasExport("before_call") {
		return nil, fmt.Errorf("wasm: module has no before_call export")
	}
	return &BeforeCallAdapter{mod: mod, name: mod.cfg.Name, config: config}, nil
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
	in, err := encodeBeforeCallIn(call, a.config)
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
	mod    *Module
	name   string
	config json.RawMessage
}

// NewBeforeChat returns a ChatHook backed by mod.
func NewBeforeChat(mod *Module) (*BeforeChatAdapter, error) {
	return NewBeforeChatWithConfig(mod, nil)
}

// NewBeforeChatWithConfig is like NewBeforeChat but passes binding config to the guest.
func NewBeforeChatWithConfig(mod *Module, config json.RawMessage) (*BeforeChatAdapter, error) {
	if mod == nil {
		return nil, fmt.Errorf("wasm: nil module")
	}
	if !mod.hasExport("before_chat") {
		return nil, fmt.Errorf("wasm: module has no before_chat export")
	}
	return &BeforeChatAdapter{mod: mod, name: mod.cfg.Name + ":before_chat", config: config}, nil
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
	in, err := encodeBeforeChatIn(body, a.config)
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

// AfterCallAdapter runs the guest after_call export.
type AfterCallAdapter struct {
	mod    *Module
	name   string
	config json.RawMessage
}

// NewAfterCall returns an AfterCallHook backed by mod.
func NewAfterCall(mod *Module) (*AfterCallAdapter, error) {
	return NewAfterCallWithConfig(mod, nil)
}

// NewAfterCallWithConfig is like NewAfterCall but passes binding config to the guest.
func NewAfterCallWithConfig(mod *Module, config json.RawMessage) (*AfterCallAdapter, error) {
	if mod == nil {
		return nil, fmt.Errorf("wasm: nil module")
	}
	if !mod.hasExport("after_call") {
		return nil, fmt.Errorf("wasm: module has no after_call export")
	}
	return &AfterCallAdapter{mod: mod, name: mod.cfg.Name + ":after_call", config: config}, nil
}

func (a *AfterCallAdapter) Name() string {
	if a == nil || a.name == "" {
		return "wasm:after_call"
	}
	return a.name
}

func (a *AfterCallAdapter) AfterCall(ctx context.Context, call *sdkhook.CallContext, info sdkhook.AfterCallInfo) error {
	if a == nil || a.mod == nil || call == nil {
		return nil
	}
	in, err := encodeAfterCallIn(call, info, a.config)
	if err != nil {
		return fmt.Errorf("wasm: encode: %w", err)
	}
	_, err = a.mod.invokeJSON(ctx, "after_call", in)
	return err
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
	_ sdkhook.AfterCallHook  = (*AfterCallAdapter)(nil)
)
