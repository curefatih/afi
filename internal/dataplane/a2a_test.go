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

func TestA2AJSONRPCProxy(t *testing.T) {
	var gotAuth, gotMethod string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		var msg struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &msg)
		gotMethod = msg.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"id":"task-1"}}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("A2A_UPSTREAM_KEY", "sk-a2a")
	raw := "sk-a2a-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-a2a",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		A2AAgents: []snapshot.A2AAgent{{
			ID: "a2a_1", OrganizationID: "o1", Alias: "helper", Name: "Helper",
			UpstreamURL: upstream.URL, APIKeyEnv: "A2A_UPSTREAM_KEY", Enabled: true,
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

	req := httptest.NewRequest(http.MethodPost, "/a2a/helper", bytes.NewBufferString(
		`{"jsonrpc":"2.0","id":1,"method":"message/send","params":{"message":{"role":"user","parts":[{"text":"hi"}]}}}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotAuth != "Bearer sk-a2a" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotMethod != "message/send" {
		t.Fatalf("method=%q", gotMethod)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(usages) != 1 || usages[0].Modality != ModalityA2A {
		t.Fatalf("usage=%+v", usages)
	}
	if usages[0].Metrics["method"] != "message/send" {
		t.Fatalf("metrics=%v", usages[0].Metrics)
	}
}

func TestA2AAgentCardRewrite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/agent-card.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"name":"Helper","url":"https://upstream.example/rpc","skills":[{"id":"chat"}]}`))
	}))
	t.Cleanup(upstream.Close)

	raw := "sk-a2a-card"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-a2a",
			OrganizationID: "o1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		A2AAgents: []snapshot.A2AAgent{{
			ID: "a2a_1", OrganizationID: "o1", Alias: "helper", Name: "Helper",
			UpstreamURL: upstream.URL, CardURL: upstream.URL + "/.well-known/agent-card.json", Enabled: true,
		}},
	}))
	p := NewPipeline(holder, nil, slog.Default())
	p.HTTP = upstream.Client()

	req := httptest.NewRequest(http.MethodGet, "/a2a/helper/.well-known/agent-card.json", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Host = "gateway.example"
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var card map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &card); err != nil {
		t.Fatal(err)
	}
	if card["url"] != "http://gateway.example/a2a/helper" {
		t.Fatalf("url=%v", card["url"])
	}
	if card["name"] != "Helper" {
		t.Fatalf("name=%v", card["name"])
	}
}

func TestA2AAgentCardFromCache(t *testing.T) {
	raw := "sk-a2a-cache"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-a2a",
			OrganizationID: "o1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		A2AAgents: []snapshot.A2AAgent{{
			ID: "a2a_1", OrganizationID: "o1", Alias: "helper", Name: "Helper",
			UpstreamURL: "https://upstream.invalid/rpc",
			CardCache:   json.RawMessage(`{"name":"Cached","url":"https://upstream.invalid/rpc"}`),
			Enabled:     true,
		}},
	}))
	p := NewPipeline(holder, nil, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/a2a/helper/.well-known/agent-card.json", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	req.Host = "gw.test"
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"url":"http://gw.test/a2a/helper"`)) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestA2ARequiresAuth(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{}))
	p := NewPipeline(holder, nil, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/a2a/helper", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
