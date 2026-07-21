package dataplane

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"

	afiWasm "github.com/curefatih/afi/internal/adapters/wasm"
	"github.com/curefatih/afi/internal/snapshot"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// WasmRunner executes org-scoped snapshot WASM bindings via a module cache.
type WasmRunner struct {
	Cache *afiWasm.ModuleCache
	Log   *slog.Logger
}

func (r *WasmRunner) log() *slog.Logger {
	if r == nil || r.Log == nil {
		return slog.Default()
	}
	return r.Log
}

func filterWasmHooks(snap *snapshot.Snapshot, orgID, phase string) []snapshot.WasmHook {
	if snap == nil || orgID == "" {
		return nil
	}
	var out []snapshot.WasmHook
	for _, h := range snap.WasmHooks {
		if !h.Enabled || h.OrganizationID != orgID || h.Phase != phase {
			continue
		}
		out = append(out, h)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (r *WasmRunner) RunBeforeCall(ctx context.Context, snap *snapshot.Snapshot, call *CallContext) (CallDecision, error) {
	if r == nil || r.Cache == nil || call == nil {
		return sdkhook.Allow(), nil
	}
	for _, h := range filterWasmHooks(snap, call.Principal.OrganizationID, snapshot.WasmPhaseBeforeCall) {
		mod, err := r.Cache.Get(ctx, h.ModuleURI, h.Digest, "snap:"+h.ID)
		if err != nil {
			return CallDecision{}, err
		}
		hook, err := afiWasm.NewBeforeCallWithConfig(mod, rawConfig(h.Config))
		if err != nil {
			return CallDecision{}, err
		}
		d, err := hook.BeforeCall(ctx, call)
		if err != nil {
			return CallDecision{}, err
		}
		if !d.Allow {
			return d, nil
		}
	}
	return sdkhook.Allow(), nil
}

func (r *WasmRunner) RunBeforeChat(ctx context.Context, snap *snapshot.Snapshot, orgID string, body []byte) ([]byte, error) {
	if r == nil || r.Cache == nil {
		return body, nil
	}
	out := body
	for _, h := range filterWasmHooks(snap, orgID, snapshot.WasmPhaseBeforeChat) {
		mod, err := r.Cache.Get(ctx, h.ModuleURI, h.Digest, "snap:"+h.ID)
		if err != nil {
			return nil, err
		}
		hook, err := afiWasm.NewBeforeChatWithConfig(mod, rawConfig(h.Config))
		if err != nil {
			return nil, err
		}
		next, err := hook.BeforeChat(ctx, out)
		if err != nil {
			return nil, err
		}
		out = next
	}
	return out, nil
}

func (r *WasmRunner) RunAfterCall(ctx context.Context, snap *snapshot.Snapshot, call *CallContext, info AfterCallInfo) {
	if r == nil || r.Cache == nil || call == nil {
		return
	}
	for _, h := range filterWasmHooks(snap, call.Principal.OrganizationID, snapshot.WasmPhaseAfterCall) {
		mod, err := r.Cache.Get(ctx, h.ModuleURI, h.Digest, "snap:"+h.ID)
		if err != nil {
			r.log().Error("wasm after_call load", "hook", h.ID, "err", err)
			continue
		}
		hook, err := afiWasm.NewAfterCallWithConfig(mod, rawConfig(h.Config))
		if err != nil {
			r.log().Error("wasm after_call adapter", "hook", h.ID, "err", err)
			continue
		}
		if err := hook.AfterCall(ctx, call, info); err != nil {
			r.log().Error("wasm after_call", "hook", h.ID, "err", err)
		}
	}
}

func rawConfig(c json.RawMessage) json.RawMessage {
	if len(c) == 0 {
		return json.RawMessage(`{}`)
	}
	return c
}
