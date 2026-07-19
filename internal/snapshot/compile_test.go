package snapshot

import "testing"

func TestCompile(t *testing.T) {
	raw := "sk-test"
	src := Source{
		APIKeys: []APIKey{{
			KeyHash: HashKey(raw), KeyPrefix: "sk-test", ProjectID: "p1", OrganizationID: "o1", Name: "dev",
		}},
		Providers: []Provider{{
			ID: "prov_openai", Type: "openai", BaseURL: "https://api.openai.com/v1",
			APIKeyEnv: "OPENAI_API_KEY", Name: "OpenAI",
		}},
		Routes: []Route{{
			OrganizationID: "o1", Model: "gpt-4o-mini", ProviderID: "prov_openai", TargetModel: "gpt-4o-mini",
		}},
	}

	snap := Compile(src)
	if len(snap.APIKeys) != 1 || len(snap.Providers) != 1 || len(snap.Routes) != 1 {
		t.Fatalf("unexpected sizes: %+v", snap)
	}

	k, ok := snap.LookupKey(raw)
	if !ok || k.ProjectID != "p1" {
		t.Fatalf("lookup key failed: %+v", k)
	}

	r, p, ok := snap.LookupRoute("o1", "gpt-4o-mini")
	if !ok || r.TargetModel != "gpt-4o-mini" || p.Type != "openai" {
		t.Fatalf("lookup route failed: %+v %+v", r, p)
	}

	if _, _, ok := snap.LookupRoute("o1", "missing"); ok {
		t.Fatal("expected missing route")
	}
}
