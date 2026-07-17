package provider

type SpeechToTextRequest struct {
	Audio Audio

	Language string

	Prompt string

	Temperature *float32
}

func (SpeechToTextRequest) Capability() Capability {
	return CapabilitySpeechToText
}