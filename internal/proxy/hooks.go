package proxy

import "time"

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
