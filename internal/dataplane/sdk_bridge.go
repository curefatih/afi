package dataplane

import (
	"context"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/sdk/chatir"
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
	return b.inner.Chat(ctx, sdkConfig(p), targetModel, body, stream)
}

func (b *sdkChatBridge) ChatIR(ctx context.Context, p snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	if irp, ok := b.inner.(sdkprovider.ChatIRProvider); ok {
		return irp.ChatIR(ctx, sdkConfig(p), targetModel, req)
	}
	// Legacy SDK / gRPC adapters: OpenAI-shaped bytes bridge.
	body, err := ir.EncodeOpenAI(req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	resp, err := b.inner.Chat(ctx, sdkConfig(p), targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, ErrorBody: raw}, nil
	}
	if req.Stream {
		ch := dialect.ParseOpenAISSE(resp.Body)
		out := make(chan chatir.StreamEvent, 16)
		go func() {
			defer close(out)
			defer resp.Body.Close()
			for ev := range ch {
				out <- ev
			}
		}()
		return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Events: out}, nil
	}
	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return ir.ChatResult{}, err
	}
	mapped, err := ir.DecodeOpenAIResponse(raw)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return ir.ChatResult{StatusCode: resp.StatusCode, Header: resp.Header, Response: &mapped}, nil
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
