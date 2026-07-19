package dataplane

import (
	"context"
	"net/http"
	"sync"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/snapshot"
)

// ProviderCaps mirrors snapshot capabilities for adapters.
type ProviderCaps struct {
	Chat   bool
	Stream bool
	TTS    bool
	STT    bool
}

// ChatProvider is the in-process adapter contract for gateway chat.
type ChatProvider interface {
	Type() string
	Capabilities() ProviderCaps
	Chat(ctx context.Context, p snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// Registry maps provider type strings to ChatProvider implementations.
type Registry struct {
	mu     sync.RWMutex
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
	return RegistryFromClients(llm.NewClients(secrets.Default()))
}

// RegistryFromClients wires ChatProvider adapters over outbound LLM clients.
func RegistryFromClients(c *llm.Clients) *Registry {
	if c == nil {
		c = llm.NewClients(nil)
	}
	return NewRegistry().
		Register(newOpenAIChatProvider("openai", c.OpenAI, ProviderCaps{Chat: true, Stream: true, TTS: true, STT: true})).
		Register(newOpenAIChatProvider("openai_compatible", c.OpenAICompatible, ProviderCaps{Chat: true, Stream: true, TTS: true, STT: true})).
		Register(newAnthropicChatProvider(c.Anthropic)).
		Register(newGeminiChatProvider(c.Gemini))
}

// RegistryWithOpenAI builds DefaultRegistry but uses the given OpenAI client for type "openai"
// (tests inject mock HTTP transports).
func RegistryWithOpenAI(openai *llm.OpenAIClient) *Registry {
	c := llm.NewClients(nil)
	if openai != nil {
		c.OpenAI = openai
	}
	return RegistryFromClients(c)
}

type openaiChatProvider struct {
	typ    string
	client *llm.OpenAIClient
	caps   ProviderCaps
}

func newOpenAIChatProvider(typ string, client *llm.OpenAIClient, caps ProviderCaps) *openaiChatProvider {
	return &openaiChatProvider{typ: typ, client: client, caps: caps}
}

func (p *openaiChatProvider) Type() string              { return p.typ }
func (p *openaiChatProvider) Capabilities() ProviderCaps { return p.caps }

func (p *openaiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.ChatCompletions(ctx, provider, targetModel, body, stream)
}

func (p *openaiChatProvider) OpenAITransport() OpenAITransport {
	if p.client == nil {
		return nil
	}
	return p.client
}

type anthropicChatProvider struct {
	client *llm.AnthropicClient
}

func newAnthropicChatProvider(client *llm.AnthropicClient) *anthropicChatProvider {
	return &anthropicChatProvider{client: client}
}

func (p *anthropicChatProvider) Type() string { return "anthropic" }
func (p *anthropicChatProvider) Capabilities() ProviderCaps {
	return ProviderCaps{Chat: true, Stream: true}
}

func (p *anthropicChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.Messages(ctx, provider, targetModel, body, stream)
}

func (p *anthropicChatProvider) AnthropicTransport() AnthropicTransport {
	if p.client == nil {
		return nil
	}
	return p.client
}

type geminiChatProvider struct {
	client *llm.GeminiClient
}

func newGeminiChatProvider(client *llm.GeminiClient) *geminiChatProvider {
	return &geminiChatProvider{client: client}
}

func (p *geminiChatProvider) Type() string { return "gemini" }
func (p *geminiChatProvider) Capabilities() ProviderCaps {
	return ProviderCaps{Chat: true, Stream: true}
}

func (p *geminiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.GenerateContent(ctx, provider, targetModel, body, stream)
}
