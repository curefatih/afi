package dataplane

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestListModelsFromRoutes(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "key1", KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
		Providers: []snapshot.Provider{{
			ID: "prov", Type: "openai", BaseURL: "http://example.invalid", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []snapshot.Route{
			{OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov", TargetModel: "gpt-4o-mini"},
			{OrganizationID: "o1", Model: "claude-x", ProviderID: "prov", TargetModel: "claude-x"},
			{OrganizationID: "o2", Model: "other-org", ProviderID: "prov", TargetModel: "x"},
		},
	}))

	p := NewPipeline(holder, NewOpenAIClient(), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-good")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Object != "list" {
		t.Fatalf("object=%s", out.Object)
	}
	ids := map[string]bool{}
	for _, m := range out.Data {
		ids[m.ID] = true
	}
	if !ids["gpt-4o-mini"] || !ids["claude-x"] {
		t.Fatalf("missing models: %+v", out.Data)
	}
	if ids["other-org"] {
		t.Fatal("leaked other org model")
	}
}

func TestListModelsUnauthorized(t *testing.T) {
	holder := NewHolder()
	holder.Set(snapshot.Compile(snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			KeyHash: snapshot.HashKey("sk-good"), ProjectID: "p1", OrganizationID: "o1",
		}},
	}))
	p := NewPipeline(holder, NewOpenAIClient(), slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer sk-bad")
	rr := httptest.NewRecorder()
	p.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rr.Code)
	}
}
