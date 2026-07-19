package openaichat

import (
	"encoding/json"
	"fmt"
	"io"
)

// ContentToString normalizes OpenAI message content (string or JSON) to text.
func ContentToString(content any) string {
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

// WriteSSEChunk writes one OpenAI chat.completion.chunk SSE event.
func WriteSSEChunk(w io.Writer, id, model string, delta map[string]any, finishReason any) error {
	chunk := map[string]any{
		"id":     id,
		"object": "chat.completion.chunk",
		"model":  model,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": finishReason,
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}

// WriteSSEDone writes the OpenAI stream terminator.
func WriteSSEDone(w io.Writer) error {
	_, err := io.WriteString(w, "data: [DONE]\n\n")
	return err
}

// ParseUsageTokens extracts prompt/completion tokens from an OpenAI chat JSON body.
func ParseUsageTokens(body []byte) (prompt, completion int64) {
	var parsed struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, 0
	}
	return parsed.Usage.PromptTokens, parsed.Usage.CompletionTokens
}
