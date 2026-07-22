package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/snapshot"
)

type OpenAIClient struct {
	HTTP    *http.Client
	Secrets secrets.Resolver
}

func NewOpenAIClient(sec secrets.Resolver) *OpenAIClient {
	if sec == nil {
		sec = secrets.Default()
	}
	return &OpenAIClient{
		HTTP:    &http.Client{Timeout: 120 * time.Second},
		Secrets: sec,
	}
}

func (c *OpenAIClient) apiKey(ctx context.Context, provider snapshot.Provider) (string, error) {
	if provider.InlineAPIKey != "" {
		return provider.InlineAPIKey, nil
	}
	key, err := c.Secrets.Get(ctx, provider.APIKeyEnv)
	if err != nil {
		return "", fmt.Errorf("%w for provider %s", err, provider.ID)
	}
	return key, nil
}

func (c *OpenAIClient) ChatCompletions(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(provider.BaseURL, "/")
	url := base + "/chat/completions"

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	payload["model"] = targetModel
	if stream {
		payload["stream"] = true
		// Ask upstream for a final usage chunk so the gateway can price streams.
		opts, _ := payload["stream_options"].(map[string]any)
		if opts == nil {
			opts = map[string]any{}
		}
		opts["include_usage"] = true
		payload["stream_options"] = opts
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
	applyExtraHeaders(ctx, req)

	return c.HTTP.Do(req)
}

func (c *OpenAIClient) AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	payload["model"] = targetModel
	rewritten, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(provider.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/speech", bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	applyExtraHeaders(ctx, req)
	return c.HTTP.Do(req)
}

func (c *OpenAIClient) AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	rewritten, newCT, err := rewriteMultipartModel(contentType, body, targetModel)
	if err != nil {
		return nil, err
	}
	base := strings.TrimRight(provider.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/audio/transcriptions", bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", newCT)
	applyExtraHeaders(ctx, req)
	return c.HTTP.Do(req)
}
