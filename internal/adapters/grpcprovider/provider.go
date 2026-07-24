package grpcprovider

import (
	"context"
	"fmt"
	"strings"
	"time"

	extensionv1 "github.com/curefatih/afi/gen/proto/afi/extension/v1"
	"github.com/curefatih/afi/sdk/chatir"
	sdkprovider "github.com/curefatih/afi/sdk/provider"
)

// ProviderAdapter implements sdk/provider.ChatProvider over gRPC ChatIR.
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
		caps:    sdkprovider.Capabilities{Chat: true, Stream: true},
	}
}

func (a *ProviderAdapter) Type() string { return a.typ }

func (a *ProviderAdapter) Capabilities() sdkprovider.Capabilities {
	if a == nil {
		return sdkprovider.Capabilities{}
	}
	return a.caps
}

func (a *ProviderAdapter) ChatIR(ctx context.Context, cfg sdkprovider.ProviderConfig, targetModel string, req chatir.Request) (chatir.Result, error) {
	if a == nil || a.client == nil {
		return chatir.Result{}, fmt.Errorf("grpc provider %q: nil client", a.typ)
	}
	pbReq := &extensionv1.ChatIRRequest{
		Config:      providerConfigProto(cfg),
		TargetModel: targetModel,
		Request:     ChatIRRequestProto(req),
	}
	// Streams outlive this call, so they own their context instead of the unary timeout.
	if req.Stream {
		return a.chatIRStream(ctx, pbReq)
	}
	cctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	resp, err := a.client.ChatIR(cctx, pbReq)
	if err != nil {
		return chatir.Result{}, fmt.Errorf("grpc provider %q ChatIR: %w", a.typ, err)
	}
	return chatIRResultFromProto(resp), nil
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
