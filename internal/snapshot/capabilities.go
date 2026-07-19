package snapshot

// ProviderCapabilities describes what a provider adapter can do.
type ProviderCapabilities struct {
	Chat   bool `json:"chat"`
	Stream bool `json:"stream"`
	TTS    bool `json:"tts"`
	STT    bool `json:"stt"`
}

// DefaultCapabilities returns catalog defaults for a provider type.
func DefaultCapabilities(typ string) ProviderCapabilities {
	switch typ {
	case "openai", "openai_compatible":
		return ProviderCapabilities{Chat: true, Stream: true, TTS: true, STT: true}
	default:
		// anthropic, gemini, …
		return ProviderCapabilities{Chat: true, Stream: true}
	}
}

// NormalizeCapabilities fills empty capabilities from the type catalog.
func NormalizeCapabilities(typ string, c ProviderCapabilities) ProviderCapabilities {
	if !c.Chat && !c.Stream && !c.TTS && !c.STT {
		return DefaultCapabilities(typ)
	}
	return c
}

// DefaultAPIKeyEnv returns the usual env var name for a provider type.
func DefaultAPIKeyEnv(typ string) string {
	switch typ {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "openai_compatible":
		return "OLLAMA_API_KEY"
	default:
		return "OPENAI_API_KEY"
	}
}
