package dataplane

import (
	"context"
	"fmt"
	"net/http"

	"github.com/curefatih/afi/internal/dataplane/ir"
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
	return ProviderCaps{Chat: c.Chat, Stream: c.Stream, TTS: c.TTS, STT: c.STT, Embedding: c.Embedding}
}

func (b *sdkChatBridge) Chat(ctx context.Context, p snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	_ = ctx
	_ = p
	_ = targetModel
	_ = body
	_ = stream
	return nil, fmt.Errorf("sdk provider %q does not support OpenAI-byte Chat; use ChatIR", b.Type())
}

func (b *sdkChatBridge) ChatIR(ctx context.Context, p snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	return b.inner.ChatIR(ctx, sdkConfig(p), targetModel, req)
}

func sdkConfig(p snapshot.Provider) sdkprovider.ProviderConfig {
	caps := snapshot.NormalizeCapabilities(p.Type, p.Capabilities)
	return sdkprovider.ConfigFromFields(
		p.ID, p.Type, p.Name, p.BaseURL, p.APIKeyEnv,
		sdkprovider.Capabilities{
			Chat: caps.Chat, Stream: caps.Stream, TTS: caps.TTS, STT: caps.STT, Embedding: caps.Embedding,
		},
	)
}

var (
	_ ChatProvider   = (*sdkChatBridge)(nil)
	_ IRChatProvider = (*sdkChatBridge)(nil)
)
