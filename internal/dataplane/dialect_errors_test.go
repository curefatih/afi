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
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
)

func TestAnthropicDialectQuotaDenyShape(t *testing.T) {
	holder := NewHolder()
	holder.Set(testSnapWithRequestQuota(0))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	p.Counters = &memCounters{used: map[string]int64{}}

	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("x-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "error" {
		t.Fatalf("expected anthropic envelope, got %s", rr.Body.String())
	}
	errObj := out["error"].(map[string]any)
	if errObj["type"] != "rate_limit_error" {
		t.Fatalf("type=%v body=%s", errObj["type"], rr.Body.String())
	}
	if errObj["message"] == "" {
		t.Fatal("missing message")
	}
}

func TestAnthropicDialectPolicyDenyShape(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://127.0.0.1:1", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
		Policies: []snapshot.Policy{{
			ID: "pol1", OrganizationID: "o1", Name: "deny-mini",
			Expression: `request.model == "gpt-4o-mini"`,
			Actions:    []snapshot.PolicyAction{{Type: snapshot.PolicyActionDeny}},
			Enabled:    true, Priority: 100,
		}},
	}))
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	p.Policies = ev

	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("x-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "error" {
		t.Fatalf("expected anthropic envelope, got %s", rr.Body.String())
	}
	errObj := out["error"].(map[string]any)
	if errObj["type"] != "permission_error" {
		t.Fatalf("type=%v body=%s", errObj["type"], rr.Body.String())
	}
}

func TestAnthropicDialectRemapsOpenAIUpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"This model does not exist","type":"invalid_request_error","code":"model_not_found"}}`)
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
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "missing-model",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(client), slog.Default())

	body := `{"model":"gpt-4o-mini","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/anthropic/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("x-api-key", "sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	raw := rr.Body.String()
	if bytes.Contains(rr.Body.Bytes(), []byte(`"code":"model_not_found"`)) {
		t.Fatalf("leaked openai error body: %s", raw)
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["type"] != "error" {
		t.Fatalf("expected anthropic envelope: %s", raw)
	}
	errObj := out["error"].(map[string]any)
	if errObj["message"] != "This model does not exist" {
		t.Fatalf("message=%v", errObj["message"])
	}
	if errObj["type"] != "invalid_request_error" {
		t.Fatalf("type=%v", errObj["type"])
	}
}
