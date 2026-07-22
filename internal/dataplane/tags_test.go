package dataplane

import (
	"net/http"
	"testing"
)

func TestParseAFITags(t *testing.T) {
	got := ParseAFITags(" env:prod , team:platform,env:staging ")
	if got["env"] != "staging" || got["team"] != "platform" {
		t.Fatalf("got %#v", got)
	}
}

func TestHeadersForPolicy(t *testing.T) {
	r := &http.Request{Header: http.Header{}}
	r.Header.Set("X-Tenant-Id", "acme")
	r.Header.Set("Authorization", "Bearer secret")
	r.Header.Set("Cookie", "a=1")
	got := HeadersForPolicy(r.Header)
	if got["x-tenant-id"] != "acme" {
		t.Fatalf("got %#v", got)
	}
	if _, ok := got["authorization"]; ok {
		t.Fatal("authorization should be omitted")
	}
	if _, ok := got["cookie"]; ok {
		t.Fatal("cookie should be omitted")
	}
}
