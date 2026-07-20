package dataplane

import (
	"bytes"
	"encoding/json"
	"github.com/curefatih/afi/internal/adapters/llm"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestRegistryUnknownType(t *testing.T) {
	reg := NewRegistry()
	if _, ok := reg.Get("nope"); ok {
		t.Fatal("expected missing")
	}
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "nope", BaseURL: "http://example.invalid", APIKeyEnv: "X",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m", ProviderID: "prov", TargetModel: "m",
		}},
	}))
	p := NewPipelineWithRegistry(holder, reg, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(
		`{"model":"m","messages":[{"role":"user","content":"hi"}]}`,
	))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestStreamRejectedViaExplicitCapabilities(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_gem", Type: "gemini", BaseURL: "http://example.invalid", APIKeyEnv: "GEMINI_API_KEY",
			Capabilities: snapshot.ProviderCapabilities{Chat: true, Stream: false},
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "g", ProviderID: "prov_gem", TargetModel: "gemini-2.0-flash",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(
		`{"model":"g","stream":true,"messages":[{"role":"user","content":"hi"}]}`,
	))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOpenAICompatibleViaRegistry(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-compat",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from-compat"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer upstream.Close()

	t.Setenv("OLLAMA_API_KEY", "ollama")
	compat := llm.NewOpenAIClient(nil)
	compat.HTTP = upstream.Client()
	reg := NewRegistry().
		Register(newOpenAIChatProvider("openai", llm.NewOpenAIClient(nil), ProviderCaps{Chat: true, Stream: true})).
		Register(newOpenAIChatProvider("openai_compatible", compat, ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(llm.NewAnthropicClient(nil))).
		Register(newGeminiChatProvider(llm.NewGeminiClient(nil)))

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_ollama", Type: "openai_compatible", BaseURL: upstream.URL,
			APIKeyEnv: "OLLAMA_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "llama3", ProviderID: "prov_ollama", TargetModel: "llama3.2",
		}},
	}))

	p := NewPipelineWithRegistry(holder, reg, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(
		`{"model":"llama3","messages":[{"role":"user","content":"hi"}]}`,
	))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("from-compat")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestListModelsIncludesStreamingCapability(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "oai", Type: "openai", BaseURL: "http://x", APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "gem", Type: "gemini", BaseURL: "http://x", APIKeyEnv: "GEMINI_API_KEY",
				Capabilities: snapshot.ProviderCapabilities{Chat: true, Stream: false}},
		},
		Routes: []snapshot.Route{
			{OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "oai", TargetModel: "gpt-4o-mini"},
			{OrganizationID: "o1", Model: "gemini-flash", ProviderID: "gem", TargetModel: "gemini-2.0-flash"},
		},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	var out struct {
		Data []struct {
			ID                string `json:"id"`
			SupportsStreaming bool   `json:"supports_streaming"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	byID := map[string]bool{}
	for _, m := range out.Data {
		byID[m.ID] = m.SupportsStreaming
	}
	if !byID["gpt-4o-mini"] {
		t.Fatal("openai route should support streaming")
	}
	if byID["gemini-flash"] {
		t.Fatal("explicit no-stream provider should report false")
	}
}
