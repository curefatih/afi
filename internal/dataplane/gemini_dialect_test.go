package dataplane

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestGeminiDialectRoutesToOpenAIUpstream(t *testing.T) {
	var upstreamBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		upstreamBody, _ = io.ReadAll(r.Body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "chatcmpl-1",
			"model": "gpt-4o-mini",
			"choices": []any{map[string]any{
				"index": 0,
				"message": map[string]any{
					"role": "assistant", "content": "hello from OpenAI",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{
				"prompt_tokens": 2, "completion_tokens": 3,
			},
		})
	}))
	t.Cleanup(upstream.Close)

	p := geminiDialectTestPipeline(t, upstream)
	body := `{
		"systemInstruction":{"parts":[{"text":"be brief"}]},
		"contents":[{"role":"user","parts":[{"text":"hello"}]}],
		"generationConfig":{"maxOutputTokens":64}
	}`
	req := httptest.NewRequest(
		http.MethodPost,
		"/gemini/v1beta/models/my-route:generateContent",
		bytes.NewBufferString(body),
	)
	req.Header.Set("x-goog-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !bytes.Contains(upstreamBody, []byte(`"model":"gpt-4o-mini"`)) ||
		!bytes.Contains(upstreamBody, []byte(`"role":"system"`)) {
		t.Fatalf("upstream body=%s", upstreamBody)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	candidates := out["candidates"].([]any)
	content := candidates[0].(map[string]any)["content"].(map[string]any)
	parts := content["parts"].([]any)
	if parts[0].(map[string]any)["text"] != "hello from OpenAI" {
		t.Fatalf("response=%v", out)
	}
}

func TestGeminiDialectStreamRoutesToOpenAIUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: "+
			`{"id":"chatcmpl-1","model":"gpt-4o-mini","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}`+
			"\n\n")
		_, _ = io.WriteString(w, "data: "+
			`{"id":"chatcmpl-1","model":"gpt-4o-mini","choices":[{"delta":{"content":"streamed"},"finish_reason":null}]}`+
			"\n\n")
		_, _ = io.WriteString(w, "data: "+
			`{"id":"chatcmpl-1","model":"gpt-4o-mini","choices":[{"delta":{},"finish_reason":"stop"}]}`+
			"\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(upstream.Close)

	p := geminiDialectTestPipeline(t, upstream)
	req := httptest.NewRequest(
		http.MethodPost,
		"/gemini/v1beta/models/my-route:streamGenerateContent?alt=sse&key=sk-good",
		bytes.NewBufferString(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	)
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"text":"streamed"`) ||
		!strings.Contains(rr.Body.String(), `"finishReason":"STOP"`) {
		t.Fatalf("stream=%s", rr.Body.String())
	}
}

func TestGeminiDialectErrorShape(t *testing.T) {
	p := geminiDialectTestPipeline(t, nil)
	req := httptest.NewRequest(
		http.MethodPost,
		"/gemini/v1beta/models/missing-route:generateContent",
		bytes.NewBufferString(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	)
	req.Header.Set("x-goog-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	errObj := out["error"].(map[string]any)
	if errObj["status"] != "INVALID_ARGUMENT" ||
		!strings.Contains(errObj["message"].(string), "no route") {
		t.Fatalf("error=%v", errObj)
	}
}

func TestGeminiDialectRouteAliasWithColon(t *testing.T) {
	var gotModel string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-1", "model": "gpt-4o-mini",
			"choices": []any{map[string]any{
				"index": 0,
				"message": map[string]any{
					"role": "assistant", "content": "ok",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	t.Cleanup(upstream.Close)

	p := geminiDialectTestPipeline(t, upstream)
	req := httptest.NewRequest(
		http.MethodPost,
		"/gemini/v1beta/models/team:prod:generateContent",
		bytes.NewBufferString(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`),
	)
	req.Header.Set("x-goog-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotModel != "gpt-4o-mini" {
		t.Fatalf("upstream model=%q", gotModel)
	}
}

func TestParseGeminiOperation(t *testing.T) {
	cases := []struct {
		op     string
		model  string
		stream bool
		ok     bool
	}{
		{op: "my-route:generateContent", model: "my-route", stream: false, ok: true},
		{op: "my-route:streamGenerateContent", model: "my-route", stream: true, ok: true},
		{op: "team:prod:generateContent", model: "team:prod", stream: false, ok: true},
		{op: "team:prod:streamGenerateContent", model: "team:prod", stream: true, ok: true},
		{op: "my-route:countTokens", ok: false},
		{op: "generateContent", ok: false},
		{op: "", ok: false},
	}
	for _, tc := range cases {
		model, stream, ok := parseGeminiOperation(tc.op)
		if ok != tc.ok || model != tc.model || stream != tc.stream {
			t.Fatalf("%q => (%q,%v,%v) want (%q,%v,%v)",
				tc.op, model, stream, ok, tc.model, tc.stream, tc.ok)
		}
	}
}

func geminiDialectTestPipeline(t *testing.T, upstream *httptest.Server) *Pipeline {
	t.Helper()
	t.Setenv("OPENAI_API_KEY", "upstream-key")
	baseURL := "http://127.0.0.1:1/v1"
	client := llm.NewOpenAIClient(nil)
	if upstream != nil {
		baseURL = upstream.URL + "/v1"
		client.HTTP = upstream.Client()
	}
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: baseURL, APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{
			{
				OrganizationID: "o1", Model: "my-route", ProviderID: "prov", TargetModel: "gpt-4o-mini",
			},
			{
				OrganizationID: "o1", Model: "team:prod", ProviderID: "prov", TargetModel: "gpt-4o-mini",
			},
		},
	}))
	return NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())
}
