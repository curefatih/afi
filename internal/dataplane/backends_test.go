package dataplane

import (
	"testing"
)

func TestRegistryAudioBackendByType(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	if _, ok := reg.AudioBackend("openai"); !ok {
		t.Fatal("expected openai audio backend")
	}
	if _, ok := reg.AudioBackend("openai_compatible"); !ok {
		t.Fatal("expected openai_compatible audio backend")
	}
	if _, ok := reg.AudioBackend("elevenlabs"); !ok {
		t.Fatal("expected elevenlabs audio backend")
	}
	if _, ok := reg.AudioBackend("anthropic"); ok {
		t.Fatal("anthropic must not expose audio backend")
	}
	if _, ok := reg.AudioBackend("gemini"); ok {
		t.Fatal("gemini must not expose audio backend")
	}
}

func TestRegistryMessagesBackendByType(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	if _, ok := reg.MessagesBackend("anthropic"); !ok {
		t.Fatal("expected anthropic messages backend")
	}
	if _, ok := reg.MessagesBackend("openai"); ok {
		t.Fatal("openai must not expose messages backend")
	}
}

func TestOpenAIProviderCapsIncludeAudio(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	cp, ok := reg.Get("openai")
	if !ok {
		t.Fatal("missing openai")
	}
	caps := cp.Capabilities()
	if !caps.TTS || !caps.STT || !caps.Embedding || !caps.Image {
		t.Fatalf("caps=%+v", caps)
	}
}

func TestRegistryEmbeddingsBackendByType(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	if _, ok := reg.EmbeddingsBackend("openai"); !ok {
		t.Fatal("expected openai embeddings backend")
	}
	if _, ok := reg.EmbeddingsBackend("openai_compatible"); !ok {
		t.Fatal("expected openai_compatible embeddings backend")
	}
	if _, ok := reg.EmbeddingsBackend("anthropic"); ok {
		t.Fatal("anthropic must not expose embeddings backend")
	}
}

func TestDefaultRegistryIncludesBuiltins(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	for _, typ := range []string{"openai", "openai_compatible", "anthropic", "gemini", "bedrock", "elevenlabs"} {
		if _, ok := reg.Get(typ); !ok {
			t.Fatalf("missing builtin %q", typ)
		}
	}
}
