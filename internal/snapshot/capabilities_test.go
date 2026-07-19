package snapshot

import "testing"

func TestDefaultCapabilities(t *testing.T) {
	t.Parallel()
	g := DefaultCapabilities("gemini")
	if !g.Chat || !g.Stream {
		t.Fatalf("gemini=%+v", g)
	}
	o := DefaultCapabilities("openai_compatible")
	if !o.Chat || !o.Stream {
		t.Fatalf("compat=%+v", o)
	}
}

func TestNormalizeCapabilitiesEmpty(t *testing.T) {
	t.Parallel()
	got := NormalizeCapabilities("gemini", ProviderCapabilities{})
	if !got.Stream {
		t.Fatal("expected stream by default")
	}
}

func TestDefaultAPIKeyEnv(t *testing.T) {
	t.Parallel()
	if DefaultAPIKeyEnv("openai_compatible") != "OLLAMA_API_KEY" {
		t.Fatal(DefaultAPIKeyEnv("openai_compatible"))
	}
}
