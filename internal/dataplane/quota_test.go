package dataplane

import (
	"bytes"
	"context"
	"github.com/curefatih/afi/internal/adapters/llm"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

type memCounters struct {
	used map[string]int64
}

func (m *memCounters) Get(_ context.Context, scopeType, scopeID, metric, window string) (int64, error) {
	return m.used[scopeType+"|"+scopeID+"|"+metric+"|"+window], nil
}

func (m *memCounters) Incr(_ context.Context, scopeType, scopeID, metric, window string, delta int64) (int64, error) {
	k := scopeType + "|" + scopeID + "|" + metric + "|" + window
	m.used[k] += delta
	return m.used[k], nil
}

func testSnapWithRequestQuota(limit int64) *snapshot.Snapshot {
	raw := "sk-good"
	return snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://127.0.0.1:1", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
		Quotas: []snapshot.Quota{{
			ID: "q1", OrganizationID: "o1", ScopeType: snapshot.ScopeAPIKey, ScopeID: "k1",
			Metric: snapshot.MetricRequests, LimitValue: limit, Window: snapshot.WindowTotal,
		}},
	})
}

func TestQuotaExceededReturns429(t *testing.T) {
	holder := NewHolder()
	holder.Set(testSnapWithRequestQuota(0))
	counters := &memCounters{used: map[string]int64{}}
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	p.Counters = counters

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestQuotaAllowsWhenUnderLimit(t *testing.T) {
	holder := NewHolder()
	holder.Set(testSnapWithRequestQuota(10))
	counters := &memCounters{used: map[string]int64{}}
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	p.Counters = counters
	// Upstream will fail (bad URL) but after quota check — expect not 429.
	t.Setenv("OPENAI_API_KEY", "x")
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code == http.StatusTooManyRequests {
		t.Fatalf("unexpected 429: %s", rr.Body.String())
	}
	if counters.used["api_key|k1|requests|total"] != 1 {
		t.Fatalf("expected request counter incr, got %+v", counters.used)
	}
}

func TestNoQuotaAllows(t *testing.T) {
	raw := "sk-good"
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://127.0.0.1:1", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini",
		}},
	}))
	p := NewPipeline(holder, RegistryWithOpenAI(llm.NewOpenAIClient(nil)), slog.Default())
	p.Counters = &memCounters{used: map[string]int64{}}
	t.Setenv("OPENAI_API_KEY", "x")
	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code == http.StatusTooManyRequests {
		t.Fatal("unexpected 429 with no quotas")
	}
}
