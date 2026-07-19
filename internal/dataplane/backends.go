package dataplane

import (
	"context"
	"io"
	"net/http"

	"github.com/curefatih/afi/internal/snapshot"
)

// OpenAITransport is the outbound OpenAI-compatible HTTP surface used by chat + audio.
type OpenAITransport interface {
	ChatCompletions(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
	AudioSpeech(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte) (*http.Response, error)
	AudioTranscriptions(ctx context.Context, provider snapshot.Provider, targetModel, contentType string, body io.Reader) (*http.Response, error)
}

// AnthropicTransport is the outbound Anthropic HTTP surface used by chat + /v1/messages.
type AnthropicTransport interface {
	PassThrough(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
	Messages(ctx context.Context, provider snapshot.Provider, targetModel string, body []byte, stream bool) (*http.Response, error)
}

// OpenAITransportProvider is implemented by ChatProvider adapters that expose OpenAI HTTP.
type OpenAITransportProvider interface {
	OpenAITransport() OpenAITransport
}

// AnthropicTransportProvider is implemented by ChatProvider adapters that expose Anthropic HTTP.
type AnthropicTransportProvider interface {
	AnthropicTransport() AnthropicTransport
}
