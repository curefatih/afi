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
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	defaultMaxMemoryPages = 512 // 512 * 64KiB = 32MiB
	defaultTimeout        = 100 * time.Millisecond
	defaultPoolSize       = 8
)

// Config controls sandbox limits and instance pooling for a loaded module.
type Config struct {
	Name           string
	Timeout        time.Duration
	MaxMemoryPages uint32
	// PoolSize is the max number of reused guest instances (default 8).
	// Set to 1 for a single reusable instance; set to 0 to use the default.
	// Negative disables pooling (instantiate+close every invoke; useful for benches).
	PoolSize int
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
	if c.PoolSize == 0 {
		c.PoolSize = defaultPoolSize
	}
	return c
}

type pooledInst struct {
	mod    api.Module
	malloc api.Function
	free   api.Function
	fns    map[string]api.Function
}

// Module is a compiled WASM hook module with an optional instance pool.
type Module struct {
	rt       wazero.Runtime
	compiled wazero.CompiledModule
	cfg      Config

	mu     sync.Mutex
	closed bool
	pool   chan *pooledInst // nil when pooling disabled
	seq    atomic.Uint64
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
			if _, ok3 := exp["after_call"]; !ok3 {
				_ = compiled.Close(ctx)
				_ = rt.Close(ctx)
				return nil, fmt.Errorf("wasm: need before_call, before_chat, and/or after_call export")
			}
		}
	}

	m := &Module{rt: rt, compiled: compiled, cfg: cfg}
	if cfg.PoolSize > 0 {
		m.pool = make(chan *pooledInst, cfg.PoolSize)
	}
	return m, nil
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

// Close drains the pool and releases the runtime.
func (m *Module) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true

	if m.pool != nil {
		close(m.pool)
		for inst := range m.pool {
			_ = inst.mod.Close(ctx)
		}
		m.pool = nil
	}

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

func (m *Module) newInst(ctx context.Context) (*pooledInst, error) {
	id := m.seq.Add(1)
	modCfg := wazero.NewModuleConfig().
		WithName(fmt.Sprintf("%s-%d", m.cfg.Name, id)).
		WithStartFunctions("_initialize")
	mod, err := m.rt.InstantiateModule(ctx, m.compiled, modCfg)
	if err != nil {
		return nil, fmt.Errorf("wasm: instantiate: %w", err)
	}
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	if malloc == nil || free == nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wasm: exports missing after instantiate")
	}
	fns := make(map[string]api.Function, 2)
	if f := mod.ExportedFunction("before_call"); f != nil {
		fns["before_call"] = f
	}
	if f := mod.ExportedFunction("before_chat"); f != nil {
		fns["before_chat"] = f
	}
	if f := mod.ExportedFunction("after_call"); f != nil {
		fns["after_call"] = f
	}
	return &pooledInst{mod: mod, malloc: malloc, free: free, fns: fns}, nil
}

func (m *Module) acquire(ctx context.Context) (*pooledInst, error) {
	m.mu.Lock()
	if m.closed || m.rt == nil || m.compiled == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("wasm: module closed")
	}
	pool := m.pool
	m.mu.Unlock()

	if pool != nil {
		select {
		case inst, ok := <-pool:
			if ok && inst != nil {
				return inst, nil
			}
		default:
		}
	}

	m.mu.Lock()
	if m.closed || m.rt == nil || m.compiled == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("wasm: module closed")
	}
	m.mu.Unlock()
	return m.newInst(ctx)
}

func (m *Module) release(inst *pooledInst) {
	if inst == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.pool == nil {
		_ = inst.mod.Close(context.Background())
		return
	}
	select {
	case m.pool <- inst:
	default:
		_ = inst.mod.Close(context.Background())
	}
}

// invokeJSON allocates input in guest memory, calls export(name), returns output bytes.
func (m *Module) invokeJSON(ctx context.Context, exportName string, in []byte) ([]byte, error) {
	if m == nil {
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

	inst, err := m.acquire(ctx)
	if err != nil {
		return nil, err
	}
	reuse := m.cfg.PoolSize > 0
	defer func() {
		if reuse {
			m.release(inst)
		} else {
			_ = inst.mod.Close(context.Background())
		}
	}()

	fn := inst.fns[exportName]
	if fn == nil {
		return nil, fmt.Errorf("wasm: missing export %q", exportName)
	}

	inSize := uint64(len(in))
	var inPtr uint64
	if inSize > 0 {
		results, err := inst.malloc.Call(ctx, inSize)
		if err != nil {
			return nil, fmt.Errorf("wasm: malloc: %w", err)
		}
		inPtr = results[0]
		defer func() { _, _ = inst.free.Call(ctx, inPtr) }()
		if !inst.mod.Memory().Write(uint32(inPtr), in) {
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
	defer func() { _, _ = inst.free.Call(ctx, uint64(outPtr)) }()

	out, ok := inst.mod.Memory().Read(outPtr, outSize)
	if !ok {
		return nil, fmt.Errorf("wasm: read output out of range")
	}
	cp := make([]byte, len(out))
	copy(cp, out)
	return cp, nil
}
