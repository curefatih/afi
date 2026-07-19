package dataplane

import (
	"context"
	"net/http"

	"github.com/curefatih/afi/internal/snapshot"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

// RegisterSDK wraps an SDK ChatProvider into the gateway registry.
func (r *Registry) RegisterSDK(p sdkprovider.ChatProvider) *Registry {
	return r.Register(&sdkChatBridge{inner: p})
}

type sdkChatBridge struct {
	inner sdkprovider.ChatProvider
}

func (b *sdkChatBridge) Type() string { return b.inner.Type() }

func (b *sdkChatBridge) Capabilities() ProviderCaps {
	c := b.inner.Capabilities()
	return ProviderCaps{Chat: c.Chat, Stream: c.Stream}
}

func (b *sdkChatBridge) Chat(ctx context.Context, p snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	caps := snapshot.NormalizeCapabilities(p.Type, p.Capabilities)
	cfg := sdkprovider.ConfigFromFields(
		p.ID, p.Type, p.Name, p.BaseURL, p.APIKeyEnv,
		sdkprovider.Capabilities{
			Chat: caps.Chat, Stream: caps.Stream, TTS: caps.TTS, STT: caps.STT,
		},
	)
	return b.inner.Chat(ctx, cfg, targetModel, body, stream)
}
