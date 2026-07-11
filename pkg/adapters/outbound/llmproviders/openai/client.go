package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/curefatih/afi/internal/core/domain"
	"github.com/curefatih/afi/internal/ports"
)

type OpenAIAdapter struct {
	vault      ports.CredentialVault
	httpClient *http.Client
	baseURL    string
}

func NewAdapter(vault ports.CredentialVault, httpClient *http.Client) *OpenAIAdapter {
	return &OpenAIAdapter{
		vault:      vault,
		httpClient: httpClient,
		baseURL:    "https://api.openai.com/v1",
	}
}

func (a *OpenAIAdapter) Call(ctx context.Context, req *domain.InternalRequest) (*domain.InternalResponse, error) {
	vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "openai")
	if err != nil {
		return nil, fmt.Errorf("openai authorization key resolution failed: %w", err)
	}

	openAIReq := mapToOpenAIRequest(req)
	jsonBytes, _ := json.Marshal(openAIReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	a.setHeaders(httpReq, vendorKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai upstream error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var vendorResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&vendorResp); err != nil {
		return nil, fmt.Errorf("failed to decode openai response: %w", err)
	}

	return mapToInternalResponse(&vendorResp), nil
}

func (a *OpenAIAdapter) StreamCall(ctx context.Context, req *domain.InternalRequest) (<-chan domain.StreamChunk, <-chan error) {
	outChunks := make(chan domain.StreamChunk)
	outErr := make(chan error, 1)

	vendorKey, err := a.vault.GetProviderKey(ctx, req.Metadata.ProjectID, "openai")
	if err != nil {
		outErr <- fmt.Errorf("openai streaming key resolution failed: %w", err)
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	openAIReq := mapToOpenAIRequest(req)
	openAIReq.Stream = true
	// Request usage details in the stream to calculate budget precisely at the end
	openAIReq.StreamOptions = &StreamOptions{IncludeUsage: true}

	jsonBytes, _ := json.Marshal(openAIReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/chat/completions", bytes.NewBuffer(jsonBytes))
	if err != nil {
		outErr <- err
		close(outChunks)
		close(outErr)
		return outChunks, outErr
	}

	a.setHeaders(httpReq, vendorKey)

	go func() {
		defer close(outChunks)
		defer close(outErr)

		resp, err := a.httpClient.Do(httpReq)
		if err != nil {
			outErr <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			outErr <- fmt.Errorf("openai upstream stream error status %d: %s", resp.StatusCode, string(bodyBytes))
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				outErr <- ctx.Err()
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						outErr <- err
					}
					return
				}

				line = strings.TrimSpace(line)
				if line == "" || !strings.HasPrefix(line, "data: ") {
					continue
				}

				dataStr := strings.TrimPrefix(line, "data: ")
				if dataStr == "[DONE]" {
					return
				}

				var chunk VendorStreamChunk
				if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
					continue
				}

				var textDelta string
				if len(chunk.Choices) > 0 {
					textDelta = chunk.Choices[0].Delta.Content
				}

				// Safely extract the usage block if it's included in this frame
				var domainUsage domain.TokenUsage
				if chunk.Usage.TotalTokens > 0 {
					domainUsage = domain.TokenUsage{
						InputTokens:  chunk.Usage.PromptTokens,
						OutputTokens: chunk.Usage.CompletionTokens,
						TotalTokens:  chunk.Usage.TotalTokens,
					}
					// Temporarily keep this print to confirm the adapter caught it over the wire!
					fmt.Printf("[ADAPTER TRACE] Extracted usage from stream chunk: %d total tokens\n", domainUsage.TotalTokens)
				}

				outChunks <- domain.StreamChunk{
					ID:        chunk.ID,
					CreatedAt: chunk.Created,
					Model:     chunk.Model,
					DeltaText: textDelta,
					Usage:     domainUsage, // Pass the cleanly parsed struct down the channel
				}
			}
		}
	}()

	return outChunks, outErr
}

func (a *OpenAIAdapter) setHeaders(req *http.Request, key string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
}
