package llm

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane"
	"github.com/curefatih/afi/internal/snapshot"
)

// DefaultRegistry registers built-in OpenAI, Anthropic, Gemini, and openai_compatible adapters.
func DefaultRegistry(sec secrets.Resolver) *dataplane.Registry {
	if sec == nil {
		sec = secrets.Default()
	}
	oai := NewOpenAIClient(sec)
	return dataplane.NewRegistry().
		Register(newOpenAIChatProvider("openai", oai, dataplane.ProviderCaps{Chat: true, Stream: true})).
		Register(newOpenAIChatProvider("openai_compatible", NewOpenAIClient(sec), dataplane.ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(NewAnthropicClient(sec))).
		Register(newGeminiChatProvider(NewGeminiClient(sec)))
}

// RegistryWithOpenAI builds DefaultRegistry but uses the given OpenAI client for type "openai"
// (tests inject mock HTTP transports).
func RegistryWithOpenAI(openai *OpenAIClient, sec secrets.Resolver) *dataplane.Registry {
	if sec == nil {
		sec = secrets.Default()
	}
	if openai == nil {
		openai = NewOpenAIClient(sec)
	}
	return dataplane.NewRegistry().
		Register(newOpenAIChatProvider("openai", openai, dataplane.ProviderCaps{Chat: true, Stream: true})).
		Register(newOpenAIChatProvider("openai_compatible", NewOpenAIClient(sec), dataplane.ProviderCaps{Chat: true, Stream: true})).
		Register(newAnthropicChatProvider(NewAnthropicClient(sec))).
		Register(newGeminiChatProvider(NewGeminiClient(sec)))
}

type openaiChatProvider struct {
	typ    string
	client *OpenAIClient
	caps   dataplane.ProviderCaps
}

func newOpenAIChatProvider(typ string, client *OpenAIClient, caps dataplane.ProviderCaps) *openaiChatProvider {
	return &openaiChatProvider{typ: typ, client: client, caps: caps}
}

func (p *openaiChatProvider) Type() string                         { return p.typ }
func (p *openaiChatProvider) Capabilities() dataplane.ProviderCaps { return p.caps }

func (p *openaiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.ChatCompletions(ctx, provider, targetModel, body, stream)
}

func (p *openaiChatProvider) OpenAITransport() dataplane.OpenAITransport { return p.client }

type anthropicChatProvider struct {
	client *AnthropicClient
}

func newAnthropicChatProvider(client *AnthropicClient) *anthropicChatProvider {
	return &anthropicChatProvider{client: client}
}

func (p *anthropicChatProvider) Type() string { return "anthropic" }
func (p *anthropicChatProvider) Capabilities() dataplane.ProviderCaps {
	return dataplane.ProviderCaps{Chat: true, Stream: true}
}

func (p *anthropicChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.Messages(ctx, provider, targetModel, body, stream)
}

func (p *anthropicChatProvider) AnthropicTransport() dataplane.AnthropicTransport { return p.client }

type geminiChatProvider struct {
	client *GeminiClient
}

func newGeminiChatProvider(client *GeminiClient) *geminiChatProvider {
	return &geminiChatProvider{client: client}
}

func (p *geminiChatProvider) Type() string { return "gemini" }
func (p *geminiChatProvider) Capabilities() dataplane.ProviderCaps {
	return dataplane.ProviderCaps{Chat: true, Stream: true}
}

func (p *geminiChatProvider) Chat(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return p.client.GenerateContent(ctx, provider, targetModel, body, stream)
}
