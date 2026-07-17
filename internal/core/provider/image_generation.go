package provider

type ImageGenerationRequest struct {
	Prompt string

	NegativePrompt string

	Width int

	Height int

	Count int

	Quality ImageQuality

	ResponseFormat ImageResponseFormat
}

func (ImageGenerationRequest) Capability() Capability {
	return CapabilityImageGeneration
}