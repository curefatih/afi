package provider

type ChatRequest struct {
	Messages []Message

	System string

	Tools []Tool

	Temperature *float32

	TopP *float32

	MaxTokens *int

	Stop []string

	ResponseFormat *ResponseFormat

	Stream bool
}

func (ChatRequest) Capability() Capability {
	return CapabilityChat
}