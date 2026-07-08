package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/curefatih/afi/internal/config"
	"github.com/dop251/goja"
)

type HookName string

const (
	HookOnRequest        HookName = "onRequest"
	HookOnBeforeUpstream HookName = "onBeforeUpstream"
	HookOnResponse       HookName = "onResponse"
)

type HookRunner struct {
	timeout time.Duration
	// not sure about this, but we can add it later. script or multiple strategies?
	// scripts []middlewareScript
	scripts []hookScript
}

type hookScript struct {
	path     string
	hookName string
	source   string
}

func NewHookRunner(cfg *config.Config) (*HookRunner, error) {
	r := &HookRunner{
		timeout: time.Duration(cfg.Hooks.TimeoutMS) * time.Millisecond,
	}

	for _, spec := range cfg.Hooks.HookSpecs {
		hook := HookName(spec.Name)
		if hook != HookOnRequest && hook != HookOnBeforeUpstream && hook != HookOnResponse {
			return nil, fmt.Errorf("unknown hook %q in hook %q", spec.Name, spec.Path)
		}

		src, err := os.ReadFile(spec.Path)
		if err != nil {
			return nil, fmt.Errorf("read hook %q: %w", spec.Path, err)
		}

		vm := goja.New()
		if _, err := vm.RunString(string(src)); err != nil {
			return nil, fmt.Errorf("compile hook %q: %w", spec.Path, err)
		}
		if vm.Get(spec.Name) == nil {
			return nil, fmt.Errorf("hook %q: function %q not found", spec.Path, spec.Name)
		}

		r.scripts = append(r.scripts, hookScript{
			path:     spec.Path,
			hookName: spec.Name,
			source:   string(src),
		})
	}
	return r, nil
}

func (r *HookRunner) Run(ctx context.Context, hook HookName, reqCtx *RequestContext) {
	for _, script := range r.scripts {
		if script.hookName != string(hook) {
			continue
		}
		if err := r.runWithTimeout(ctx, script, reqCtx); err != nil {
			slog.Warn("hook bypassed", "hook", hook, "path", script.path, "error", err)
		}
	}
}

func (r *HookRunner) runWithTimeout(ctx context.Context, script hookScript, reqCtx *RequestContext) error {
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	type result struct {
		err error
	}
	ch := make(chan result, 1)

	go func() {
		ch <- result{err: r.invoke(script, reqCtx)}
	}()

	select {
	case <-runCtx.Done():
		return fmt.Errorf("hook timeout after %s", r.timeout)
	case res := <-ch:
		return res.err
	}
}

func (r *HookRunner) invoke(script hookScript, reqCtx *RequestContext) error {
	vm := goja.New()
	if _, err := vm.RunString(script.source); err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	fnValue := vm.Get(script.hookName)
	fn, ok := goja.AssertFunction(fnValue)
	if !ok {
		return fmt.Errorf("function %q not found", script.hookName)
	}

	out, err := fn(goja.Undefined(), vm.ToValue(reqCtx.ToHookMap()))
	if err != nil {
		return err
	}
	if out != nil && !goja.IsUndefined(out) && !goja.IsNull(out) {
		if m, ok := out.Export().(map[string]any); ok {
			reqCtx.ApplyHookMap(m)
		}
	}
	return nil
}
