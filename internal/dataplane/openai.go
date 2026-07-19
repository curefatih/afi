package dataplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/snapshot"
)

type OpenAIClient struct {
	HTTP *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		HTTP: &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *OpenAIClient) ChatCompletions(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	apiKey := os.Getenv(provider.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing env %s for provider %s", provider.APIKeyEnv, provider.ID)
	}

	base := strings.TrimRight(provider.BaseURL, "/")
	url := base + "/chat/completions"

	// Rewrite model to target model while preserving the rest of the body.
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	payload["model"] = targetModel
	if stream {
		payload["stream"] = true
	}
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CopyResponse copies an upstream response to the client writer.
func CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	for k, vals := range resp.Header {
		// Avoid hop-by-hop issues; pass content-type and others through.
		if strings.EqualFold(k, "Transfer-Encoding") || strings.EqualFold(k, "Connection") {
			continue
		}
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}
