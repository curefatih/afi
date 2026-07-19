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

const anthropicVersion = "2023-06-01"

type AnthropicClient struct {
	HTTP *http.Client
}

func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		HTTP: &http.Client{Timeout: 120 * time.Second},
	}
}

// Messages translates an OpenAI-shaped chat body to Anthropic /v1/messages
// and returns an OpenAI-shaped chat completion response.
func (c *AnthropicClient) Messages(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error) {
	apiKey := os.Getenv(provider.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing env %s for provider %s", provider.APIKeyEnv, provider.ID)
	}

	anthBody, err := openAIChatToAnthropic(body, targetModel)
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(provider.BaseURL, "/")
	url := base + "/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(anthBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return resp, nil
	}

	raw, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}

	mapped, err := anthropicToOpenAIChat(raw)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(mapped)),
	}, nil
}

func openAIChatToAnthropic(body []byte, targetModel string) ([]byte, error) {
	var in struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
		MaxTokens           *int `json:"max_tokens"`
		MaxCompletionTokens *int `json:"max_completion_tokens"`
		Temperature         any  `json:"temperature"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	var systemParts []string
	var messages []map[string]any
	for _, m := range in.Messages {
		text := contentToString(m.Content)
		switch m.Role {
		case "system":
			if text != "" {
				systemParts = append(systemParts, text)
			}
		case "user", "assistant":
			messages = append(messages, map[string]any{
				"role":    m.Role,
				"content": text,
			})
		}
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}

	maxTokens := 4096
	if in.MaxTokens != nil && *in.MaxTokens > 0 {
		maxTokens = *in.MaxTokens
	} else if in.MaxCompletionTokens != nil && *in.MaxCompletionTokens > 0 {
		maxTokens = *in.MaxCompletionTokens
	}

	out := map[string]any{
		"model":      targetModel,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if len(systemParts) > 0 {
		out["system"] = strings.Join(systemParts, "\n\n")
	}
	if in.Temperature != nil {
		out["temperature"] = in.Temperature
	}
	return json.Marshal(out)
}

func contentToString(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func anthropicToOpenAIChat(raw []byte) ([]byte, error) {
	var in struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, fmt.Errorf("invalid anthropic response: %w", err)
	}
	if in.Error != nil {
		return nil, fmt.Errorf("anthropic: %s", in.Error.Message)
	}

	var text strings.Builder
	for _, block := range in.Content {
		if block.Type == "text" || block.Type == "" {
			text.WriteString(block.Text)
		}
	}

	finish := "stop"
	switch in.StopReason {
	case "max_tokens":
		finish = "length"
	case "end_turn", "stop_sequence", "":
		finish = "stop"
	default:
		finish = in.StopReason
	}

	role := in.Role
	if role == "" {
		role = "assistant"
	}

	out := map[string]any{
		"id":      in.ID,
		"object":  "chat.completion",
		"model":   in.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]string{
					"role":    role,
					"content": text.String(),
				},
				"finish_reason": finish,
			},
		},
		"usage": map[string]int64{
			"prompt_tokens":     in.Usage.InputTokens,
			"completion_tokens": in.Usage.OutputTokens,
			"total_tokens":      in.Usage.InputTokens + in.Usage.OutputTokens,
		},
	}
	return json.Marshal(out)
}
