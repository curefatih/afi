package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestAuthenticateKey(t *testing.T) {
	snap := snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{Key: "sk-good", ProjectID: "p1"}},
	})
	if _, err := AuthenticateKey(snap, "sk-good"); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateKey(snap, "sk-bad"); err != kernel.ErrUnauthorized {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if _, err := AuthenticateKey(nil, "sk-good"); err != kernel.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestChatCompletionsUnauthorized(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{Key: "sk-good", ProjectID: "p1"}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini"}},
	}))

	p := NewPipeline(holder, NewOpenAIClient(), slog.Default())
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-bad")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestChatCompletionsNonStreamMockUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-test",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "pong"}},
			},
		})
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{Key: "sk-good", ProjectID: "p1"}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: upstream.URL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini"}},
	}))

	client := NewOpenAIClient()
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, client, slog.Default())

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	b, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(b, []byte("pong")) {
		t.Fatalf("unexpected body: %s", b)
	}
}
