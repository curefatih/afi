package snapshot

import "testing"

func TestLookupRouteOrgScoped(t *testing.T) {
	t.Parallel()
	src := Source{
		Providers: []Provider{{
			ID: "p1", Type: "openai", BaseURL: "https://api.openai.com/v1", APIKeyEnv: "OPENAI_API_KEY",
		}},
		Routes: []Route{
			{OrganizationID: "org_a", Model: "gpt-4o-mini", ProviderID: "p1", TargetModel: "gpt-4o-mini"},
			{OrganizationID: "org_b", Model: "gpt-4o-mini", ProviderID: "p1", TargetModel: "gpt-4o"},
		},
	}
	snap := Compile(src)

	r, p, ok := snap.LookupRoute("org_a", "gpt-4o-mini")
	if !ok || r.TargetModel != "gpt-4o-mini" || p.ID != "p1" {
		t.Fatalf("org_a lookup failed: %+v %+v ok=%v", r, p, ok)
	}
	r, _, ok = snap.LookupRoute("org_b", "gpt-4o-mini")
	if !ok || r.TargetModel != "gpt-4o" {
		t.Fatalf("org_b lookup failed: %+v", r)
	}
	if _, _, ok := snap.LookupRoute("org_c", "gpt-4o-mini"); ok {
		t.Fatal("expected miss for unknown org")
	}
}

func TestLookupKeyByHash(t *testing.T) {
	t.Parallel()
	raw := "sk-project-local-dev-token-12345"
	h := HashKey(raw)
	snap := Compile(Source{
		APIKeys: []APIKey{{
			KeyHash: h, KeyPrefix: "sk-proj", ProjectID: "p1", OrganizationID: "o1", Name: "dev",
		}},
	})
	k, ok := snap.LookupKey(raw)
	if !ok || k.ProjectID != "p1" {
		t.Fatalf("lookup by raw key failed: %+v ok=%v", k, ok)
	}
	if _, ok := snap.LookupKey("sk-wrong"); ok {
		t.Fatal("expected miss")
	}
	// Snapshot payload must not contain the raw secret.
	for key := range snap.APIKeys {
		if key == raw {
			t.Fatal("raw key must not be map key")
		}
	}
}
