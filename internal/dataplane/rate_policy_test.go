package dataplane

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/curefatih/afi/internal/policy"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/redis/go-redis/v9"
)

func TestRedisMinuteWindowRateLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	fixed := time.Unix(1_700_000_000, 0)
	counters := CompositeCounters{
		Total: &memCounters{used: map[string]int64{}},
		Timed: &RedisCounters{Client: rdb, Now: func() time.Time { return fixed }},
	}

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
		Quotas: []snapshot.Quota{{
			ID: "q1", OrganizationID: "o1", ScopeType: snapshot.ScopeAPIKey, ScopeID: "k1",
			Metric: snapshot.MetricRequests, LimitValue: 1, Window: snapshot.WindowMinute,
		}},
	}))
	p := NewPipeline(holder, NewOpenAIClient(), slog.Default())
	p.Counters = counters
	t.Setenv("OPENAI_API_KEY", "x")

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code == http.StatusTooManyRequests {
		t.Fatalf("first request should pass: %s", rr.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req2.Header.Set("Authorization", "Bearer sk-good")
	rr2 := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request want 429, got %d body=%s", rr2.Code, rr2.Body.String())
	}
}

func TestCELPolicyDenies(t *testing.T) {
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
		Policies: []snapshot.Policy{{
			ID: "pol1", OrganizationID: "o1", Name: "deny-mini",
			Expression: `request.model != "gpt-4o-mini"`, Enabled: true, Priority: 100,
		}},
	}))
	ev, err := policy.NewEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	p := NewPipeline(holder, NewOpenAIClient(), slog.Default())
	p.Policies = ev
	p.Counters = &memCounters{used: map[string]int64{}}

	body := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRedisCountersBucket(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	c := &RedisCounters{Client: rdb, Now: func() time.Time { return time.Unix(100, 0) }}
	ctx := context.Background()
	n, err := c.Incr(ctx, "api_key", "k1", "requests", snapshot.WindowMinute, 2)
	if err != nil || n != 2 {
		t.Fatalf("incr: %v %d", err, n)
	}
	got, err := c.Get(ctx, "api_key", "k1", "requests", snapshot.WindowMinute)
	if err != nil || got != 2 {
		t.Fatalf("get: %v %d", err, got)
	}
}
