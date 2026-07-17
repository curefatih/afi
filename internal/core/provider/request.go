package provider

type Request struct {
	Provider Provider

	Model Model

	Capability Capability

	Body any

	Metadata map[string]any
}
