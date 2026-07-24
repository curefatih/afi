package provider

import (
	"context"

	"github.com/curefatih/afi/sdk/chatir"
)

// Capabilities describes what an adapter supports on the gateway surface.
type Capabilities struct {
	Chat      bool `json:"chat"`
	Stream    bool `json:"stream"`
	TTS       bool `json:"tts"`
	STT       bool `json:"stt"`
	Embedding bool `json:"embedding"`
}

// ProviderConfig is the snapshot view passed into ChatIR (IDs, base URL, key env).
// Mirrors internal/snapshot.Provider fields needed by adapters.
type ProviderConfig struct {
	ID           string
	Type         string
	BaseURL      string
	APIKeyEnv    string
	Name         string
	Capabilities Capabilities
}

// ChatProvider is the in-process adapter contract for out-of-tree extensions.
//
// Example wiring (gateway bootstrap):
//
//	reg := dataplane.DefaultRegistry()
//	reg.RegisterSDK(echo.New())
//	pipeline := dataplane.NewPipelineWithRegistry(holder, reg, log)
type ChatProvider interface {
	Type() string
	Capabilities() Capabilities
	ChatIR(ctx context.Context, cfg ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error)
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
