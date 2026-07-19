package modelcatalog

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

// Mode values align with LiteLLM model_prices_and_context_window.json.
const (
	ModeChat               = "chat"
	ModeAudioSpeech        = "audio_speech"
	ModeAudioTranscription = "audio_transcription"
)

// Entry is curated metadata for a provider target model.
type Entry struct {
	Mode                  string   `json:"mode"`
	MaxInputTokens        int      `json:"max_input_tokens,omitempty"`
	MaxOutputTokens       int      `json:"max_output_tokens,omitempty"`
	InputCostPerMTok      *float64 `json:"input_cost_per_mtok,omitempty"`
	OutputCostPerMTok     *float64 `json:"output_cost_per_mtok,omitempty"`
	InputCostPerCharacter *float64 `json:"input_cost_per_character,omitempty"`
	InputCostPerSecond    *float64 `json:"input_cost_per_second,omitempty"`
	SupportsStreaming     *bool    `json:"supports_streaming,omitempty"`
	SupportsVision        bool     `json:"supports_vision,omitempty"`
	SupportsTools         bool     `json:"supports_tools,omitempty"`
}

//go:embed catalog.json
var catalogJSON []byte

var (
	loadOnce sync.Once
	entries  map[string]Entry
	loadErr  error
)

func load() {
	loadOnce.Do(func() {
		entries = make(map[string]Entry)
		loadErr = json.Unmarshal(catalogJSON, &entries)
	})
}

// Key builds the catalog lookup key provider_type/model.
func Key(providerType, model string) string {
	return strings.ToLower(strings.TrimSpace(providerType)) + "/" + strings.TrimSpace(model)
}

// Lookup returns curated metadata for (providerType, targetModel).
// openai_compatible falls back to openai entries for shared model ids.
func Lookup(providerType, model string) (Entry, bool) {
	load()
	if loadErr != nil || model == "" {
		return Entry{}, false
	}
	providerType = strings.ToLower(strings.TrimSpace(providerType))
	model = strings.TrimSpace(model)
	if e, ok := entries[Key(providerType, model)]; ok {
		return e, true
	}
	for _, alias := range providerAliases(providerType) {
		if e, ok := entries[Key(alias, model)]; ok {
			return e, true
		}
	}
	return Entry{}, false
}

func providerAliases(providerType string) []string {
	switch providerType {
	case "openai_compatible":
		return []string{"openai"}
	default:
		return nil
	}
}

// IsChat reports whether the entry is a chat/completions model.
func (e Entry) IsChat() bool {
	return e.Mode == ModeChat || e.Mode == ""
}

// IsTTS reports whether the entry is a text-to-speech model.
func (e Entry) IsTTS() bool {
	return e.Mode == ModeAudioSpeech
}

// IsSTT reports whether the entry is a speech-to-text model.
func (e Entry) IsSTT() bool {
	return e.Mode == ModeAudioTranscription
}

// StreamingEnabled returns whether streaming is supported (chat default true).
func (e Entry) StreamingEnabled() bool {
	if e.SupportsStreaming != nil {
		return *e.SupportsStreaming
	}
	return e.IsChat()
}
