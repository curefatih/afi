// Package tagquota is an **example** BeforeCall hook that rate-limits by an X-AFI-Tags value.
//
// It is not registered by the gateway. Copy or adapt it when you want per-tag limits:
//
//	import "github.com/curefatih/afi/extensions/tagquota"
//
//	hooks.RegisterBeforeCall(tagquota.New(pipeline.Counters, tagquota.Config{
//	    TagKey: "end-user-id",
//	    Limit:  100,
//	    Window: snapshot.WindowTotal,
//	}))
//
// Each distinct tag value gets its own counter. Requests without the tag key are skipped.
package tagquota

import (
	"context"
	"fmt"
	"net/http"

	"github.com/curefatih/afi/internal/dataplane"
	"github.com/curefatih/afi/internal/snapshot"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// Name is the hook identifier if you register this example.
const Name = "tag_quota"

// Config controls per-tag request limits.
type Config struct {
	TagKey string // e.g. end-user-id
	Limit  int64
	Window string // total | minute | hour | day; empty → total
	// ScopeParent is "project" (default when principal has a project) or "organization".
	ScopeParent string
}

// Hook implements sdkhook.BeforeCallHook.
type Hook struct {
	Counters dataplane.CounterStore
	Config   Config
}

// New builds a tag quota example hook. Counters may be nil (hook becomes a no-op).
func New(counters dataplane.CounterStore, cfg Config) *Hook {
	if cfg.TagKey == "" {
		cfg.TagKey = "end-user-id"
	}
	if cfg.Limit <= 0 {
		cfg.Limit = 100
	}
	if cfg.Window == "" {
		cfg.Window = snapshot.WindowTotal
	}
	if cfg.ScopeParent == "" {
		cfg.ScopeParent = "project"
	}
	return &Hook{Counters: counters, Config: cfg}
}

func (Hook) Name() string { return Name }

func (h *Hook) BeforeCall(ctx context.Context, call *sdkhook.CallContext) (sdkhook.CallDecision, error) {
	if h == nil || h.Counters == nil || call == nil {
		return sdkhook.Allow(), nil
	}
	val, ok := call.Tags[h.Config.TagKey]
	if !ok || val == "" {
		return sdkhook.Allow(), nil
	}

	parentType, parentID := h.parentScope(call.Principal)
	if parentID == "" {
		return sdkhook.Allow(), nil
	}

	scopeID := fmt.Sprintf("%s:%s:%s:%s", parentType, parentID, h.Config.TagKey, val)
	used, err := h.Counters.Get(ctx, "tag", scopeID, snapshot.MetricRequests, h.Config.Window)
	if err != nil {
		return sdkhook.CallDecision{}, err
	}
	if used >= h.Config.Limit {
		return sdkhook.Deny(http.StatusTooManyRequests, "insufficient_quota",
			fmt.Sprintf("tag quota exceeded for %s=%s", h.Config.TagKey, val)), nil
	}
	if _, err := h.Counters.Incr(ctx, "tag", scopeID, snapshot.MetricRequests, h.Config.Window, 1); err != nil {
		return sdkhook.CallDecision{}, err
	}
	return sdkhook.Allow(), nil
}

func (h *Hook) parentScope(p sdkhook.Principal) (scopeType, scopeID string) {
	switch h.Config.ScopeParent {
	case "organization":
		return snapshot.ScopeOrganization, p.OrganizationID
	default:
		if p.ProjectID != "" {
			return snapshot.ScopeProject, p.ProjectID
		}
		return snapshot.ScopeOrganization, p.OrganizationID
	}
}

var _ sdkhook.BeforeCallHook = (*Hook)(nil)
