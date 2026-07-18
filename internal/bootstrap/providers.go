package bootstrap

import (
	"github.com/curefatih/afi/internal/adapters/providers/openai"
	"github.com/curefatih/afi/internal/core/provider"
)

type Providers struct {
	Registry provider.Registry
}

func buildProviders(
	cfg Config,
	transports *Transports,
) (*Providers, error) {
	registry := NewProviderRegistry()

	registry.Register(
		provider.Provider{
			ID:      "openai",
			Name:    "openai",
			Enabled: true,
			Capabilities: []provider.Capability{
				provider.CapabilityChat,
				provider.CapabilityEmbedding,
				provider.CapabilityImageGeneration,
				provider.CapabilityImageEdit,
				provider.CapabilitySpeechToText,
				provider.CapabilityTextToSpeech,
				provider.CapabilityModeration,
				provider.CapabilityRerank,
				provider.CapabilityVideoGeneration,
				provider.CapabilityRealtime,
			},
		},
		openai.NewClient(
			openai.Config{
				BaseURL: cfg.OpenAI.BaseURL,
				APIKey:  cfg.OpenAI.APIKey,
			},
			transports.HTTP,
		),
	)

	return &Providers{
		Registry: registry,
	}, nil
}

type ProviderRegistry struct {
}

func NewProviderRegistry() provider.Registry {
	return ProviderRegistry{}
}

var _ provider.Registry = ProviderRegistry{}

// Client implements provider.Registry.
func (p ProviderRegistry) Client(providerID string) (provider.Client, error) {
	panic("unimplemented")
}

// GetProviderByID implements provider.Registry.
func (p ProviderRegistry) GetProviderByID(providerID string) (provider.Provider, error) {
	panic("unimplemented")
}

// List implements provider.Registry.
func (p ProviderRegistry) List() []provider.Provider {
	panic("unimplemented")
}

// Register implements provider.Registry.
func (p ProviderRegistry) Register(provider provider.Provider, client provider.Client) {
	panic("unimplemented")
}
