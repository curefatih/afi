package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type GeminiAdapter struct {
	vault      ports.CredentialVault
	httpClient *http.Client
}

func NewAdapter(vault ports.CredentialVault, httpClient *http.Client) *GeminiAdapter {
	return &GeminiAdapter{
		vault:      vault,
		httpClient: httpClient,
	}
}

func (a *GeminiAdapter) Call(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error) {
	vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "gemini")
	if err != nil {
		return nil, fmt.Errorf("gemini credentials retrieval breakdown: %w", err)
	}

	payload := mapToGemini(req)
	jsonBytes, _ := json.Marshal(payload)

	// API target URL using Gemini v1beta model pathways
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", req.Model, vendorKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini upstream returned invalid status code %d", resp.StatusCode)
	}

	return mapToInternalResponse(resp, req.Model)
}

func (a *GeminiAdapter) StreamCall(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error) {
	ch := make(chan domain.StreamChunk)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "gemini")
		if err != nil {
			errCh <- fmt.Errorf("gemini credentials retrieval breakdown: %w", err)
			return
		}

		payload := mapToGemini(req)
		jsonBytes, _ := json.Marshal(payload)

		// streaming API call
		apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", req.Model, vendorKey)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBytes))
		if err != nil {
			errCh <- err
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		respStream, err := a.httpClient.Do(httpReq)
		if err != nil {
			errCh <- err
			return
		}
		defer respStream.Body.Close()

		if respStream.StatusCode != http.StatusOK {
			errCh <- fmt.Errorf("gemini upstream stream error status %d", respStream.StatusCode)
			return
		}

		// Read Server-Sent Events (SSE) line-by-line using a scanner
		scanner := bufio.NewScanner(respStream.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			// Slice "data: " out to extract raw JSON
			dataBytes := []byte(strings.TrimPrefix(line, "data: "))

			var chunkResp GeminiResponse
			if err := json.Unmarshal(dataBytes, &chunkResp); err != nil {
				continue
			}

			// Extract delta text chunks securely
			if len(chunkResp.Candidates) > 0 && len(chunkResp.Candidates[0].Content.Parts) > 0 {
				deltaText := chunkResp.Candidates[0].Content.Parts[0].Text
				if deltaText != "" {
					ch <- domain.StreamChunk{
						DeltaText: deltaText,
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("gemini stream scanning exception: %w", err)
		}
	}()

	return ch, errCh
}
