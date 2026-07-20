package dataplane

import (
	"context"
	"net/http"
	"testing"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func TestParseAFITags(t *testing.T) {
	t.Parallel()
	got := ParseAFITags("end-user-id:fatihcure, something-different:123")
	if got["end-user-id"] != "fatihcure" || got["something-different"] != "123" {
		t.Fatalf("%v", got)
	}
	got = ParseAFITags("a:1,a:2")
	if got["a"] != "2" {
		t.Fatalf("last wins: %v", got)
	}
	if len(ParseAFITags("")) != 0 {
		t.Fatal("empty")
	}
	if len(ParseAFITags("nocolon")) != 0 {
		t.Fatal("pair without colon should be skipped")
	}
	if _, ok := ParseAFITags(":noval")[""]; ok {
		t.Fatal("empty key should be skipped")
	}
}

type denyHook struct{}

func (denyHook) Name() string { return "deny" }
func (denyHook) BeforeCall(context.Context, *CallContext) (CallDecision, error) {
	return sdkhook.Deny(http.StatusForbidden, "blocked", "nope"), nil
}

type mutateHook struct{}

func (mutateHook) Name() string { return "mutate" }
func (mutateHook) BeforeCall(_ context.Context, call *CallContext) (CallDecision, error) {
	call.Tags["enriched"] = "1"
	return sdkhook.Allow(), nil
}

func TestBeforeCallDenyShortCircuits(t *testing.T) {
	t.Parallel()
	c := NewHookChain().RegisterBeforeCall(mutateHook{}).RegisterBeforeCall(denyHook{})
	call := &CallContext{Tags: map[string]string{}, Metadata: map[string]any{}}
	d, err := c.RunBeforeCall(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if d.Allow || call.Tags["enriched"] != "1" {
		t.Fatalf("mutate should run before deny: %+v tags=%v", d, call.Tags)
	}
	// Second deny-only chain without mutate
	c2 := NewHookChain().RegisterBeforeCall(denyHook{}).RegisterBeforeCall(mutateHook{})
	call2 := &CallContext{Tags: map[string]string{}, Metadata: map[string]any{}}
	d2, err := c2.RunBeforeCall(context.Background(), call2)
	if err != nil {
		t.Fatal(err)
	}
	if d2.Allow {
		t.Fatal("expected deny")
	}
	if call2.Tags["enriched"] == "1" {
		t.Fatal("mutate must not run after deny")
	}
}

func TestHookInfoBeforeCall(t *testing.T) {
	t.Parallel()
	c := NewHookChain().RegisterBeforeCall(denyHook{}).Register(prefixHook{})
	infos := c.Infos()
	if len(infos) != 2 {
		t.Fatalf("%+v", infos)
	}
	byName := map[string]HookInfo{}
	for _, i := range infos {
		byName[i.Name] = i
	}
	if !byName["deny"].BeforeCall || byName["deny"].BeforeChat {
		t.Fatalf("%+v", byName["deny"])
	}
	if !byName["prefix"].BeforeChat || byName["prefix"].BeforeCall {
		t.Fatalf("%+v", byName["prefix"])
	}
}
