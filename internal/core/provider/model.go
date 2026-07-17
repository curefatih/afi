package provider

type Model struct {
	ID string

	ProviderID string

	Name string

	Capabilities []Capability

	ContextWindow int

	MaxOutputTokens int

	Enabled bool
}
