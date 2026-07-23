package dataplane

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestEmbeddingsPassThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"model":"text-embedding-3-small"`)) {
			t.Fatalf("body=%s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"object":"embedding","index":0,"embedding":[0.1,0.2]}],"model":"text-embedding-3-small","usage":{"prompt_tokens":4,"total_tokens":4}}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()

	raw := "sk-embed-test"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-embed",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
			Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "text-embedding-3-small", ProviderID: "prov_openai", TargetModel: "text-embedding-3-small",
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString(
		`{"model":"text-embedding-3-small","input":"hello"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte(`"embedding"`)) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestEmbeddingsRewritesTargetModel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte(`"model":"text-embedding-3-large"`)) {
			t.Fatalf("expected target rewrite, body=%s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[],"usage":{"prompt_tokens":1,"total_tokens":1}}`))
	}))
	t.Cleanup(upstream.Close)

	t.Setenv("OPENAI_API_KEY", "sk-test")
	client := llm.NewOpenAIClient(nil)
	client.HTTP = upstream.Client()

	raw := "sk-embed-rewrite"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-embed",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: upstream.URL + "/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
			Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "embed-virtual", ProviderID: "prov_openai", TargetModel: "text-embedding-3-large",
		}},
	}))

	p := NewPipelineWithRegistry(holder, RegistryWithOpenAI(client), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString(
		`{"model":"embed-virtual","input":["a","b"]}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestEmbeddingsRejectsChatModel(t *testing.T) {
	raw := "sk-embed-wrong"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), KeyPrefix: "sk-embed",
			OrganizationID: "o1", ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: "https://example.invalid/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
			Capabilities: snapshot.DefaultCapabilities("openai"),
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov_openai", TargetModel: "gpt-4o-mini",
		}},
	}))

	p := NewPipelineWithRegistry(holder, DefaultRegistry(), slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString(
		`{"model":"gpt-4o-mini","input":"hi"}`,
	))
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
