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

// HookChain runs BeforeChat hooks in registration order.
type HookChain struct {
	mu    sync.RWMutex
	hooks []ChatHook
}

func NewHookChain() *HookChain {
	return &HookChain{}
}

func (c *HookChain) Register(h ChatHook) *HookChain {
	if h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks = append(c.hooks, h)
	return c
}

func (c *HookChain) Names() []string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.hooks))
	for _, h := range c.hooks {
		out = append(out, h.Name())
	}
	return out
}

func (c *HookChain) RunBeforeChat(ctx context.Context, body []byte) ([]byte, error) {
	if c == nil {
		return body, nil
	}
	c.mu.RLock()
	hooks := append([]ChatHook(nil), c.hooks...)
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
