package dataplane

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/curefatih/afi/internal/snapshot"
)

func TestAuthenticateRejectsForeignOrgAllowlist(t *testing.T) {
	t.Parallel()
	raw := "sk-foreign"
	src := snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "org_foreign",
			ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
		Routes: []snapshot.Route{{
			OrganizationID: "org_foreign", Model: "m", ProviderID: "prov", TargetModel: "m",
		}},
		Providers: []snapshot.Provider{{ID: "prov", Type: "openai", Name: "o"}},
	}
	snap := snapshot.CompileRegion(src, []string{"org_allowed"})
	// Key is present (over-broad blob) but allowlist excludes the org.
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	_, err := authenticateGatewayRequest(req.Context(), snap, nil, req, nil)
	if err == nil {
		t.Fatal("expected forbid")
	}
}

func TestAuthenticateAllowsListedOrg(t *testing.T) {
	t.Parallel()
	raw := "sk-ok"
	src := snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "org_a",
			ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
	}
	snap := snapshot.CompileRegion(src, []string{"org_a"})
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	p, err := authenticateGatewayRequest(req.Context(), snap, nil, req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.OrganizationID != "org_a" {
		t.Fatalf("%+v", p)
	}
}

func TestAuthenticateGlobalSnapshotUnrestricted(t *testing.T) {
	t.Parallel()
	raw := "sk-g"
	src := snapshot.Source{
		APIKeys: []snapshot.APIKey{{
			ID: "k1", KeyHash: snapshot.HashKey(raw), OrganizationID: "org_x",
			ProjectID: "p1", Name: "t", Kind: snapshot.KeyKindServiceAccount,
		}},
	}
	snap := snapshot.Compile(src) // nil allowlist
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	if _, err := authenticateGatewayRequest(req.Context(), snap, nil, req, nil); err != nil {
		t.Fatal(err)
	}
}
