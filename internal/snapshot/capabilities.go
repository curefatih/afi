package snapshot

import (
	"github.com/curefatih/afi/internal/providercatalog"
)

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
	if s, ok := providercatalog.Lookup(typ); ok {
		return capsFromCatalog(s.Capabilities)
	}
	// Unknown types default to chat+stream (historical behavior for anthropic/gemini-like).
	return ProviderCapabilities{Chat: true, Stream: true}
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
	if s, ok := providercatalog.Lookup(typ); ok {
		return s.DefaultAPIKeyEnv
	}
	return "OPENAI_API_KEY"
}

func capsFromCatalog(c providercatalog.Capabilities) ProviderCapabilities {
	return ProviderCapabilities{
		Chat: c.Chat, Stream: c.Stream, TTS: c.TTS, STT: c.STT,
		Embedding: c.Embedding, Image: c.Image,
	}
}
