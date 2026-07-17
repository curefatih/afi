package provider

type Capability string

const (
	CapabilityChat Capability = "CHAT"

	CapabilityEmbedding Capability = "EMBEDDING"

	CapabilityImageGeneration Capability = "IMAGE_GENERATION"

	CapabilityImageEdit Capability = "IMAGE_EDIT"

	CapabilitySpeechToText Capability = "SPEECH_TO_TEXT"

	CapabilityTextToSpeech Capability = "TEXT_TO_SPEECH"

	CapabilityModeration Capability = "MODERATION"

	CapabilityRerank Capability = "RERANK"

	CapabilityVideoGeneration Capability = "VIDEO_GENERATION"

	CapabilityRealtime Capability = "REALTIME"
)
