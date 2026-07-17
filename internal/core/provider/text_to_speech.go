package provider

type TextToSpeechRequest struct {
	Text string

	Voice string

	Speed float32

	Format AudioFormat
}

func (TextToSpeechRequest) Capability() Capability {
	return CapabilityTextToSpeech
}