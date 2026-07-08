package types

import "encoding/json"

// ChatCompletionRequest is an OpenAI-compatible chat completion payload.
// This will be used as internal representation of the chat completion request.
// Providers will be responsible for converting this to their own request format.
type ChatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []ChatMessage   `json:"messages"`
	Tools          json.RawMessage `json:"tools,omitempty"`
	ResponseFormat json.RawMessage `json:"response_format,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	TopP           *float64        `json:"top_p,omitempty"`
	Stop           json.RawMessage `json:"stop,omitempty"`
	User           string          `json:"user,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string          `json:"name,omitempty"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	} `json:"function,omitempty"`
}

type ChatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// MessageText extracts plain text from a message content field.
func MessageText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(content, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &parts); err == nil {
		var out string
		for _, p := range parts {
			if p.Type == "text" {
				out += p.Text
			}
		}
		return out
	}
	return string(content)
}
