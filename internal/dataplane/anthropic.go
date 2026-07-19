package dataplane

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/curefatih/afi/internal/dataplane/openaichat"
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

// PassThrough forwards an Anthropic-shaped /v1/messages body to upstream,
// rewriting only the model field to targetModel.
func (c *AnthropicClient) PassThrough(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	apiKey := os.Getenv(provider.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing env %s for provider %s", provider.APIKeyEnv, provider.ID)
	}

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

	base := strings.TrimRight(provider.BaseURL, "/")
	url := base + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rewritten))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
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

// Messages translates an OpenAI-shaped chat body to Anthropic /v1/messages
// and returns an OpenAI-shaped chat completion response (JSON or SSE).
func (c *AnthropicClient) Messages(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error) {
	apiKey := os.Getenv(provider.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("missing env %s for provider %s", provider.APIKeyEnv, provider.ID)
	}

	anthBody, err := openAIChatToAnthropic(body, targetModel, stream)
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
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return resp, nil
	}

	if stream {
		pr, pw := io.Pipe()
		go func() {
			defer resp.Body.Close()
			err := translateAnthropicSSE(resp.Body, pw)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"text/event-stream"},
				"Cache-Control": []string{"no-cache"},
			},
			Body: pr,
		}, nil
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

func openAIChatToAnthropic(body []byte, targetModel string, stream bool) ([]byte, error) {
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
		text := openaichat.ContentToString(m.Content)
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
	if stream {
		out["stream"] = true
	}
	if len(systemParts) > 0 {
		out["system"] = strings.Join(systemParts, "\n\n")
	}
	if in.Temperature != nil {
		out["temperature"] = in.Temperature
	}
	return json.Marshal(out)
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

	finish := mapAnthropicStopReason(in.StopReason)
	role := in.Role
	if role == "" {
		role = "assistant"
	}

	out := map[string]any{
		"id":     in.ID,
		"object": "chat.completion",
		"model":  in.Model,
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

func mapAnthropicStopReason(r string) string {
	switch r {
	case "max_tokens":
		return "length"
	case "end_turn", "stop_sequence", "":
		return "stop"
	default:
		return r
	}
}

// translateAnthropicSSE reads Anthropic event-stream and writes OpenAI chat.completion.chunk SSE.
func translateAnthropicSSE(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		msgID   = "chatcmpl-anthropic"
		model   string
		started bool
	)

	writeChunk := func(delta map[string]any, finish any) error {
		return openaichat.WriteSSEChunk(w, msgID, model, delta, finish)
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(payload), &raw); err != nil {
			continue
		}
		evType, _ := raw["type"].(string)

		switch evType {
		case "message_start":
			if msg, ok := raw["message"].(map[string]any); ok {
				if id, ok := msg["id"].(string); ok && id != "" {
					msgID = id
				}
				if m, ok := msg["model"].(string); ok && m != "" {
					model = m
				}
			}
			if !started {
				started = true
				if err := writeChunk(map[string]any{"role": "assistant"}, nil); err != nil {
					return err
				}
			}
		case "content_block_start":
			if block, ok := raw["content_block"].(map[string]any); ok {
				if typ, _ := block["type"].(string); typ == "text" {
					if text, ok := block["text"].(string); ok && text != "" {
						if err := writeChunk(map[string]any{"content": text}, nil); err != nil {
							return err
						}
					}
				}
			}
		case "content_block_delta":
			if delta, ok := raw["delta"].(map[string]any); ok {
				text, _ := delta["text"].(string)
				if text != "" {
					if err := writeChunk(map[string]any{"content": text}, nil); err != nil {
						return err
					}
				}
			}
		case "message_delta":
			if delta, ok := raw["delta"].(map[string]any); ok {
				if sr, ok := delta["stop_reason"].(string); ok && sr != "" {
					if err := writeChunk(map[string]any{}, mapAnthropicStopReason(sr)); err != nil {
						return err
					}
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return openaichat.WriteSSEDone(w)
}
