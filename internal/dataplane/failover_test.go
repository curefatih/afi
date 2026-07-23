package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestFailoverPrimary500UsesFallback(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-fb",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from-fallback"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer fallback.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "m1"}},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw, _ := io.ReadAll(rr.Body)
	if !bytes.Contains(raw, []byte("from-fallback")) {
		t.Fatalf("body=%s", raw)
	}
	if primaryHits.Load() != 1 {
		t.Fatalf("primary hits=%d", primaryHits.Load())
	}
}

func TestRetrySameTargetBeforeFallback(t *testing.T) {
	var primaryHits, fallbackHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := primaryHits.Add(1)
		if n < 3 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-ok",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "after-retry"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
	}))
	defer fallback.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "m1"}},
			Retry: &snapshot.RetryConfig{
				MaxAttempts: 3,
				Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1ms"},
			},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("after-retry")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if primaryHits.Load() != 3 {
		t.Fatalf("primary hits=%d want 3", primaryHits.Load())
	}
	if fallbackHits.Load() != 0 {
		t.Fatalf("fallback should not be used, hits=%d", fallbackHits.Load())
	}
}

func TestRetryExhaustedThenFailover(t *testing.T) {
	var primaryHits, fallbackHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-fb",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from-fallback"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer fallback.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "m1"}},
			Retry: &snapshot.RetryConfig{
				MaxAttempts: 2,
				Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1ms"},
			},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("from-fallback")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if primaryHits.Load() != 2 {
		t.Fatalf("primary hits=%d want 2", primaryHits.Load())
	}
	if fallbackHits.Load() != 1 {
		t.Fatalf("fallback hits=%d want 1", fallbackHits.Load())
	}
}

func TestRetryUsesOrgDefaultWhenRouteUnset(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := primaryHits.Add(1)
		if n < 2 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-ok",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "org-default-retry"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer primary.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
		}},
		DefaultRetries: map[string]*snapshot.RetryConfig{
			"o1": {
				MaxAttempts: 2,
				Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1ms"},
			},
		},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("org-default-retry")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if primaryHits.Load() != 2 {
		t.Fatalf("primary hits=%d want 2", primaryHits.Load())
	}
}

func TestRetrySkippedOn400(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer primary.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Retry: &snapshot.RetryConfig{
				MaxAttempts: 3,
				Backoff:     snapshot.BackoffConfig{Strategy: snapshot.BackoffFixed, BaseDelay: "1ms"},
			},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if primaryHits.Load() != 1 {
		t.Fatalf("primary hits=%d want 1", primaryHits.Load())
	}
}

func TestFailoverPrimary400NoFallback(t *testing.T) {
	var fallbackHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad request"}}`))
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
	}))
	defer fallback.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "m1"}},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if fallbackHits.Load() != 0 {
		t.Fatalf("fallback should not be called, hits=%d", fallbackHits.Load())
	}
}

func TestFailoverAllFailReturnsLastError(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "p", http.StatusBadGateway)
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "f", http.StatusServiceUnavailable)
	}))
	defer fallback.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: primary.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: fallback.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "m1", ProviderID: "prov_a", TargetModel: "m1",
			Fallbacks: []snapshot.RouteTarget{{ProviderID: "prov_b", TargetModel: "m1"}},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = primary.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestChatCompletionsAnthropicProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "role": "assistant",
			"content":     []map[string]string{{"type": "text", "text": "anth-pong"}},
			"stop_reason": "end_turn",
			"usage":       map[string]int{"input_tokens": 2, "output_tokens": 2},
		})
	}))
	defer upstream.Close()

	t.Setenv("ANTHROPIC_API_KEY", "ak")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov_anth", Type: "anthropic", BaseURL: upstream.URL, APIKeyEnv: "ANTHROPIC_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "claude-x", ProviderID: "prov_anth", TargetModel: "claude-x",
		}},
	}))

	anth := llm.NewAnthropicClient(nil)
	anth.HTTP = upstream.Client()
	reg := NewRegistry().
		Register(newOpenAIChatProvider("openai", llm.NewOpenAIClient(nil), ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(anth)).
		Register(newGeminiChatProvider(llm.NewGeminiClient(nil)))
	p := NewPipelineWithRegistry(holder, reg, slog.Default())

	body := `{"model":"claude-x","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("anth-pong")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestWeightedRoutingFirstPickThenFailoverOrder(t *testing.T) {
	var aHits, bHits atomic.Int32
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		aHits.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-a",
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "from-a"}},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer a.Close()

	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bHits.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer b.Close()

	t.Setenv("OPENAI_API_KEY", "k")

	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{
			{ID: "prov_a", Type: "openai", BaseURL: a.URL, APIKeyEnv: "OPENAI_API_KEY"},
			{ID: "prov_b", Type: "openai", BaseURL: b.URL, APIKeyEnv: "OPENAI_API_KEY"},
		},
		Routes: []snapshot.Route{{
			OrganizationID:  "o1",
			Model:           "m1",
			ProviderID:      "prov_a",
			TargetModel:     "m1",
			RoutingStrategy: "weighted",
			Weight:          1,
			Fallbacks: []snapshot.RouteTarget{
				{ProviderID: "prov_b", TargetModel: "m1", Weight: 1},
			},
		}},
	}))

	client := llm.NewOpenAIClient(nil)
	client.HTTP = a.Client()
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	// Find a seed that picks prov_b first (Intn(2)==1), then failover to prov_a.
	var seed int64 = -1
	for s := int64(0); s < 500; s++ {
		rng := rand.New(rand.NewSource(s))
		if rng.Intn(2) == 1 {
			seed = s
			break
		}
	}
	if seed < 0 {
		t.Fatal("no seed")
	}
	p.RouteRand = rand.New(rand.NewSource(seed))

	body := `{"model":"m1","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("from-a")) {
		t.Fatalf("body=%s", rr.Body.String())
	}
	if bHits.Load() != 1 || aHits.Load() != 1 {
		t.Fatalf("hits a=%d b=%d", aHits.Load(), bHits.Load())
	}
}
