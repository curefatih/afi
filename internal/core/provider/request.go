package provider

type Request struct {
	APIKey     string
	Provider   Provider
	Model      Model
	Capability Capability
	Payload    Payload
	Metadata   map[string]any
}
