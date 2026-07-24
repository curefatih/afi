package dataplane

import (
	"context"
	"errors"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

type prefixHook struct{ prefix string }

func (prefixHook) Name() string { return "prefix" }
func (h prefixHook) BeforeChat(_ context.Context, req ir.ChatRequest) (ir.ChatRequest, error) {
	for i := range req.Messages {
		req.Messages[i].Content = h.prefix + req.Messages[i].Content
	}
	return req, nil
}

type systemHook struct{}

func (systemHook) Name() string { return "typed-system" }
func (systemHook) BeforeChat(_ context.Context, req ir.ChatRequest) (ir.ChatRequest, error) {
	req.System = "typed"
	return req, nil
}

type failingChatHook struct{ err error }

func (failingChatHook) Name() string { return "boom" }
func (h failingChatHook) BeforeChat(_ context.Context, req ir.ChatRequest) (ir.ChatRequest, error) {
	return req, h.err
}

func TestHookChainOrder(t *testing.T) {
	t.Parallel()
	c := NewHookChain().Register(prefixHook{prefix: "X"}).Register(prefixHook{prefix: "Y"})
	out, err := c.RunBeforeChat(context.Background(), ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "ab"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Messages[0].Content != "YXab" {
		t.Fatalf("content=%q", out.Messages[0].Content)
	}
	if got := c.Names(); len(got) != 1 || got[0] != "prefix" {
		t.Fatalf("%v", got)
	}
}

func TestHookChainTypedChat(t *testing.T) {
	t.Parallel()
	c := NewHookChain().RegisterHook(systemHook{})
	out, err := c.RunBeforeChat(context.Background(), ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.System != "typed" {
		t.Fatalf("request=%+v", out)
	}
	infos := c.Infos()
	if len(infos) != 1 || !infos[0].BeforeChat || infos[0].AfterChat {
		t.Fatalf("infos=%+v", infos)
	}
}

func TestHookChainBeforeChatStopsOnError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("hook failed")
	c := NewHookChain().
		Register(failingChatHook{err: sentinel}).
		Register(systemHook{})
	out, err := c.RunBeforeChat(context.Background(), ir.ChatRequest{Model: "m"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err=%v", err)
	}
	if out.System != "" {
		t.Fatalf("later hook ran: %+v", out)
	}
}

func TestHookChainBeforeChatNilSafety(t *testing.T) {
	t.Parallel()
	var c *HookChain
	req := ir.ChatRequest{Model: "m"}
	out, err := c.RunBeforeChat(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if out.Model != "m" {
		t.Fatalf("out=%+v", out)
	}
	if got := NewHookChain().Register(nil).Names(); len(got) != 0 {
		t.Fatalf("nil hook registered: %v", got)
	}
}

type afterHook struct{ called *bool }

func (afterHook) Name() string { return "after" }
func (a afterHook) AfterChat(context.Context, AfterChatInfo) error {
	*a.called = true
	return nil
}

func TestHookChainAfter(t *testing.T) {
	t.Parallel()
	called := false
	c := NewHookChain().RegisterHook(afterHook{called: &called})
	c.RunAfterChat(context.Background(), AfterChatInfo{Status: "ok"})
	if !called {
		t.Fatal("expected AfterChat")
	}
	infos := c.Infos()
	if len(infos) != 1 || !infos[0].AfterChat || infos[0].BeforeChat || infos[0].BeforeCall {
		t.Fatalf("%+v", infos)
	}
}
