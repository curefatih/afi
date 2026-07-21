// Package wasm hosts TinyGo-compiled lifecycle hooks via wazero.
//
// It adapts guest exports (before_call / before_chat) to sdk/hook interfaces.
// Control-plane domains must not import this package; execution stays in the data plane.
package wasm

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	defaultMaxMemoryPages = 512 // 512 * 64KiB = 32MiB
	defaultTimeout        = 100 * time.Millisecond
)

// Config controls sandbox limits for a loaded module.
type Config struct {
	Name           string
	Timeout        time.Duration
	MaxMemoryPages uint32
}

func (c Config) withDefaults() Config {
	if c.Name == "" {
		c.Name = "wasm"
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	if c.MaxMemoryPages == 0 {
		c.MaxMemoryPages = defaultMaxMemoryPages
	}
	return c
}

// Module is a compiled WASM hook module.
type Module struct {
	rt       wazero.Runtime
	compiled wazero.CompiledModule
	cfg      Config
	mu       sync.Mutex
}

// Compile loads and compiles wasmBytes. Call Close when done.
func Compile(ctx context.Context, wasmBytes []byte, cfg Config) (*Module, error) {
	cfg = cfg.withDefaults()
	if len(wasmBytes) == 0 {
		return nil, fmt.Errorf("wasm: empty module")
	}

	rtCfg := wazero.NewRuntimeConfig().WithMemoryLimitPages(cfg.MaxMemoryPages)
	rt := wazero.NewRuntimeWithConfig(ctx, rtCfg)

	// TinyGo may import WASI; provide a no-op preview1 (no FS/net).
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("wasm: wasi: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("wasm: compile: %w", err)
	}

	// Validate required exports exist at load time.
	exp := compiled.ExportedFunctions()
	for _, name := range []string{"malloc", "free"} {
		if _, ok := exp[name]; !ok {
			_ = compiled.Close(ctx)
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("wasm: missing export %q", name)
		}
	}
	if _, ok := exp["before_call"]; !ok {
		if _, ok2 := exp["before_chat"]; !ok2 {
			_ = compiled.Close(ctx)
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("wasm: need before_call and/or before_chat export")
		}
	}

	return &Module{rt: rt, compiled: compiled, cfg: cfg}, nil
}

// CompileFile reads path and compiles.
func CompileFile(ctx context.Context, path string, cfg Config) (*Module, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wasm: read %s: %w", path, err)
	}
	if cfg.Name == "" {
		cfg.Name = "wasm:" + path
	}
	return Compile(ctx, b, cfg)
}

// Close releases the runtime.
func (m *Module) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	if m.compiled != nil {
		err = m.compiled.Close(ctx)
		m.compiled = nil
	}
	if m.rt != nil {
		if e := m.rt.Close(ctx); e != nil && err == nil {
			err = e
		}
		m.rt = nil
	}
	return err
}

func (m *Module) hasExport(name string) bool {
	if m == nil || m.compiled == nil {
		return false
	}
	_, ok := m.compiled.ExportedFunctions()[name]
	return ok
}

// invokeJSON allocates input in guest memory, calls export(name), returns output bytes.
func (m *Module) invokeJSON(ctx context.Context, exportName string, in []byte) ([]byte, error) {
	if m == nil || m.rt == nil || m.compiled == nil {
		return nil, fmt.Errorf("wasm: module closed")
	}
	if !m.hasExport(exportName) {
		return nil, fmt.Errorf("wasm: missing export %q", exportName)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, m.cfg.Timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// TinyGo -buildmode=c-shared exports _initialize (not _start).
	modCfg := wazero.NewModuleConfig().WithName("").WithStartFunctions("_initialize")
	mod, err := m.rt.InstantiateModule(ctx, m.compiled, modCfg)
	if err != nil {
		return nil, fmt.Errorf("wasm: instantiate: %w", err)
	}
	defer mod.Close(ctx)

	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	fn := mod.ExportedFunction(exportName)
	if malloc == nil || free == nil || fn == nil {
		return nil, fmt.Errorf("wasm: exports missing after instantiate")
	}

	inSize := uint64(len(in))
	var inPtr uint64
	if inSize > 0 {
		results, err := malloc.Call(ctx, inSize)
		if err != nil {
			return nil, fmt.Errorf("wasm: malloc: %w", err)
		}
		inPtr = results[0]
		defer func() { _, _ = free.Call(ctx, inPtr) }()
		if !mod.Memory().Write(uint32(inPtr), in) {
			return nil, fmt.Errorf("wasm: write input out of range")
		}
	}

	results, err := fn.Call(ctx, inPtr, inSize)
	if err != nil {
		return nil, fmt.Errorf("wasm: %s: %w", exportName, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("wasm: %s: empty result", exportName)
	}
	packed := results[0]
	outPtr := uint32(packed >> 32)
	outSize := uint32(packed)
	if outPtr == 0 || outSize == 0 {
		return []byte{}, nil
	}
	defer func() { _, _ = free.Call(ctx, uint64(outPtr)) }()

	out, ok := mod.Memory().Read(outPtr, outSize)
	if !ok {
		return nil, fmt.Errorf("wasm: read output out of range")
	}
	cp := make([]byte, len(out))
	copy(cp, out)
	return cp, nil
}
