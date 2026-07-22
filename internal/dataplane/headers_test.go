package dataplane

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplyResponseHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	call := &CallContext{ResponseHeaders: map[string]string{"x-afi-hook": "1"}}
	applyResponseHeaders(rec, call)
	if got := rec.Header().Get("X-Afi-Hook"); got != "1" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeRequestHeaders(t *testing.T) {
	dst := make(http.Header)
	call := &CallContext{RequestHeaders: map[string]string{"x-custom": "yes"}}
	mergeRequestHeaders(dst, call)
	if got := dst.Get("X-Custom"); got != "yes" {
		t.Fatalf("got %q", got)
	}
}
