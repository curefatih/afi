package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestMCPProxyPassThrough(t *testing.T) {
	var gotAuth, gotSession, gotMethod string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotSession = r.Header.Get("Mcp-Session-Id")
		body, _ := io.ReadAll(r.Body)
		var msg struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &msg)
		gotMethod = msg.Method
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "sess-upstream")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"ok":true}}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("MCP_UPSTREAM_KEY", "sk-upstream")
	raw := "sk-mcp-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-mcp",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		MCPBackends: []snapshot.MCPBackend{{
			ID: "mcp_1", OrganizationID: "o1", Alias: "docs", Name: "Docs",
			BaseURL: upstream.URL, APIKeyEnv: "MCP_UPSTREAM_KEY", Enabled: true,
		}},
	}))

	var mu sync.Mutex
	var usages []UsageEvent
	p := NewPipeline(holder, nil, slog.Default())
	p.HTTP = upstream.Client()
	p.Usage = func(e UsageEvent) {
		mu.Lock()
		usages = append(usages, e)
		mu.Unlock()
	}

	req := httptest.NewRequest(http.MethodPost, "/mcp/docs", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Session-Id", "sess-client")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotAuth != "Bearer sk-upstream" {
		t.Fatalf("upstream auth=%q", gotAuth)
	}
	if gotSession != "sess-client" {
		t.Fatalf("session=%q", gotSession)
	}
	if gotMethod != "tools/list" {
		t.Fatalf("method=%q", gotMethod)
	}
	if rr.Header().Get("Mcp-Session-Id") != "sess-upstream" {
		t.Fatalf("response session=%q", rr.Header().Get("Mcp-Session-Id"))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(usages) != 1 || usages[0].Modality != ModalityMCP {
		t.Fatalf("usage=%+v", usages)
	}
	if usages[0].Metrics["method"] != "tools/list" {
		t.Fatalf("metrics=%v", usages[0].Metrics)
	}
}

func TestMCPRequiresAuth(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{}))
	p := NewPipeline(holder, nil, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/mcp/docs", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}

func TestMCPMethodAllowlist(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	t.Cleanup(upstream.Close)

	raw := "sk-mcp-deny"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-mcp",
			OrganizationID: "o1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		MCPBackends: []snapshot.MCPBackend{{
			ID: "mcp_1", OrganizationID: "o1", Alias: "docs", Name: "Docs",
			BaseURL: upstream.URL, Enabled: true,
			MethodAllowlist: []string{"tools/list"},
		}},
	}))
	p := NewPipeline(holder, nil, slog.Default())
	p.HTTP = upstream.Client()
	req := httptest.NewRequest(http.MethodPost, "/mcp/docs", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"x"}}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMCPUnknownAlias(t *testing.T) {
	raw := "sk-mcp-miss"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-mcp",
			OrganizationID: "o1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
	}))
	p := NewPipeline(holder, nil, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/mcp/missing", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rr.Code)
	}
}
