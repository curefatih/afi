package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/adapters/secrets"
	"github.com/curefatih/afi/internal/dataplane/dialect"
	"github.com/curefatih/afi/internal/dataplane/ir"
	"github.com/curefatih/afi/internal/snapshot"
	"github.com/curefatih/afi/internal/telemetry"
)

const (
	azureAPIStyleDeployments = "deployments"
	azureAPIStyleOpenAIV1    = "openai_v1"
	azureDefaultAPIVersion   = "2024-10-21"
)

// AzureOpenAIClient talks to Azure OpenAI using either the classic deployments
// API or the OpenAI v1-compatible path layout, selected via provider config.
type AzureOpenAIClient struct {
	HTTP    *http.Client
	Secrets secrets.Resolver
}

func NewAzureOpenAIClient(sec secrets.Resolver) *AzureOpenAIClient {
	if sec == nil {
		sec = secrets.Default()
	}
	return &AzureOpenAIClient{
		HTTP:    &http.Client{Timeout: 120 * time.Second, Transport: telemetry.WrapTransport(http.DefaultTransport)},
		Secrets: sec,
	}
}

type azureProviderConfig struct {
	APIStyle   string `json:"api_style"`
	APIVersion string `json:"api_version"`
}

func parseAzureConfig(raw json.RawMessage) (azureProviderConfig, error) {
	cfg := azureProviderConfig{APIStyle: azureAPIStyleDeployments}
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return azureProviderConfig{}, fmt.Errorf("invalid azure_openai config: %w", err)
	}
	cfg.APIStyle = strings.ToLower(strings.TrimSpace(cfg.APIStyle))
	if cfg.APIStyle == "" {
		cfg.APIStyle = azureAPIStyleDeployments
	}
	switch cfg.APIStyle {
	case azureAPIStyleDeployments, azureAPIStyleOpenAIV1:
	default:
		return azureProviderConfig{}, fmt.Errorf("unsupported azure_openai api_style %q (want deployments or openai_v1)", cfg.APIStyle)
	}
	cfg.APIVersion = strings.TrimSpace(cfg.APIVersion)
	return cfg, nil
}

func (c *AzureOpenAIClient) apiKey(ctx context.Context, provider snapshot.Provider) (string, error) {
	if provider.InlineAPIKey != "" {
		return provider.InlineAPIKey, nil
	}
	key, err := c.Secrets.Get(ctx, provider.APIKeyEnv)
	if err != nil {
		return "", fmt.Errorf("%w for provider %s", err, provider.ID)
	}
	return key, nil
}

// azureEndpointURL builds the upstream URL for a modality path segment
// (e.g. "chat/completions", "embeddings", "audio/speech").
func azureEndpointURL(provider snapshot.Provider, deployment, endpoint string) (string, error) {
	cfg, err := parseAzureConfig(provider.Config)
	if err != nil {
		return "", err
	}
	base := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("azure_openai base_url is required")
	}
	deployment = strings.TrimSpace(deployment)
	endpoint = strings.Trim(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return "", fmt.Errorf("azure_openai endpoint is required")
	}

	var raw string
	switch cfg.APIStyle {
	case azureAPIStyleDeployments:
		if deployment == "" {
			return "", fmt.Errorf("azure_openai deployment (target model) is required for deployments api_style")
		}
		ver := cfg.APIVersion
		if ver == "" {
			ver = azureDefaultAPIVersion
		}
		raw = fmt.Sprintf("%s/openai/deployments/%s/%s", base, url.PathEscape(deployment), endpoint)
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		q := u.Query()
		q.Set("api-version", ver)
		u.RawQuery = q.Encode()
		return u.String(), nil
	case azureAPIStyleOpenAIV1:
		raw = base + "/" + endpoint
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		if cfg.APIVersion != "" {
			q := u.Query()
			q.Set("api-version", cfg.APIVersion)
			u.RawQuery = q.Encode()
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported azure_openai api_style %q", cfg.APIStyle)
	}
}

func (c *AzureOpenAIClient) doJSON(ctx context.Context, provider snapshot.Provider, targetModel, endpoint string, body []byte, stream bool, acceptSSE bool) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	payload["model"] = targetModel
	if stream {
		payload["stream"] = true
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
	urlStr, err := azureEndpointURL(provider, targetModel, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	if acceptSSE {
		req.Header.Set("Accept", "text/event-stream")
	}
	applyExtraHeaders(ctx, req)
	return c.HTTP.Do(req)
}

func (c *AzureOpenAIClient) ChatCompletions(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	return c.doJSON(ctx, provider, targetModel, "chat/completions", body, stream, stream)
}

func (c *AzureOpenAIClient) AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	return c.doJSON(ctx, provider, targetModel, "audio/speech", body, false, false)
}

func (c *AzureOpenAIClient) Embeddings(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	return c.doJSON(ctx, provider, targetModel, "embeddings", body, false, false)
}

func (c *AzureOpenAIClient) Images(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	return c.doJSON(ctx, provider, targetModel, "images/generations", body, false, false)
}

func (c *AzureOpenAIClient) AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error) {
	apiKey, err := c.apiKey(ctx, provider)
	if err != nil {
		return nil, err
	}
	rewritten, newCT, err := rewriteMultipartModel(contentType, body, targetModel)
	if err != nil {
		return nil, err
	}
	urlStr, err := azureEndpointURL(provider, targetModel, "audio/transcriptions")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", newCT)
	applyExtraHeaders(ctx, req)
	return c.HTTP.Do(req)
}

// ChatIR runs chat against Azure OpenAI using OpenAI-shaped IR encode/decode.
func (c *AzureOpenAIClient) ChatIR(ctx context.Context, provider snapshot.Provider, targetModel string, req ir.ChatRequest) (ir.ChatResult, error) {
	body, err := ir.EncodeOpenAI(req)
	if err != nil {
		return ir.ChatResult{}, err
	}
	resp, err := c.ChatCompletions(ctx, provider, targetModel, body, req.Stream)
	if err != nil {
		return ir.ChatResult{}, err
	}
	return mapHTTPToChatResult(resp, req.Stream, ir.DecodeOpenAIResponse, dialect.ParseOpenAISSE)
}
