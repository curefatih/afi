package provider

type Selector interface {
	Select(
		model string,
		capability Capability,
	) (*Provider, error)
}
