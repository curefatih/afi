package dataplane

import (
	"context"
	"sync"
)

// ChatHook mutates the OpenAI chat request body before provider dispatch.
type ChatHook interface {
	Name() string
	BeforeChat(ctx context.Context, body []byte) ([]byte, error)
}

// AfterChatInfo is passed to AfterChatHook after a chat attempt finishes.
type AfterChatInfo struct {
	Model        string
	Status       string
	LatencyMs    int64
	ProviderType string
	TargetModel  string
}

// AfterChatHook runs after the chat attempt completes (success or error).
type AfterChatHook interface {
	Name() string
	AfterChat(ctx context.Context, info AfterChatInfo) error
}

// HookChain runs BeforeChat / AfterChat hooks in registration order.
type HookChain struct {
	mu     sync.RWMutex
	before []ChatHook
	after  []AfterChatHook
}

func NewHookChain() *HookChain {
	return &HookChain{}
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

// RegisterHook registers a value that may implement ChatHook and/or AfterChatHook.
func (c *HookChain) RegisterHook(h any) *HookChain {
	if h == nil {
		return c
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
	for _, h := range c.before {
		n := h.Name()
		if _, ok := byName[n]; !ok {
			byName[n] = &HookInfo{Name: n}
			order = append(order, n)
		}
		byName[n].BeforeChat = true
	}
	for _, h := range c.after {
		n := h.Name()
		if _, ok := byName[n]; !ok {
			byName[n] = &HookInfo{Name: n}
			order = append(order, n)
		}
		byName[n].AfterChat = true
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
