package provider

type RerankRequest struct {
	Query string

	Documents []string

	TopK int
}

func (RerankRequest) Capability() Capability {
	return CapabilityRerank
}