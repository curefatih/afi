package dataplane

import (
	"github.com/curefatih/afi/internal/adapters/llm"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(raw, []byte("native pong")) {
		t.Fatalf("body=%s", raw)
	}
	if bytes.Contains(raw, []byte("chat.completion")) {
		t.Fatal("expected anthropic-shaped response, not openai")
	}
}

func TestNativeMessagesRejectsNonAnthropic(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
