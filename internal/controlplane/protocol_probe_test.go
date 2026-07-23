package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeMCPReachable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		var msg map[string]any
		_ = json.NewDecoder(r.Body).Decode(&msg)
		if msg["method"] != "initialize" {
			t.Fatalf("method=%v", msg["method"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	}))
	t.Cleanup(upstream.Close)

	res, err := probeMCP(context.Background(), upstream.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.StatusCode != 200 {
		t.Fatalf("%+v", res)
	}
}

func TestProbeA2AReachable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent-card.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"x","url":"http://x"}`))
	}))
	t.Cleanup(upstream.Close)

	res, err := probeA2A(context.Background(), upstream.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.StatusCode != 200 {
		t.Fatalf("%+v", res)
	}
}

func TestProbeMCPInvalidURL(t *testing.T) {
	_, err := probeMCP(context.Background(), "not-a-url", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
