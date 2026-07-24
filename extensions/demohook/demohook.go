package demohook

import (
	"context"
	"log/slog"

	"github.com/curefatih/afi/sdk/chatir"
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

func (Hook) BeforeChat(_ context.Context, req chatir.Request) (chatir.Request, error) {
	const prefix = "[hook:demo] "
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role != "user" {
			continue
		}
		content := req.Messages[i].Content
		if len(content) >= len(prefix) && content[:len(prefix)] == prefix {
			break
		}
		req.Messages[i].Content = prefix + content
		break
	}
	return req, nil
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
		"dialect", info.Dialect,
		"modality", info.Modality,
	)
	return nil
}

var (
	_ sdkhook.ChatHook      = (*Hook)(nil)
	_ sdkhook.AfterChatHook = (*Hook)(nil)
)
