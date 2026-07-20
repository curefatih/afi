package llm

import (
	"github.com/curefatih/afi/internal/adapters/secrets"
)

// Clients holds the built-in outbound LLM HTTP clients.
type Clients struct {
	OpenAI           *OpenAIClient
	OpenAICompatible *OpenAIClient
	Anthropic        *AnthropicClient
	Gemini           *GeminiClient
}

// NewClients constructs vendor clients with a shared secret resolver.
func NewClients(sec secrets.Resolver) *Clients {
	if sec == nil {
		sec = secrets.Default()
	}
	return &Clients{
		OpenAI:           NewOpenAIClient(sec),
		OpenAICompatible: NewOpenAIClient(sec),
		Anthropic:        NewAnthropicClient(sec),
		Gemini:           NewGeminiClient(sec),
	}
}
