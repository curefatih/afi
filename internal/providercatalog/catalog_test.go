package providercatalog

import "testing"

func TestBuiltinSpecsRegistered(t *testing.T) {
	t.Parallel()
	want := []string{"openai", "openai_compatible", "anthropic", "gemini", "bedrock", "elevenlabs", "echo"}
	for _, typ := range want {
		if _, ok := Lookup(typ); !ok {
			t.Fatalf("missing spec %q", typ)
		}
	}
	if n := len(All()); n < len(want) {
		t.Fatalf("All()=%d want >= %d", n, len(want))
	}
	ui := UIVisible()
	if len(ui) < len(want) {
		t.Fatalf("UIVisible()=%d", len(ui))
	}
	seed := Seedable()
	if len(seed) < len(want) {
		t.Fatalf("Seedable()=%d", len(seed))
	}
}

func TestBedrockAuthOptional(t *testing.T) {
	t.Parallel()
	if !AllowsEmptyAPIKey("bedrock") {
		t.Fatal("bedrock should allow empty api_key_env")
	}
	if AllowsEmptyAPIKey("openai") {
		t.Fatal("openai must require api key")
	}
}

func TestCatalogAliases(t *testing.T) {
	t.Parallel()
	got := CatalogAliases("openai_compatible")
	if len(got) != 1 || got[0] != "openai" {
		t.Fatalf("got=%v", got)
	}
	if CatalogAliases("anthropic") != nil {
		t.Fatal("anthropic should have no alias")
	}
}

func TestSeedProviderID(t *testing.T) {
	t.Parallel()
	s, _ := Lookup("openai_compatible")
	if s.SeedProviderID() != "prov_ollama" {
		t.Fatal(s.SeedProviderID())
	}
	s, _ = Lookup("bedrock")
	if s.SeedProviderID() != "prov_bedrock" {
		t.Fatal(s.SeedProviderID())
	}
}
