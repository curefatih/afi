package dataplane

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func newCallContext(key snapshot.APIKey, model, path, modality string, stream bool, body []byte, tags map[string]string) *CallContext {
	if tags == nil {
		tags = map[string]string{}
	}
	return &CallContext{
		Principal: Principal{
			OrganizationID: key.OrganizationID,
			ProjectID:      key.ProjectID,
			APIKeyID:       key.ID,
			Kind:           key.Kind,
			OwnerUserID:    key.OwnerUserID,
			Name:           key.Name,
		},
		Route: RouteContext{
			Model:    model,
			Path:     path,
			Stream:   stream,
			Modality: modality,
		},
		Tags:     tags,
		Metadata: map[string]any{},
		Body:     body,
	}
}

// gateCall runs CEL policy, then user BeforeCall hooks, then request quota.
// Built-ins are invoked here (not via p.Hooks) so reassigning Hooks cannot bypass them.
// Quota runs last among BeforeCall gates so a later deny does not consume request quota.
// ok=false means the response was already written. call.Body may be mutated.
func (p *Pipeline) gateCall(ctx context.Context, w http.ResponseWriter, snap *snapshot.Snapshot, call *CallContext) bool {
	if call.Metadata == nil {
		call.Metadata = map[string]any{}
	}

	// Policy → user hooks → quota (quota last: checkAndIncr has side effects).
	if !p.applyBeforeCall(ctx, w, &policyCallHook{p: p, snap: snap}, call) {
		return false
	}
	d, err := p.Hooks.RunBeforeCall(ctx, call)
	if err != nil {
		if p.Log != nil {
			p.Log.Error("before_call hook", "err", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "call hook failed", "type": "server_error"},
		})
		return false
	}
	if !d.Allow {
		writeCallDeny(w, d)
		return false
	}
	return p.applyBeforeCall(ctx, w, &quotaCallHook{p: p, snap: snap}, call)
}

func (p *Pipeline) applyBeforeCall(ctx context.Context, w http.ResponseWriter, h BeforeCallHook, call *CallContext) bool {
	d, err := h.BeforeCall(ctx, call)
	if err != nil {
		if p.Log != nil {
			p.Log.Error("before_call hook", "err", err, "hook", h.Name())
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": map[string]string{"message": "call hook failed", "type": "server_error"},
		})
		return false
	}
	if !d.Allow {
		writeCallDeny(w, d)
		return false
	}
	return true
}

// builtinHookInfos returns healthz entries for gates that always run in gateCall.
func builtinHookInfos() []HookInfo {
	return []HookInfo{
		{Name: "cel_policy", BeforeCall: true},
		{Name: "quota", BeforeCall: true},
	}
}

type policyCallHook struct {
	p    *Pipeline
	snap *snapshot.Snapshot
}

func (h *policyCallHook) Name() string { return "cel_policy" }

func (h *policyCallHook) BeforeCall(_ context.Context, call *CallContext) (CallDecision, error) {
	p := h.p
	if p == nil || p.Policies == nil {
		return sdkhook.Allow(), nil
	}
	snap := h.snap
	if snap == nil || len(snap.Policies) == 0 {
		return sdkhook.Allow(), nil
	}
	key := snapshot.APIKey{
		ID:             call.Principal.APIKeyID,
		OrganizationID: call.Principal.OrganizationID,
		ProjectID:      call.Principal.ProjectID,
		Kind:           call.Principal.Kind,
		OwnerUserID:    call.Principal.OwnerUserID,
		Name:           call.Principal.Name,
	}
	allowed, deniedName, err := p.Policies.Evaluate(snap.Policies, key, policy.Request{
		Model:  call.Route.Model,
		Path:   call.Route.Path,
		Stream: call.Route.Stream,
	})
	if err != nil {
		return CallDecision{}, err
	}
	if !allowed {
		msg := "request denied by policy"
		if deniedName != "" {
			msg = "request denied by policy: " + deniedName
		}
		return sdkhook.Deny(http.StatusForbidden, "policy_violation", msg), nil
	}
	return sdkhook.Allow(), nil
}

type quotaCallHook struct {
	p    *Pipeline
	snap *snapshot.Snapshot
}

func (h *quotaCallHook) Name() string { return "quota" }

func (h *quotaCallHook) BeforeCall(ctx context.Context, call *CallContext) (CallDecision, error) {
	p := h.p
	if p == nil || p.Counters == nil {
		return sdkhook.Allow(), nil
	}
	snap := h.snap
	if snap == nil {
		return sdkhook.Allow(), nil
	}
	key := snapshot.APIKey{
		ID:             call.Principal.APIKeyID,
		OrganizationID: call.Principal.OrganizationID,
		ProjectID:      call.Principal.ProjectID,
		Kind:           call.Principal.Kind,
		OwnerUserID:    call.Principal.OwnerUserID,
		Name:           call.Principal.Name,
	}
	denied, err := p.checkAndIncrRequests(ctx, snap, key)
	if err != nil {
		return CallDecision{}, err
	}
	if denied {
		return sdkhook.Deny(http.StatusTooManyRequests, "insufficient_quota", "quota exceeded"), nil
	}
	return sdkhook.Allow(), nil
}
