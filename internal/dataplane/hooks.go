package dataplane

import (
	"context"
	"net/http"
	"sync"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// Re-export SDK hook types so existing extensions can keep importing dataplane.
type (
	CallContext    = sdkhook.CallContext
	CallDecision   = sdkhook.CallDecision
	Principal      = sdkhook.Principal
	RouteContext   = sdkhook.RouteContext
	AfterCallInfo  = sdkhook.AfterCallInfo
	AfterChatInfo  = sdkhook.AfterChatInfo
	BeforeCallHook = sdkhook.BeforeCallHook
	AfterCallHook  = sdkhook.AfterCallHook
	ChatHook       = sdkhook.ChatHook
	AfterChatHook  = sdkhook.AfterChatHook
)

// HookChain runs BeforeCall / AfterCall / BeforeChat / AfterChat hooks in registration order.
type HookChain struct {
	mu         sync.RWMutex
	beforeCall []BeforeCallHook
	afterCall  []AfterCallHook
	before     []ChatHook
	after      []AfterChatHook
}

func NewHookChain() *HookChain {
	return &HookChain{}
}

// RegisterBeforeCall adds a BeforeCall hook (appended; runs after earlier entries).
func (c *HookChain) RegisterBeforeCall(h BeforeCallHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beforeCall = append(c.beforeCall, h)
	return c
}

// PrependBeforeCall inserts a BeforeCall hook at the front of the chain.
func (c *HookChain) PrependBeforeCall(h BeforeCallHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beforeCall = append([]BeforeCallHook{h}, c.beforeCall...)
	return c
}

// RegisterAfterCall adds an AfterCall hook.
func (c *HookChain) RegisterAfterCall(h AfterCallHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.afterCall = append(c.afterCall, h)
	return c
}

// Register adds a BeforeChat hook.
func (c *HookChain) Register(h ChatHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.before = append(c.before, h)
	return c
}

// RegisterAfter adds an AfterChat hook.
func (c *HookChain) RegisterAfter(h AfterChatHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.after = append(c.after, h)
	return c
}

// RegisterHook registers a value that may implement any of the hook interfaces.
func (c *HookChain) RegisterHook(h any) *HookChain {
	if h == nil {
		return c
	}
	if bc, ok := h.(BeforeCallHook); ok {
		c.RegisterBeforeCall(bc)
	}
	if ac, ok := h.(AfterCallHook); ok {
		c.RegisterAfterCall(ac)
	}
	if b, ok := h.(ChatHook); ok {
		c.Register(b)
	}
	if a, ok := h.(AfterChatHook); ok {
		c.RegisterAfter(a)
	}
	return c
}

// HookInfo describes a registered hook for healthz / UI.
type HookInfo struct {
	Name       string `json:"name"`
	BeforeCall bool   `json:"before_call"`
	AfterCall  bool   `json:"after_call"`
	BeforeChat bool   `json:"before_chat"`
	AfterChat  bool   `json:"after_chat"`
}

func (c *HookChain) Infos() []HookInfo {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	byName := map[string]*HookInfo{}
	order := make([]string, 0)
	add := func(name string) *HookInfo {
		if _, ok := byName[name]; !ok {
			byName[name] = &HookInfo{Name: name}
			order = append(order, name)
		}
		return byName[name]
	}
	for _, h := range c.beforeCall {
		add(h.Name()).BeforeCall = true
	}
	for _, h := range c.afterCall {
		add(h.Name()).AfterCall = true
	}
	for _, h := range c.before {
		add(h.Name()).BeforeChat = true
	}
	for _, h := range c.after {
		add(h.Name()).AfterChat = true
	}
	out := make([]HookInfo, 0, len(order))
	for _, n := range order {
		out = append(out, *byName[n])
	}
	return out
}

func (c *HookChain) Names() []string {
	infos := c.Infos()
	out := make([]string, 0, len(infos))
	for _, i := range infos {
		out = append(out, i.Name)
	}
	return out
}

// RunBeforeCall runs BeforeCall hooks. First deny wins. Mutates call in place.
func (c *HookChain) RunBeforeCall(ctx context.Context, call *CallContext) (CallDecision, error) {
	if c == nil || call == nil {
		return sdkhook.Allow(), nil
	}
	c.mu.RLock()
	hooks := append([]BeforeCallHook(nil), c.beforeCall...)
	c.mu.RUnlock()
	for _, h := range hooks {
		d, err := h.BeforeCall(ctx, call)
		if err != nil {
			return CallDecision{}, err
		}
		if !d.Allow {
			return d, nil
		}
	}
	return sdkhook.Allow(), nil
}

// RunAfterCall runs AfterCall hooks (errors ignored).
func (c *HookChain) RunAfterCall(ctx context.Context, call *CallContext, info AfterCallInfo) {
	if c == nil {
		return
	}
	c.mu.RLock()
	hooks := append([]AfterCallHook(nil), c.afterCall...)
	c.mu.RUnlock()
	for _, h := range hooks {
		_ = h.AfterCall(ctx, call, info)
	}
}

func (c *HookChain) RunBeforeChat(ctx context.Context, body []byte) ([]byte, error) {
	if c == nil {
		return body, nil
	}
	c.mu.RLock()
	hooks := append([]ChatHook(nil), c.before...)
	c.mu.RUnlock()
	var err error
	for _, h := range hooks {
		body, err = h.BeforeChat(ctx, body)
		if err != nil {
			return body, err
		}
	}
	return body, nil
}

func (c *HookChain) RunAfterChat(ctx context.Context, info AfterChatInfo) {
	if c == nil {
		return
	}
	c.mu.RLock()
	hooks := append([]AfterChatHook(nil), c.after...)
	c.mu.RUnlock()
	for _, h := range hooks {
		_ = h.AfterChat(ctx, info)
	}
}

func writeCallDeny(w http.ResponseWriter, d CallDecision) {
	status := d.Status
	if status == 0 {
		status = http.StatusForbidden
	}
	for k, v := range d.Headers {
		w.Header().Set(k, v)
	}
	reason := d.Reason
	if reason == "" {
		reason = "policy_violation"
	}
	msg := d.Message
	if msg == "" {
		msg = reason
	}
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"message": msg,
			"type":    reason,
			"code":    reason,
		},
	})
}
