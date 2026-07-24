package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestNativeMessagesPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["model"] != "claude-sonnet-4-20250514" {
			http.Error(w, "bad model", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "msg_native",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{"type": "text", "text": "native pong"},
			},
			"usage": map[string]int{"input_tokens": 2, "output_tokens": 2},
		})
	}))
	defer upstream.Close()

	t.Setenv("ANTHROPIC_API_KEY", "ak")
	anth := llm.NewAnthropicClient(nil)
	anth.HTTP = upstream.Client()
	reg := NewRegistry().
		Register(newOpenAIChatProvider("openai", llm.NewOpenAIClient(nil), ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(anth)).
		Register(newGeminiChatProvider(llm.NewGeminiClient(nil)))

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_anth", Type: "anthropic", BaseURL: upstream.URL, APIKeyEnv: "ANTHROPIC_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "claude-sonnet", ProviderID: "prov_anth",
			TargetModel: "claude-sonnet-4-20250514",
		}},
	}))

	p := NewPipelineWithRegistry(holder, reg, slog.Default())
	body := `{"model":"claude-sonnet","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	for _, path := range []string{"/v1/messages", "/anthropic/v1/messages"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer sk-good")
		rr := httptest.NewRecorder()
		p.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
		raw, _ := io.ReadAll(rr.Body)
		if !bytes.Contains(raw, []byte("native pong")) {
			t.Fatalf("%s body=%s", path, raw)
		}
		if bytes.Contains(raw, []byte("chat.completion")) {
			t.Fatalf("%s expected anthropic-shaped response, not openai", path)
		}
	}
}

func TestMessagesDialectOpenAIUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from openai"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 2},
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
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw := rr.Body.Bytes()
	if !bytes.Contains(raw, []byte("from openai")) {
		t.Fatalf("body=%s", raw)
	}
	if bytes.Contains(raw, []byte("chat.completion")) {
		t.Fatal("expected anthropic dialect response")
	}
	var msg struct {
		Type    string `json:"type"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Type != "message" || len(msg.Content) == 0 || msg.Content[0].Text != "from openai" {
		t.Fatalf("%+v", msg)
	}
}

func TestMessagesAcceptsXAPIKey(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("x-api-key", "sk-good")
	req.Header.Set("anthropic-version", "2023-06-01")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOpenAIDialectAliasPaths(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "pong"}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	for _, path := range []string{"/v1/chat/completions", "/openai/v1/chat/completions"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer sk-good")
		rr := httptest.NewRecorder()
		p.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
		if !bytes.Contains(rr.Body.Bytes(), []byte("pong")) {
			t.Fatalf("%s body=%s", path, rr.Body.String())
		}
		if !bytes.Contains(rr.Body.Bytes(), []byte("chat.completion")) {
			t.Fatalf("%s expected openai shape: %s", path, rr.Body.String())
		}
	}
}
