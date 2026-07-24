package ir

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeAnthropic marshals a ChatRequest to Anthropic messages JSON.
func EncodeAnthropic(req ChatRequest, targetModel string) ([]byte, error) {
	model := targetModel
	if model == "" {
		model = req.Model
	}
	messages := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case "user", "assistant":
			messages = append(messages, map[string]any{
				"role":    m.Role,
				"content": m.Content,
			})
		case "system":
			// Prefer top-level system; fold extras into system string below.
		}
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one user/assistant message is required")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	out := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if req.Stream {
		out["stream"] = true
	}
	sys := req.System
	if sys == "" {
		var parts []string
		for _, m := range req.Messages {
			if m.Role == "system" && m.Content != "" {
				parts = append(parts, m.Content)
			}
		}
		sys = strings.Join(parts, "\n\n")
	}
	if sys != "" {
		out["system"] = sys
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if len(req.Stop) > 0 {
		out["stop_sequences"] = req.Stop
	}
	return json.Marshal(out)
}

// DecodeAnthropicRequest parses Anthropic messages JSON into ChatRequest.
func DecodeAnthropicRequest(body []byte) (ChatRequest, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid anthropic messages body: %w", err)
	}
	if err := rejectAnthropicUnsupported(raw); err != nil {
		return ChatRequest{}, err
	}

	var in struct {
		Model       string   `json:"model"`
		Stream      bool     `json:"stream"`
		MaxTokens   int      `json:"max_tokens"`
		Temperature *float64 `json:"temperature"`
		System      any      `json:"system"`
		Stop        []string `json:"stop_sequences"`
		Messages    []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatRequest{}, fmt.Errorf("invalid anthropic messages body: %w", err)
	}
	if strings.TrimSpace(in.Model) == "" {
		return ChatRequest{}, fmt.Errorf("model is required")
	}
	req := ChatRequest{
		Model:       in.Model,
		Stream:      in.Stream,
		MaxTokens:   in.MaxTokens,
		Temperature: in.Temperature,
		Stop:        in.Stop,
		System:      anthropicSystemToString(in.System),
	}
	for _, m := range in.Messages {
		text, err := textFromAnthropicContent(m.Content)
		if err != nil {
			return ChatRequest{}, err
		}
		switch m.Role {
		case "user", "assistant":
			req.Messages = append(req.Messages, Message{Role: m.Role, Content: text})
		}
	}
	if len(req.Messages) == 0 {
		return ChatRequest{}, fmt.Errorf("at least one user/assistant message is required")
	}
	return req, nil
}

// EncodeAnthropicResponse marshals ChatResponse to Anthropic message JSON.
func EncodeAnthropicResponse(resp ChatResponse) ([]byte, error) {
	role := resp.Role
	if role == "" {
		role = "assistant"
	}
	out := map[string]any{
		"id":    resp.ID,
		"type":  "message",
		"role":  role,
		"model": resp.Model,
		"content": []map[string]string{
			{"type": "text", "text": resp.Content},
		},
		"stop_reason": mapFinishToAnthropic(resp.FinishReason),
		"usage": map[string]int64{
			"input_tokens":  resp.Usage.PromptTokens,
			"output_tokens": resp.Usage.CompletionTokens,
		},
	}
	return json.Marshal(out)
}

// DecodeAnthropicResponse parses Anthropic message JSON into ChatResponse.
func DecodeAnthropicResponse(body []byte) (ChatResponse, error) {
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
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return ChatResponse{}, fmt.Errorf("invalid anthropic response: %w", err)
	}
	if in.Error != nil {
		return ChatResponse{}, fmt.Errorf("anthropic: %s", in.Error.Message)
	}
	var text strings.Builder
	for _, block := range in.Content {
		if block.Type == "text" || block.Type == "" {
			text.WriteString(block.Text)
		}
	}
	role := in.Role
	if role == "" {
		role = "assistant"
	}
	return ChatResponse{
		ID:           in.ID,
		Model:        in.Model,
		Role:         role,
		Content:      text.String(),
		FinishReason: MapAnthropicStopReason(in.StopReason),
		Usage: Usage{
			PromptTokens:     in.Usage.InputTokens,
			CompletionTokens: in.Usage.OutputTokens,
		},
	}, nil
}

// MapAnthropicStopReason maps Anthropic stop_reason to IR finish_reason.
func MapAnthropicStopReason(r string) string {
	switch r {
	case "max_tokens":
		return "length"
	case "end_turn", "stop_sequence", "":
		return "stop"
	default:
		return r
	}
}

func mapFinishToAnthropic(r string) string {
	switch r {
	case "length":
		return "max_tokens"
	case "stop", "":
		return "end_turn"
	default:
		return r
	}
}

func anthropicSystemToString(system any) string {
	switch v := system.(type) {
	case string:
		return v
	case nil:
		return ""
	case []any:
		var parts []string
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			if t, ok := bm["text"].(string); ok && t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n\n")
	default:
		return contentToString(v)
	}
}
