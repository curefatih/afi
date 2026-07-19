package provider

import (
	"context"
	"net/http"
)

// Capabilities describes what an adapter supports on the OpenAI-compatible
// gateway surface.
type Capabilities struct {
	Chat   bool `json:"chat"`
	Stream bool `json:"stream"`
	TTS    bool `json:"tts"`
	STT    bool `json:"stt"`
}

// ProviderConfig is the snapshot view passed into Chat (IDs, base URL, key env).
// Mirrors internal/snapshot.Provider fields needed by adapters.
type ProviderConfig struct {
	ID           string
	Type         string
	BaseURL      string
	APIKeyEnv    string
	Name         string
	Capabilities Capabilities
}

// ChatProvider is the stable in-process adapter contract for out-of-tree
// extensions (see extensions/).
//
// Example wiring (gateway bootstrap):
//
//	reg := dataplane.DefaultRegistry()
//	reg.RegisterSDK(echo.New())
//	pipeline := dataplane.NewPipelineWithRegistry(holder, reg, log)
type ChatProvider interface {
	Type() string
	Capabilities() Capabilities
	Chat(ctx context.Context, cfg ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// ConfigFromFields builds a ProviderConfig from discrete snapshot-like fields.
func ConfigFromFields(id, typ, name, baseURL, apiKeyEnv string, caps Capabilities) ProviderConfig {
	return ProviderConfig{
		ID:           id,
		Type:         typ,
		Name:         name,
		BaseURL:      baseURL,
		APIKeyEnv:    apiKeyEnv,
		Capabilities: caps,
	}
}
