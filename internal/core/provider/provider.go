package provider

type Provider struct {
	ID string

	Name string

	Enabled bool

	Capabilities []Capability
}
