package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
	sdkhook "github.com/curefatih/afi/sdk/hook"
)

type captureAfterChat struct {
	mu   sync.Mutex
	info []AfterChatInfo
}

func (c *captureAfterChat) Name() string { return "capture_after_chat" }
func (c *captureAfterChat) AfterChat(_ context.Context, info AfterChatInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.info = append(c.info, info)
	return nil
}

func (c *captureAfterChat) last() AfterChatInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.info) == 0 {
		return AfterChatInfo{}
	}
	return c.info[len(c.info)-1]
}

func TestAfterChatRunsForAnthropicDialect(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "pong"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "ok")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	capture := &captureAfterChat{}
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	p.Hooks = NewHookChain().RegisterAfter(capture)

	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("x-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	got := capture.last()
	if got.Status != "ok" {
		t.Fatalf("after_chat status=%q infos=%+v", got.Status, capture.info)
	}
	if got.Dialect != "anthropic" || got.Modality != ModalityMessages {
		t.Fatalf("dialect/modality=%q/%q", got.Dialect, got.Modality)
	}
	if got.Model != "gpt-4o-mini" {
		t.Fatalf("model=%q", got.Model)
	}
}

type captureBeforeCall struct {
	mu         sync.Mutex
	dialect    string
	clientBody []byte
	callBody   []byte
}

type replaceSystemHook struct{}

func (replaceSystemHook) Name() string { return "replace_system" }
func (replaceSystemHook) BeforeChat(_ context.Context, req ir.ChatRequest) (ir.ChatRequest, error) {
	req.System = "typed system"
	return req, nil
}

func (c *captureBeforeCall) Name() string { return "capture_before_call" }
func (c *captureBeforeCall) BeforeCall(_ context.Context, call *CallContext) (CallDecision, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if call.Metadata != nil {
		if s, ok := call.Metadata["dialect"].(string); ok {
			c.dialect = s
		}
		if b, ok := call.Metadata["client_body"].([]byte); ok {
			c.clientBody = append([]byte(nil), b...)
		}
	}
	c.callBody = append([]byte(nil), call.Body...)
	return sdkhook.Allow(), nil
}

type captureAfterCall struct {
	mu   sync.Mutex
	body []byte
}

func (c *captureAfterCall) Name() string { return "capture_after_call" }
func (c *captureAfterCall) AfterCall(_ context.Context, call *CallContext, _ AfterCallInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.body = append([]byte(nil), call.Body...)
	return nil
}

func TestBeforeCallSeesClientBodyAndTypedChatHookReachesUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		// Bridge must be OpenAI-shaped for the OpenAI upstream.
		if !bytes.Contains(raw, []byte(`"messages"`)) {
			http.Error(w, "expected openai body", http.StatusBadRequest)
			return
		}
		if !bytes.Contains(raw, []byte(`"content":"typed system"`)) {
			http.Error(w, "expected typed hook mutation", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "ok"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "ok")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	capture := &captureBeforeCall{}
	afterCall := &captureAfterCall{}
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	p.Hooks = NewHookChain().
		RegisterBeforeCall(capture).
		RegisterAfterCall(afterCall).
		Register(replaceSystemHook{})

	anthBody := `{"model":"gpt-4o-mini","max_tokens":64,"system":"be brief","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(anthBody))
	req.Header.Set("x-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if capture.dialect != "anthropic" {
		t.Fatalf("dialect=%q", capture.dialect)
	}
	if !bytes.Contains(capture.clientBody, []byte(`"system":"be brief"`)) {
		t.Fatalf("client_body missing anthropic system: %s", capture.clientBody)
	}
	// BeforeCall runs before chat decoding hooks, so it still observes the raw client body.
	if !bytes.Equal(capture.callBody, capture.clientBody) {
		t.Fatalf("before_call body=%s want client body %s", capture.callBody, capture.clientBody)
	}
	// AfterCall observes the OpenAI-shaped bridge built from the hook-mutated IR.
	if bytes.Contains(afterCall.body, []byte(`"system":"be brief"`)) {
		t.Fatalf("after_call body should be openai-shaped, got %s", afterCall.body)
	}
	if !bytes.Contains(afterCall.body, []byte(`"role":"system"`)) {
		t.Fatalf("after_call body should lift system into messages: %s", afterCall.body)
	}
	if !bytes.Contains(afterCall.body, []byte(`"content":"typed system"`)) {
		t.Fatalf("after_call body missing typed hook mutation: %s", afterCall.body)
	}
}
