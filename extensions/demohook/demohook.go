package demohook

import (
	"context"
	"encoding/json"
	"fmt"
)

// Name is the hook identifier exposed on gateway healthz.
const Name = "demo_tag"

// Hook prefixes the last user message with [hook:demo] so echo (and other
// providers) can prove the in-process BeforeChat chain ran.
type Hook struct{}

func New() *Hook { return &Hook{} }

func (Hook) Name() string { return Name }

func (Hook) BeforeChat(_ context.Context, body []byte) ([]byte, error) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body, fmt.Errorf("demohook: %w", err)
	}
	msgs, ok := req["messages"].([]any)
	if !ok || len(msgs) == 0 {
		return body, nil
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		m, ok := msgs[i].(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role != "user" {
			continue
		}
		content, _ := m["content"].(string)
		const prefix = "[hook:demo] "
		if len(content) >= len(prefix) && content[:len(prefix)] == prefix {
			break
		}
		m["content"] = prefix + content
		msgs[i] = m
		req["messages"] = msgs
		out, err := json.Marshal(req)
		if err != nil {
			return body, err
		}
		return out, nil
	}
	return body, nil
}
