package demohook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	sdkhook "github.com/curefatih/afi/sdk/hook"
)

// Name is the hook identifier exposed on gateway healthz.
const Name = "demo_tag"

// Hook prefixes the last user message with [hook:demo] (BeforeChat) and logs
// AfterChat outcomes to the process logger.
type Hook struct {
	Log *slog.Logger
}

func New() *Hook { return &Hook{Log: slog.Default()} }

func NewWithLog(log *slog.Logger) *Hook {
	if log == nil {
		log = slog.Default()
	}
	return &Hook{Log: log}
}

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

func (h *Hook) AfterChat(_ context.Context, info sdkhook.AfterChatInfo) error {
	log := h.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("demohook.after_chat",
		"model", info.Model,
		"status", info.Status,
		"latency_ms", info.LatencyMs,
		"provider_type", info.ProviderType,
		"target_model", info.TargetModel,
	)
	return nil
}

var (
	_ sdkhook.ChatHook      = (*Hook)(nil)
	_ sdkhook.AfterChatHook = (*Hook)(nil)
)
