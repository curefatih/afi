package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type AnthropicAdapter struct {
	vault      ports.CredentialVault // Injected boundary port
	httpClient *http.Client
}

func NewAdapter(vault ports.CredentialVault, httpClient *http.Client) *AnthropicAdapter {
	return &AnthropicAdapter{
		vault:      vault,
		httpClient: httpClient,
	}
}

func (a *AnthropicAdapter) Call(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error) {
	// 1. Intercept the request context metadata and pull the live key dynamically
	vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "anthropic")
	if err != nil {
		return nil, fmt.Errorf("anthropic credentials retrieval breakdown: %w", err)
	}

	// 2. Map internal representation to vendor-specific body
	anthropicPayload := mapToAnthropic(req)
	jsonBytes, _ := json.Marshal(anthropicPayload)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	// 3. Inject the resolved user key directly into headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", vendorKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// 4. Perform network request execution
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 5. Parse and return your internal domain mapping response...
	return mapToInternalResponse(resp)
}

func (a *AnthropicAdapter) StreamCall(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error) {
	ch := make(chan domain.StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "anthropic")
		if err != nil {
			errCh <- fmt.Errorf("anthropic credentials retrieval breakdown: %w", err)
			return
		}

		anthropicReq := mapToAnthropicStreamReq(req)
		jsonBytes, _ := json.Marshal(anthropicReq)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBytes))
		if err != nil {
			errCh <- err
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", vendorKey)
		httpReq.Header.Set("anthropic-version", "2023-06-01")

		respStream, err := a.httpClient.Do(httpReq)
		if err != nil {
			errCh <- err
			return
		}
		defer respStream.Body.Close()

		if respStream.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("anthropic upstream stream error status %d", respStream.StatusCode)
			return
		}

		for {
			var event map[string]any
			if err := json.NewDecoder(respStream.Body).Decode(&event); err != nil {
				return
			}
			if event["type"] == "message" {
				if deltaText, ok := event["delta_text"].(string); ok {
					chunk := domain.StreamChunk{
						DeltaText: deltaText,
					}
					ch <- chunk
				}
			}
		}
	}()

	return ch, errCh
}
