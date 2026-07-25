package dataplane

import (
	"github.com/curefatih/afi/internal/adapters/llm"
	"github.com/curefatih/afi/internal/adapters/secrets"
)

func init() {
	registerBuiltin("openai", func(sec secrets.Resolver) ChatProvider {
		return newOpenAIChatProvider("openai", llm.NewOpenAIClient(sec), providerCapsFromSpec("openai"))
	})
	registerBuiltin("openai_compatible", func(sec secrets.Resolver) ChatProvider {
		return newOpenAIChatProvider("openai_compatible", llm.NewOpenAIClient(sec), providerCapsFromSpec("openai_compatible"))
	})
	registerBuiltin("azure_openai", func(sec secrets.Resolver) ChatProvider {
		return newOpenAIChatProvider("azure_openai", llm.NewAzureOpenAIClient(sec), providerCapsFromSpec("azure_openai"))
	})
	registerBuiltin("anthropic", func(sec secrets.Resolver) ChatProvider {
		return newAnthropicChatProvider(llm.NewAnthropicClient(sec))
	})
	registerBuiltin("gemini", func(sec secrets.Resolver) ChatProvider {
		return newGeminiChatProvider(llm.NewGeminiClient(sec))
	})
	registerBuiltin("bedrock", func(sec secrets.Resolver) ChatProvider {
		return newBedrockChatProvider(llm.NewBedrockClient(sec))
	})
	registerBuiltin("elevenlabs", func(sec secrets.Resolver) ChatProvider {
		return newElevenLabsProvider(llm.NewElevenLabsClient(sec))
	})
}
