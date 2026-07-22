package hook_test

import (
	"testing"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

func TestCallContextHeaderHelpers(t *testing.T) {
	call := &sdkhook.CallContext{}
	call.SetRequestHeader("x-trace", "abc")
	call.SetResponseHeader("x-afi-hook", "1")
	if call.RequestHeaders["X-Trace"] != "abc" {
		t.Fatalf("request: %#v", call.RequestHeaders)
	}
	if call.ResponseHeaders["X-Afi-Hook"] != "1" {
		t.Fatalf("response: %#v", call.ResponseHeaders)
	}
	call.DeleteRequestHeader("x-trace")
	if len(call.RequestHeaders) != 0 {
		t.Fatalf("expected empty request headers, got %#v", call.RequestHeaders)
	}
}
