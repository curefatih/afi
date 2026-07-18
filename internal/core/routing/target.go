package routing

type Target struct {
	ProviderID string

	ProviderModelID string

	Weight int

	Priority int

	Enabled bool
}
