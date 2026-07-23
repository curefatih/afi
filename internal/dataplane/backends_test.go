package dataplane

import (
	"testing"

	"github.com/curefatih/afi/internal/adapters/llm"
)

func TestRegistryAudioBackendByType(t *testing.T) {
	t.Parallel()
	reg := RegistryFromClients(llm.NewClients(nil))
	if _, ok := reg.AudioBackend("openai"); !ok {
		t.Fatal("expected openai audio backend")
	}
	if _, ok := reg.AudioBackend("openai_compatible"); !ok {
		t.Fatal("expected openai_compatible audio backend")
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
	reg := RegistryFromClients(llm.NewClients(nil))
	if _, ok := reg.MessagesBackend("anthropic"); !ok {
		t.Fatal("expected anthropic messages backend")
	}
	if _, ok := reg.MessagesBackend("openai"); ok {
		t.Fatal("openai must not expose messages backend")
	}
}

func TestOpenAIProviderCapsIncludeAudio(t *testing.T) {
	t.Parallel()
	reg := RegistryFromClients(llm.NewClients(nil))
	cp, ok := reg.Get("openai")
	if !ok {
		t.Fatal("missing openai")
	}
	caps := cp.Capabilities()
	if !caps.TTS || !caps.STT || !caps.Embedding {
		t.Fatalf("caps=%+v", caps)
	}
}

func TestRegistryEmbeddingsBackendByType(t *testing.T) {
	t.Parallel()
	reg := RegistryFromClients(llm.NewClients(nil))
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
