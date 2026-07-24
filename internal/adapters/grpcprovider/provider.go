package grpcprovider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

// ProviderAdapter implements sdk/provider.ChatProvider over gRPC.
type ProviderAdapter struct {
	client  extensionv1.ProviderClient
	typ     string
	timeout time.Duration
	caps    sdkprovider.Capabilities
}

func newProviderAdapter(client extensionv1.ProviderClient, typ string, timeout time.Duration) *ProviderAdapter {
	return &ProviderAdapter{
		client:  client,
		typ:     typ,
		timeout: timeout,
		caps:    sdkprovider.Capabilities{Chat: true, Stream: false},
	}
}

func (a *ProviderAdapter) Type() string { return a.typ }

func (a *ProviderAdapter) Capabilities() sdkprovider.Capabilities {
	if a == nil {
		return sdkprovider.Capabilities{}
	}
	return a.caps
}

func (a *ProviderAdapter) Chat(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, body []byte, stream bool) (*http.Response, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("grpc provider %q: nil client", a.typ)
	}
	if stream {
		return nil, fmt.Errorf("streaming is not supported for grpc provider type %q", a.typ)
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	resp, err := a.client.Chat(cctx, &extensionv1.ChatRequest{
		Config: &extensionv1.ProviderConfig{
			Id:        cfg.ID,
			Type:      cfg.Type,
			BaseUrl:   cfg.BaseURL,
			ApiKeyEnv: cfg.APIKeyEnv,
			Name:      cfg.Name,
			Capabilities: &extensionv1.Capabilities{
				Chat:      cfg.Capabilities.Chat,
				Stream:    cfg.Capabilities.Stream,
				Tts:       cfg.Capabilities.TTS,
				Stt:       cfg.Capabilities.STT,
				Embedding: cfg.Capabilities.Embedding,
			},
		},
		TargetModel: targetModel,
		Body:        body,
		Stream:      stream,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc provider %q Chat: %w", a.typ, err)
	}
	status := int(resp.GetStatusCode())
	if status == 0 {
		status = http.StatusOK
	}
	hdr := make(http.Header)
	for k, v := range resp.GetHeaders() {
		if k == "" {
			continue
		}
		hdr.Set(k, v)
	}
	if hdr.Get("Content-Type") == "" {
		hdr.Set("Content-Type", "application/json")
	}
	return &http.Response{
		StatusCode: status,
		Header:     hdr,
		Body:       io.NopCloser(bytes.NewReader(resp.GetBody())),
	}, nil
}

var _ sdkprovider.ChatProvider = (*ProviderAdapter)(nil)

func resolveProviderType(manifest Manifest, hs *extensionv1.HandshakeResponse) (string, error) {
	if t := strings.TrimSpace(manifest.ProviderType); t != "" {
		return t, nil
	}
	if hs != nil {
		if t := strings.TrimSpace(hs.GetProviderType()); t != "" {
			return t, nil
		}
	}
	return "", fmt.Errorf("grpc extension %q: provider_type missing (set manifest.provider_type or Handshake.provider_type)", manifest.ID)
}
