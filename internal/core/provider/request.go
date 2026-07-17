package provider

type Request struct {
	Model string

	Capability Capability

	Body any
}
