package dataplane

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/curefatih/afi/internal/snapshot"
)

// ProviderCaps mirrors snapshot capabilities for adapters.
type ProviderCaps struct {
	Chat   bool
	Stream bool
}

// ChatProvider is the in-process adapter contract for gateway chat.
type ChatProvider interface {
	Type() string
	Capabilities() ProviderCaps
	Chat(ctx context.Context, p snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// Registry maps provider type strings to ChatProvider implementations.
type Registry struct {
	mu   sync.RWMutex
	byType map[string]ChatProvider
}

func NewRegistry() *Registry {
	return &Registry{byType: make(map[string]ChatProvider)}
}

func (r *Registry) Register(p ChatProvider) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byType[p.Type()] = p
	return r
}

func (r *Registry) Get(typ string) (ChatProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.byType[typ]
	return p, ok
}

func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byType))
	for t := range r.byType {
		out = append(out, t)
	}
	return out
}

// DefaultRegistry registers built-in OpenAI, Anthropic, Gemini, and openai_compatible adapters.
func DefaultRegistry() *Registry {
	oai := NewOpenAIClient()
	return NewRegistry().
		Register(newOpenAIChatProvider("openai", oai, ProviderCaps{Chat: true, Stream: true})).
		Register(newOpenAIChatProvider("openai_compatible", NewOpenAIClient(), ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(NewAnthropicClient())).
		Register(newGeminiChatProvider(NewGeminiClient()))
}

// RegistryWithOpenAI builds DefaultRegistry but uses the given OpenAI client for type "openai"
// (tests inject mock HTTP transports).
func RegistryWithOpenAI(openai *OpenAIClient) *Registry {
	if openai == nil {
		openai = NewOpenAIClient()
	}
	return NewRegistry().
		Register(newOpenAIChatProvider("openai", openai, ProviderCaps{Chat: true, Stream: true})).
		Register(newOpenAIChatProvider("openai_compatible", NewOpenAIClient(), ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(NewAnthropicClient())).
		Register(newGeminiChatProvider(NewGeminiClient()))
}

type openaiChatProvider struct {
	typ    string
	client *OpenAIClient
	caps   ProviderCaps
}

func newOpenAIChatProvider(typ string, client *OpenAIClient, caps ProviderCaps) *openaiChatProvider {
	return &openaiChatProvider{typ: typ, client: client, caps: caps}
}

func (p *openaiChatProvider) Type() string              { return p.typ }
func (p *openaiChatProvider) Capabilities() ProviderCaps { return p.caps }

func (p *openaiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.ChatCompletions(ctx, provider, targetModel, body, stream)
}

type anthropicChatProvider struct {
	client *AnthropicClient
}

func newAnthropicChatProvider(client *AnthropicClient) *anthropicChatProvider {
	return &anthropicChatProvider{client: client}
}

func (p *anthropicChatProvider) Type() string { return "anthropic" }
func (p *anthropicChatProvider) Capabilities() ProviderCaps {
	return ProviderCaps{Chat: true, Stream: true}
}

func (p *anthropicChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.Messages(ctx, provider, targetModel, body, stream)
}

type geminiChatProvider struct {
	client *GeminiClient
}

func newGeminiChatProvider(client *GeminiClient) *geminiChatProvider {
	return &geminiChatProvider{client: client}
}

func (p *geminiChatProvider) Type() string { return "gemini" }
func (p *geminiChatProvider) Capabilities() ProviderCaps {
	return ProviderCaps{Chat: true, Stream: false}
}

func (p *geminiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	if stream {
		return nil, fmt.Errorf("streaming is not supported for provider type %q", p.Type())
	}
	return p.client.GenerateContent(ctx, provider, targetModel, body)
}
