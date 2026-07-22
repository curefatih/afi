package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestAuthenticateKey(t *testing.T) {
	raw := "sk-good"
	snap := snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey(raw), ProjectID: "p1", OrganizationID: "o1",
		}},
	})
	if _, err := AuthenticateKey(snap, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := AuthenticateKey(snap, "sk-bad"); err != kernel.ErrUnauthorized {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if _, err := AuthenticateKey(nil, raw); err != kernel.ErrNotFound {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestChatCompletionsUnauthorized(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))

	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
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
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

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
	if got.PromptTokens != 3 || got.CompletionTokens != 1 || got.Status != "ok" {
		t.Fatalf("usage event: %+v", got)
	}
}

func TestChatCompletionsStreamParsesUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		opts, _ := req["stream_options"].(map[string]any)
		if opts == nil || opts["include_usage"] != true {
			http.Error(w, "missing stream_options.include_usage", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":9,\"completion_tokens\":2}}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
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
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.PromptTokens != 9 || got.CompletionTokens != 2 {
		t.Fatalf("usage event: %+v", got)
	}
}

type stubCredOpener struct {
	secret string
}

func (s stubCredOpener) Open(context.Context, snapshot.Credential) (string, error) {
	return s.secret, nil
}

func TestChatCompletionsPolicyCredentialMarksBYOK(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-partner" {
			http.Error(w, "want partner key, got "+got, http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"pong"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer upstream.Close()

	t.Setenv("OPENAI_API_KEY", "sk-platform")

	credCfg, _ := json.Marshal(map[string]string{
		"credential_name_expr": `request.headers["x-tenant-id"]`,
	})
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
		Credentials: []snapshot.Credential{{
			ID: "cred_partner", OrganizationID: "o1", Name: "acme",
			ProviderType: "openai", StorageKind: snapshot.CredentialStorageEnv,
			SecretRef: "PARTNER_KEY", Status: snapshot.CredentialStatusActive,
		}},
		Policies: []snapshot.Policy{{
			ID: "pol1", OrganizationID: "o1", Name: "byok-header",
			Expression:   `("x-tenant-id" in request.headers) && request.headers["x-tenant-id"] != ""`,
			Action:       snapshot.PolicyActionUseCredential,
			ActionConfig: credCfg,
			Enabled:      true, Priority: 100,
		}},
	}))

	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
	p.Policies = ev
	p.Credentials = stubCredOpener{secret: "sk-partner"}
	var got UsageEvent
	p.Usage = func(e UsageEvent) { got = e }

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	req.Header.Set("X-Tenant-Id", "acme")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got.CredentialID != "cred_partner" {
		t.Fatalf("credential_id=%q want cred_partner; event=%+v", got.CredentialID, got)
	}
	if !got.UsedBYOK {
		t.Fatalf("expected used_byok=true; event=%+v", got)
	}
}
