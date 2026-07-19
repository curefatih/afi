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

// ChatProvider is the stable in-process adapter contract.
//
// Example wiring (gateway bootstrap):
//
//	reg := dataplane.NewRegistry()
//	reg.Register(myAdapter)
//	pipeline := dataplane.NewPipelineWithRegistry(holder, reg, log)
type ChatProvider interface {
	Type() string
	Capabilities() Capabilities
	Chat(ctx context.Context, cfg ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error)
}
