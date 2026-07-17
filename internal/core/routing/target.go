package routing

type Target struct {
	ProviderID string

	ModelID string

	Weight int

	Priority int

	Enabled bool
}
