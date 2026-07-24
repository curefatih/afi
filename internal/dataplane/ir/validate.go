package ir

// rejectOpenAIUnsupported returns an error if the raw OpenAI chat body uses
// top-level features the chat IR cannot represent. Content-level features
// (tools, images, tool messages) are validated while decoding messages.
func rejectOpenAIUnsupported(raw map[string]any) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw["functions"]; ok && !isJSONNull(v) {
		return Unsupported("functions", "functions are not supported; use tools instead")
	}
	if v, ok := raw["function_call"]; ok && !isJSONNull(v) {
		return Unsupported("function_call", "function_call is not supported; use tool_choice instead")
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
		if t, isNum := v.(float64); isNum && t > 1 {
			return Unsupported("n", "n > 1 is not supported by the gateway chat dialect yet")
		}
	}
	return nil
}

// rejectAnthropicUnsupported returns an error if the raw Anthropic messages body
// uses top-level features the chat IR cannot represent. Content-level features
// are validated while decoding messages.
func rejectAnthropicUnsupported(raw map[string]any) error {
	if raw == nil {
		return nil
	}
	if v, ok := raw["thinking"]; ok && !isJSONNull(v) {
		return Unsupported("thinking", "thinking is not supported by the gateway chat dialect yet")
	}
	return nil
}

func isJSONNull(v any) bool {
	return v == nil
}
