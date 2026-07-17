package provider

type Request struct {
	Provider   Provider
	Model      Model
	Capability Capability
	Payload    Payload
	Metadata   map[string]any
}
