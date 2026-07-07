package proxy

import (
	"context"
	"time"

	"github.com/curefatih/afi/internal/config"
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
}

func NewHookRunner(cfg *config.Config) (*HookRunner, error) {
	return &HookRunner{
		timeout: time.Duration(cfg.Hooks.TimeoutMS) * time.Millisecond,
	}, nil
}

func (r *HookRunner) Run(ctx context.Context, hook HookName, reqCtx *RequestContext) {

}
