package ir

import (
	"fmt"
)

// rejectOpenAIUnsupported returns an error if the raw OpenAI chat body uses unsupported features.
func rejectOpenAIUnsupported(raw map[string]any) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw["tools"]; ok && !isJSONNull(v) {
		return Unsupported("tools", "tools are not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["tool_choice"]; ok && !isJSONNull(v) {
		return Unsupported("tool_choice", "tool_choice is not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["functions"]; ok && !isJSONNull(v) {
		return Unsupported("functions", "functions are not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["function_call"]; ok && !isJSONNull(v) {
		return Unsupported("function_call", "function_call is not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["logprobs"]; ok {
		switch t := v.(type) {
		case bool:
			if t {
				return Unsupported("logprobs", "logprobs are not supported by the gateway chat dialect yet")
			}
		case float64:
			if t != 0 {
				return Unsupported("logprobs", "logprobs are not supported by the gateway chat dialect yet")
			}
		default:
			if !isJSONNull(v) {
				return Unsupported("logprobs", "logprobs are not supported by the gateway chat dialect yet")
			}
		}
	}
	if v, ok := raw["n"]; ok {
		switch t := v.(type) {
		case float64:
			if t > 1 {
				return Unsupported("n", "n > 1 is not supported by the gateway chat dialect yet")
			}
		}
	}
	msgs, _ := raw["messages"].([]any)
	for _, item := range msgs {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		role, _ := m["role"].(string)
		if role == "tool" || role == "function" {
			return Unsupported("tool_messages", "tool/function messages are not supported by the gateway chat dialect yet")
		}
		if _, ok := m["tool_calls"]; ok {
			return Unsupported("tool_calls", "tool_calls are not supported by the gateway chat dialect yet")
		}
		if _, ok := m["function_call"]; ok {
			return Unsupported("function_call", "function_call message fields are not supported by the gateway chat dialect yet")
		}
		if err := rejectOpenAIContent(m["content"]); err != nil {
			return err
		}
	}
	return nil
}

func rejectOpenAIContent(content any) error {
	switch v := content.(type) {
	case nil, string:
		return nil
	case []any:
		for _, part := range v {
			pm, _ := part.(map[string]any)
			if pm == nil {
				continue
			}
			typ, _ := pm["type"].(string)
			switch typ {
			case "", "text":
				continue
			case "image_url":
				return Unsupported("vision", "image_url content is not supported by the gateway chat dialect yet")
			default:
				return Unsupported("content_parts", fmt.Sprintf("content part type %q is not supported by the gateway chat dialect yet", typ))
			}
		}
		return nil
	default:
		// Non-string, non-array content is unexpected; reject rather than corrupt.
		return Unsupported("content", "message content must be a string or text content parts")
	}
}

// rejectAnthropicUnsupported returns an error if the raw Anthropic messages body uses unsupported features.
func rejectAnthropicUnsupported(raw map[string]any) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw["tools"]; ok && !isJSONNull(v) {
		return Unsupported("tools", "tools are not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["tool_choice"]; ok && !isJSONNull(v) {
		return Unsupported("tool_choice", "tool_choice is not supported by the gateway chat dialect yet")
	}
	if v, ok := raw["thinking"]; ok && !isJSONNull(v) {
		return Unsupported("thinking", "thinking is not supported by the gateway chat dialect yet")
	}
	msgs, _ := raw["messages"].([]any)
	for _, item := range msgs {
		m, _ := item.(map[string]any)
		if m == nil {
			continue
		}
		if err := rejectAnthropicContent(m["content"]); err != nil {
			return err
		}
	}
	return nil
}

func rejectAnthropicContent(content any) error {
	switch v := content.(type) {
	case nil, string:
		return nil
	case []any:
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			typ, _ := bm["type"].(string)
			switch typ {
			case "", "text":
				continue
			case "image":
				return Unsupported("vision", "image content blocks are not supported by the gateway chat dialect yet")
			case "tool_use", "tool_result":
				return Unsupported("tools", fmt.Sprintf("%s content blocks are not supported by the gateway chat dialect yet", typ))
			case "thinking", "redacted_thinking":
				return Unsupported("thinking", "thinking content blocks are not supported by the gateway chat dialect yet")
			case "document":
				return Unsupported("document", "document content blocks are not supported by the gateway chat dialect yet")
			default:
				return Unsupported("content_blocks", fmt.Sprintf("content block type %q is not supported by the gateway chat dialect yet", typ))
			}
		}
		return nil
	default:
		return Unsupported("content", "message content must be a string or text content blocks")
	}
}

func isJSONNull(v any) bool {
	return v == nil
}

// textFromOpenAIContent extracts plain text from OpenAI message content (string or text parts only).
func textFromOpenAIContent(content any) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	case []any:
		var b []byte
		first := true
		for _, part := range v {
			pm, _ := part.(map[string]any)
			if pm == nil {
				continue
			}
			typ, _ := pm["type"].(string)
			if typ != "" && typ != "text" {
				return "", Unsupported("content_parts", fmt.Sprintf("content part type %q is not supported by the gateway chat dialect yet", typ))
			}
			t, _ := pm["text"].(string)
			if !first {
				b = append(b, '\n')
			}
			first = false
			b = append(b, t...)
		}
		return string(b), nil
	default:
		return "", Unsupported("content", "message content must be a string or text content parts")
	}
}

// textFromAnthropicContent extracts plain text from Anthropic message content.
func textFromAnthropicContent(content any) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	case []any:
		var out string
		for _, block := range v {
			bm, _ := block.(map[string]any)
			if bm == nil {
				continue
			}
			typ, _ := bm["type"].(string)
			switch typ {
			case "", "text":
				if t, ok := bm["text"].(string); ok {
					out += t
				}
			default:
				return "", rejectAnthropicContent(content)
			}
		}
		return out, nil
	default:
		return "", Unsupported("content", "message content must be a string or text content blocks")
	}
}
