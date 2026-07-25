package dataplane

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
)

// ProviderCaps mirrors snapshot capabilities for adapters.
type ProviderCaps struct {
	Chat      bool
	Stream    bool
	TTS       bool
	STT       bool
	Embedding bool
	Image     bool
}

// ChatProvider is the in-process adapter contract for gateway chat HTTP transport.
type ChatProvider interface {
	Type() string
	Capabilities() ProviderCaps
	Chat(ctx context.Context, p snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// IRChatProvider is implemented by built-in adapters that speak chat IR.
type IRChatProvider interface {
	ChatIR(ctx context.Context, p snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error)
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

// DefaultRegistry registers all in-tree adapters via registerBuiltin factories.
func DefaultRegistry() *Registry {
	return buildBuiltinRegistry(secrets.Default())
}

// RegistryWithSecrets builds the builtin registry with a custom secret resolver.
func RegistryWithSecrets(sec secrets.Resolver) *Registry {
	return buildBuiltinRegistry(sec)
}

// RegistryWithOpenAI builds DefaultRegistry but uses the given OpenAI client for type "openai"
// (tests inject mock HTTP transports).
func RegistryWithOpenAI(openai *llm.OpenAIClient) *Registry {
	reg := DefaultRegistry()
	if openai != nil {
		reg.Register(newOpenAIChatProvider("openai", openai, providerCapsFromSpec("openai")))
	}
	return reg
}

type openAIHTTPClient interface {
	OpenAITransport
	ChatIR(ctx context.Context, p snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error)
}

type openaiChatProvider struct {
	typ    string
	client openAIHTTPClient
	caps   ProviderCaps
}

func newOpenAIChatProvider(typ string, client openAIHTTPClient, caps ProviderCaps) *openaiChatProvider {
	return &openaiChatProvider{typ: typ, client: client, caps: caps}
}

func (p *openaiChatProvider) Type() string               { return p.typ }
func (p *openaiChatProvider) Capabilities() ProviderCaps { return p.caps }

func (p *openaiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.ChatCompletions(ctx, provider, targetModel, body, stream)
}

func (p *openaiChatProvider) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return p.client.ChatIR(ctx, provider, targetModel, req)
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
	return providerCapsFromSpec("anthropic")
}

func (p *anthropicChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.Messages(ctx, provider, targetModel, body, stream)
}

func (p *anthropicChatProvider) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return p.client.ChatIR(ctx, provider, targetModel, req)
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
	return providerCapsFromSpec("gemini")
}

func (p *geminiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.GenerateContent(ctx, provider, targetModel, body, stream)
}

func (p *geminiChatProvider) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return p.client.ChatIR(ctx, provider, targetModel, req)
}

type bedrockChatProvider struct {
	client *llm.BedrockClient
}

func newBedrockChatProvider(client *llm.BedrockClient) *bedrockChatProvider {
	return &bedrockChatProvider{client: client}
}

func (p *bedrockChatProvider) Type() string { return "bedrock" }
func (p *bedrockChatProvider) Capabilities() ProviderCaps {
	return providerCapsFromSpec("bedrock")
}

func (p *bedrockChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return nil, errors.New("bedrock provider requires ChatIR")
}

func (p *bedrockChatProvider) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return p.client.ChatIR(ctx, provider, targetModel, req)
}

type elevenLabsProvider struct {
	client *llm.ElevenLabsClient
}

func newElevenLabsProvider(client *llm.ElevenLabsClient) *elevenLabsProvider {
	return &elevenLabsProvider{client: client}
}

func (p *elevenLabsProvider) Type() string { return "elevenlabs" }
func (p *elevenLabsProvider) Capabilities() ProviderCaps {
	return providerCapsFromSpec("elevenlabs")
}

func (p *elevenLabsProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return nil, errors.New("elevenlabs provider does not support chat")
}

func (p *elevenLabsProvider) AudioBackend() AudioBackend {
	if p.client == nil {
		return nil
	}
	return p.client
}
