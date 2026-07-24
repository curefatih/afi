package dataplane

import (
	"context"
	"testing"

	"github.com/curefatih/afi/internal/dataplane/ir"
)

type prefixHook struct{}

func (prefixHook) Name() string { return "prefix" }
func (prefixHook) BeforeChat(_ context.Context, body []byte) ([]byte, error) {
	return append([]byte("X"), body...), nil
}

type typedSystemHook struct{}

func (typedSystemHook) Name() string { return "typed-system" }
func (typedSystemHook) BeforeChatIR(_ context.Context, req ir.ChatRequest) (ir.ChatRequest, error) {
	req.System = "typed"
	return req, nil
}

func TestHookChainOrder(t *testing.T) {
	t.Parallel()
	c := NewHookChain().Register(prefixHook{})
	out, err := c.RunBeforeChat(context.Background(), []byte("ab"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "Xab" {
		t.Fatalf("%q", out)
	}
	if got := c.Names(); len(got) != 1 || got[0] != "prefix" {
		t.Fatalf("%v", got)
	}
}

func TestHookChainTypedChat(t *testing.T) {
	t.Parallel()
	c := NewHookChain().RegisterHook(typedSystemHook{})
	out, err := c.RunBeforeChatIR(context.Background(), ir.ChatRequest{
		Messages: []ir.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.System != "typed" {
		t.Fatalf("request=%+v", out)
	}
	infos := c.Infos()
	if len(infos) != 1 || !infos[0].BeforeChatIR || infos[0].BeforeChat {
		t.Fatalf("infos=%+v", infos)
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
