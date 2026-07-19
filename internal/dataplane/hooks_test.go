package dataplane

import (
	"context"
	"testing"
)

type prefixHook struct{}

func (prefixHook) Name() string { return "prefix" }
func (prefixHook) BeforeChat(_ context.Context, body []byte) ([]byte, error) {
	return append([]byte("X"), body...), nil
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
