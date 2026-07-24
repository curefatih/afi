package ir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeOpenAI marshals a ChatRequest to OpenAI chat.completions JSON (for hooks / legacy adapters).
func EncodeOpenAI(req ChatRequest) ([]byte, error) {
	msgs := make([]map[string]any, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, map[string]any{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		if m.Role == "system" && req.System != "" {
			// Already lifted to top-level system; skip duplicates when both present.
			continue
		}
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	out := map[string]any{
		"model":    req.Model,
		"messages": msgs,
	}
	if req.Stream {
		out["stream"] = true
	}
	if req.MaxTokens > 0 {
		out["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if len(req.Stop) > 0 {
		out["stop"] = req.Stop
	}
	return json.Marshal(out)
}

// DecodeOpenAIRequest parses OpenAI chat.completions JSON into ChatRequest.
func DecodeOpenAIRequest(body []byte) (ChatRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid openai chat body: %w", err)
	}
	if err := rejectOpenAIUnsupported(raw); err != nil {
		return ChatRequest{}, err
	}

	var in struct {
		Model               string   `json:"model"`
		Stream              bool     `json:"stream"`
		MaxTokens           *int     `json:"max_tokens"`
		MaxCompletionTokens *int     `json:"max_completion_tokens"`
		Temperature         *float64 `json:"temperature"`
		Stop                any      `json:"stop"`
		Messages            []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid openai chat body: %w", err)
	}
	if strings.TrimSpace(in.Model) == "" {
		return ChatRequest{}, fmt.Errorf("model is required")
	}
	req := ChatRequest{Model: in.Model, Stream: in.Stream, Temperature: in.Temperature}
	if in.MaxTokens != nil && *in.MaxTokens > 0 {
		req.MaxTokens = *in.MaxTokens
	} else if in.MaxCompletionTokens != nil && *in.MaxCompletionTokens > 0 {
		req.MaxTokens = *in.MaxCompletionTokens
	}
	req.Stop = normalizeStop(in.Stop)

	var systemParts []string
	for _, m := range in.Messages {
		text, err := textFromOpenAIContent(m.Content)
		if err != nil {
			return ChatRequest{}, err
		}
		switch m.Role {
		case "system":
			if text != "" {
				systemParts = append(systemParts, text)
			}
		case "user", "assistant":
			req.Messages = append(req.Messages, Message{Role: m.Role, Content: text})
		default:
			if text != "" {
				req.Messages = append(req.Messages, Message{Role: m.Role, Content: text})
			}
		}
	}
	if len(systemParts) > 0 {
		req.System = strings.Join(systemParts, "\n\n")
	}
	if len(req.Messages) == 0 {
		return ChatRequest{}, fmt.Errorf("at least one user/assistant message is required")
	}
	return req, nil
}

// EncodeOpenAIResponse marshals ChatResponse to OpenAI chat.completion JSON.
func EncodeOpenAIResponse(resp ChatResponse) ([]byte, error) {
	role := resp.Role
	if role == "" {
		role = "assistant"
	}
	out := map[string]any{
		"id":     resp.ID,
		"object": "chat.completion",
		"model":  resp.Model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]string{
					"role":    role,
					"content": resp.Content,
				},
				"finish_reason": resp.FinishReason,
			},
		},
		"usage": map[string]int64{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

// DecodeOpenAIResponse parses OpenAI chat.completion JSON into ChatResponse.
func DecodeOpenAIResponse(body []byte) (ChatResponse, error) {
	var in struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatResponse{}, fmt.Errorf("invalid openai chat response: %w", err)
	}
	if in.Error != nil {
		return ChatResponse{}, fmt.Errorf("openai: %s", in.Error.Message)
	}
	if len(in.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai: empty choices")
	}
	c := in.Choices[0]
	return ChatResponse{
		ID:           in.ID,
		Model:        in.Model,
		Role:         c.Message.Role,
		Content:      contentToString(c.Message.Content),
		FinishReason: c.FinishReason,
		Usage: Usage{
			PromptTokens:     in.Usage.PromptTokens,
			CompletionTokens: in.Usage.CompletionTokens,
		},
	}, nil
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

func normalizeStop(stop any) []string {
	switch v := stop.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
