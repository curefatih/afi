package provider

type Model struct {
	ID string

	ProviderModelID string

	Name string

	Capabilities []Capability

	ContextWindow int

	MaxOutputTokens int

	Enabled bool
}
