package dataplane

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/snapshot"
)

// OpenAITransport is the outbound OpenAI-compatible HTTP surface used by chat + audio + embeddings.
type OpenAITransport interface {
	ChatCompletions(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
	AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error)
	AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error)
	Embeddings(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error)
}

// AnthropicTransport is the outbound Anthropic HTTP surface used by chat + /v1/messages.
type AnthropicTransport interface {
	PassThrough(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
	Messages(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// AudioBackend is the modality port for TTS/STT. OpenAI-compatible ChatProviders
// that implement OpenAITransportProvider expose this via the registry.
type AudioBackend interface {
	AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error)
	AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error)
}

// EmbeddingsBackend is the modality port for /v1/embeddings (OpenAI-compatible).
type EmbeddingsBackend interface {
	Embeddings(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error)
}

// MessagesBackend is the modality port for native Anthropic /v1/messages.
type MessagesBackend interface {
	PassThrough(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// OpenAITransportProvider is implemented by ChatProvider adapters that expose OpenAI HTTP.
type OpenAITransportProvider interface {
	OpenAITransport() OpenAITransport
}

// AnthropicTransportProvider is implemented by ChatProvider adapters that expose Anthropic HTTP.
type AnthropicTransportProvider interface {
	AnthropicTransport() AnthropicTransport
}

// OpenAITransport looks up an OpenAI-compatible transport by provider type.
func (r *Registry) OpenAITransport(typ string) (OpenAITransport, bool) {
	if r == nil {
		return nil, false
	}
	cp, ok := r.Get(typ)
	if !ok {
		return nil, false
	}
	a, ok := cp.(OpenAITransportProvider)
	if !ok {
		return nil, false
	}
	t := a.OpenAITransport()
	if t == nil {
		return nil, false
	}
	return t, true
}

// AnthropicTransport looks up an Anthropic transport by provider type.
func (r *Registry) AnthropicTransport(typ string) (AnthropicTransport, bool) {
	if r == nil {
		return nil, false
	}
	cp, ok := r.Get(typ)
	if !ok {
		return nil, false
	}
	a, ok := cp.(AnthropicTransportProvider)
	if !ok {
		return nil, false
	}
	t := a.AnthropicTransport()
	if t == nil {
		return nil, false
	}
	return t, true
}

// AudioBackend returns the TTS/STT port for a provider type (OpenAI-compatible only).
func (r *Registry) AudioBackend(typ string) (AudioBackend, bool) {
	t, ok := r.OpenAITransport(typ)
	if !ok {
		return nil, false
	}
	return t, true
}

// EmbeddingsBackend returns the /v1/embeddings port for a provider type (OpenAI-compatible only).
func (r *Registry) EmbeddingsBackend(typ string) (EmbeddingsBackend, bool) {
	t, ok := r.OpenAITransport(typ)
	if !ok {
		return nil, false
	}
	return t, true
}

// MessagesBackend returns the native /v1/messages port for a provider type.
func (r *Registry) MessagesBackend(typ string) (MessagesBackend, bool) {
	t, ok := r.AnthropicTransport(typ)
	if !ok {
		return nil, false
	}
	return t, true
}

// audioBackend resolves TTS/STT by the routed provider's type (not a hardcoded "openai").
func (p *Pipeline) audioBackend(providerType string) (AudioBackend, error) {
	if p.Providers == nil {
		return nil, fmt.Errorf("audio backend not registered for type %q", providerType)
	}
	b, ok := p.Providers.AudioBackend(providerType)
	if !ok {
		return nil, fmt.Errorf("audio backend not registered for type %q", providerType)
	}
	return b, nil
}

// embeddingsBackend resolves /v1/embeddings by the routed provider's type.
func (p *Pipeline) embeddingsBackend(providerType string) (EmbeddingsBackend, error) {
	if p.Providers == nil {
		return nil, fmt.Errorf("embeddings backend not registered for type %q", providerType)
	}
	b, ok := p.Providers.EmbeddingsBackend(providerType)
	if !ok {
		return nil, fmt.Errorf("embeddings backend not registered for type %q", providerType)
	}
	return b, nil
}

// messagesBackend resolves native /v1/messages by the routed provider's type.
func (p *Pipeline) messagesBackend(providerType string) (MessagesBackend, error) {
	if p.Providers == nil {
		return nil, fmt.Errorf("messages backend not registered for type %q", providerType)
	}
	b, ok := p.Providers.MessagesBackend(providerType)
	if !ok {
		return nil, fmt.Errorf("messages backend not registered for type %q", providerType)
	}
	return b, nil
}
