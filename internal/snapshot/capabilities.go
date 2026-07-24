package snapshot

// ProviderCapabilities describes what a provider adapter can do.
type ProviderCapabilities struct {
	Chat      bool `json:"chat"`
	Stream    bool `json:"stream"`
	TTS       bool `json:"tts"`
	STT       bool `json:"stt"`
	Embedding bool `json:"embedding"`
	Image     bool `json:"image"`
}

// DefaultCapabilities returns catalog defaults for a provider type.
func DefaultCapabilities(typ string) ProviderCapabilities {
	switch typ {
	case "openai", "openai_compatible":
		return ProviderCapabilities{Chat: true, Stream: true, TTS: true, STT: true, Embedding: true, Image: true}
	case "echo":
		return ProviderCapabilities{Chat: true, Stream: false}
	default:
		// anthropic, gemini, …
		return ProviderCapabilities{Chat: true, Stream: true}
	}
}

// NormalizeCapabilities fills empty capabilities from the type catalog.
func NormalizeCapabilities(typ string, c ProviderCapabilities) ProviderCapabilities {
	def := DefaultCapabilities(typ)
	if !c.Chat && !c.Stream && !c.TTS && !c.STT && !c.Embedding && !c.Image {
		return def
	}
	// Older snapshots only stored chat/stream. Promote TTS/STT/embedding/image from type
	// defaults when unset so openai providers keep working after modality cycles.
	if !c.TTS && !c.STT && (def.TTS || def.STT) {
		c.TTS = def.TTS
		c.STT = def.STT
	}
	if !c.Embedding && def.Embedding {
		c.Embedding = true
	}
	if !c.Image && def.Image {
		c.Image = true
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
	case "echo":
		return "ECHO_UNUSED"
	default:
		return "OPENAI_API_KEY"
	}
}
